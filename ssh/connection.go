/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

// code based on https://gist.github.com/protosam/53cf7970e17e06135f1622fa9955415f#file-basic-sshd-go
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type sshConnectionHandler struct {
	globalRequests      <-chan *ssh.Request
	channel             <-chan ssh.NewChannel
	connection          *ssh.ServerConn
	log                 *zap.Logger
	parent              *sshServer
	namespace           string
	authenticatedUserID string
	maxKeepAliveRetries int
}

// NewSSHConnectionHandler returns a sshConnection.
func NewSSHConnectionHandler(parent *sshServer, connection *ssh.ServerConn, channel <-chan ssh.NewChannel, reqs <-chan *ssh.Request) *sshConnectionHandler {
	logIdentifier := base64.StdEncoding.EncodeToString(connection.SessionID())
	return &sshConnectionHandler{
		connection:          connection,
		channel:             channel,
		globalRequests:      reqs,
		log:                 parent.log.Named(logIdentifier),
		parent:              parent,
		namespace:           connection.User(),
		authenticatedUserID: connection.Permissions.Extensions[authenticatedUserID],
		maxKeepAliveRetries: 3,
	}
}

func (s *sshConnectionHandler) HandleGlobalConnection(ctx context.Context) {
	// if the connection is dead terminate it.
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		if err := s.connection.Close(); err != nil {
			s.log.Error("failed to close connection", zap.Error(err))
		}
	}()

	go s.keepAlive(ctx, cancel, s.connection)

	s.log.Info("start handling new ssh connection")

	// Discard all global out-of-band Requests.
	// We dont care about graceful termination of this routine.
	go func() {
		for req := range s.globalRequests {
			if req.WantReply {
				if err := req.Reply(false, nil); err != nil {
					s.log.Error("failed to reply to request", zap.Error(err))
				}
			}
			s.log.Info("discared request")
		}
	}()

	// Check if the pods are ready and we can exec on them.
	// Otherwise spawn the pods.
	if err := s.parent.client.CreateAndWaitForRessources(ctx, s.connection.User(), s.authenticatedUserID); err != nil {
		s.log.Error("creating/waiting for kubernetes ressources",
			zap.Error(err),
			zap.String("userID", s.authenticatedUserID),
			zap.String("namespace", s.connection.User()),
		)
		return
	}
	// handle channel requests
	s.handleChannels(ctx, s.channel)
	s.log.Info("closing ssh session")
}

func (s *sshConnectionHandler) handleChannels(ctx context.Context, chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	handleChannelWg := &sync.WaitGroup{}
	defer handleChannelWg.Wait()
	for {
		select {
		case <-ctx.Done():
			return
		case newChannel, ok := <-chans:
			// when we close the "channel" from newChannel.Accept() in s.handleChannel the ssh.NewChannel
			// is closed from the library side as well.
			if !ok {
				s.log.Debug("channel closed")
				return
			}
			handleChannelWg.Add(1)
			s.log.Debug("handling new channel request")
			go s.handleChannel(ctx, handleChannelWg, newChannel)
		}
	}
}

func (s *sshConnectionHandler) handleChannel(ctx context.Context, wg *sync.WaitGroup, newChannel ssh.NewChannel) {
	defer wg.Done()

	// Since we're handling a shell, we expect a
	// channel type of "session". The also describes
	// "x11", "direct-tcpip" and "forwarded-tcpip"
	// channel types.
	switch newChannel.ChannelType() {
	case "session":
		s.handleChannelTypeSession(ctx, newChannel)
	case "direct-tcpip":
		s.handleChannelTypeDirectTCPIP(ctx, newChannel)
	default:
		s.log.Error("unknown channel type", zap.String("type", newChannel.ChannelType()))
		err := newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()))
		if err != nil {
			s.log.Error("failed to reject channel", zap.Error(err))
		}
	}
}

func (s *sshConnectionHandler) handleChannelTypeSession(ctx context.Context, newChannel ssh.NewChannel) {
	// At this point, we have the opportunity to reject the client's
	// request for another logical channel
	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.log.Error("could not accept the channel", zap.Error(err))
		return
	}
	defer func(log *zap.Logger) {
		err := channel.Close()
		if err != nil {
			log.Error("closing connection", zap.Error(err))
		}
		log.Debug("closed channel connection")
	}(s.log)

	channelStruct := NewSSHChannelHandler(s, channel, requests)
	channelStruct.Serve(ctx)
}

// handleChannelTypeDirectTCPIP handles the DirectTCPIP request from the client. We get a channel and should connect it to the
// Address and Port requested in the ExtraData from the channel.
// Note that the lifetime of the portForwarding is bound to the SSH connection, not the the channel itself.
func (s *sshConnectionHandler) handleChannelTypeDirectTCPIP(ctx context.Context, newChannel ssh.NewChannel) {
	var payload ForwardTCPChannelOpenPayload
	err := ssh.Unmarshal(newChannel.ExtraData(), &payload)
	if err != nil {
		s.log.Error("could not unmarshal payload", zap.Error(err))
		err := newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("could not unmarshel extradata in channel: %s", newChannel.ChannelType()))
		if err != nil {
			s.log.Error("failed to reject channel", zap.Error(err))
		}
		return
	}
	s.log.Debug("payload", zap.Any("payload", payload))

	channel, _, err := newChannel.Accept()
	if err != nil {
		s.log.Error("could not accept the channel", zap.Error(err))
		return
	}
	err = s.parent.client.CreatePodPortForward(ctx, s.namespace, fmt.Sprintf("%s-statefulset-0", s.authenticatedUserID), fmt.Sprint(payload.PortToConnect), channel)
	if err != nil {
		s.log.Error("could not create port forward", zap.Error(err))
		return
	}
	defer func(log *zap.Logger) {
		err := channel.Close()
		if err != nil {
			log.Error("closing connection", zap.Error(err))
		}
		log.Debug("closed \"DirectTCPIP\" channel")
	}(s.log)
	// stopChan <- struct{}{}
}

func (s *sshConnectionHandler) keepAlive(ctx context.Context, cancel context.CancelFunc, sshConn *ssh.ServerConn) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	s.log.Debug("starting keepAlive")
	retries := 0
	for {
		select {
		case <-t.C:
			if _, _, err := sshConn.SendRequest("keepalive@golang.org", true, nil); err != nil {
				s.log.Info("keepAlive did not received a response", zap.Error(err))
				retries++
			} else {
				retries = 0
			}

			if retries > 3 {
				s.log.Info("keepAlive failed 3 times, closing connection")
				cancel()
			}
		case <-ctx.Done():
			s.log.Debug("stopping keepAlive")
			return
		}
	}
}
