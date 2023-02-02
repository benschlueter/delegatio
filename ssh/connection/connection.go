/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/channels"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Handler is the connection handler. It handles the global connection and the channels.
type Handler struct {
	log                   *zap.Logger
	wg                    *sync.WaitGroup
	connection            *ssh.ServerConn
	globalRequests        <-chan *ssh.Request
	channel               <-chan ssh.NewChannel
	maxKeepAliveRetries   int
	keepAliveInterval     time.Duration
	createWaitFunc        func(context.Context, *config.KubeRessourceIdentifier) error
	newDirectTCPIPHandler func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared, *payload.ForwardTCPChannelOpen) (channels.Channel, error)
	newSessionHandler     func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared) (channels.Channel, error)
	// Also needed by channel handlers
	*channels.Shared
}

// HandleGlobalConnection handles the global connection and is the entry point for this handler.
func (c *Handler) HandleGlobalConnection(ctx context.Context) {
	// if the connection is dead terminate it.
	defer func() {
		if err := c.connection.Close(); err != nil {
			c.log.Error("failed to close connection", zap.Error(err))
		}
	}()
	c.log.Info("starting ssh session")

	// keepAlive checks if the connection is still alive and if not, it will cancel the ctx.
	ctx, closeAndWaitForKeepAlive := c.keepAlive(ctx, c.connection, make(chan struct{}))
	defer closeAndWaitForKeepAlive()

	// Discard all global out-of-band Requests.
	closeAndWaitForHandleGlobalRequests := c.handleGlobalRequests(ctx, make(chan struct{}))
	defer closeAndWaitForHandleGlobalRequests()

	c.log.Info("waiting for ressources to be ready")
	// Check that all kubernetes ressources are ready and usable for future use.
	if err := c.createWaitFunc(ctx, &config.KubeRessourceIdentifier{Namespace: c.Namespace, UserIdentifier: c.AuthenticatedUserID}); err != nil {
		c.log.Error("creating/waiting for kubernetes ressources",
			zap.Error(err),
			zap.String("userID", c.AuthenticatedUserID),
			zap.String("namespace", c.connection.User()),
		)
		return
	}
	c.log.Info("ressources are ready, serving channels")
	// handle channel requests
	c.handleChannels(ctx)
	c.log.Info("closed handleGlobalConnection gracefully")
}

// handle channel will run as log as h.channel is open and the context is not cancelled.
func (c *Handler) handleChannels(ctx context.Context) {
	// Service the incoming Channel channel in go routine
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		c.log.Info("waiting for channels to shutdown gracefully")
		c.wg.Wait()
		c.log.Info("channel shutdowns done")
	}()
	for {
		select {
		case <-ctx.Done():
			c.log.Debug("context cancelled")
			return
		case newChannel, ok := <-c.channel:
			if !ok {
				c.log.Debug("global channel closed")
				return
			}
			c.wg.Add(1)
			c.log.Debug("handling new global channel request")
			go c.handleChannel(ctx, newChannel)
		}
	}
}

func (c *Handler) handleChannel(ctx context.Context, newChannel ssh.NewChannel) {
	defer c.wg.Done()
	// Currently unsupported channel types: "x11", and "forwarded-tcpip".
	switch newChannel.ChannelType() {
	case "session":
		c.handleChannelTypeSession(ctx, newChannel)
	case "direct-tcpip":
		c.handleChannelTypeDirectTCPIP(ctx, newChannel)
	default:
		c.log.Error("unknown channel type", zap.String("type", newChannel.ChannelType()))
		err := newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()))
		if err != nil {
			c.log.Error("failed to reject channel", zap.Error(err))
		}
	}
}

// handleChannelTypeSession handles the channelSession, it will block until the connection is closed by the client,
// or the ctx is cancelled.
func (c *Handler) handleChannelTypeSession(ctx context.Context, newChannel ssh.NewChannel) {
	c.log.Debug("handling new session channel request")
	channel, requests, err := newChannel.Accept()
	if err != nil {
		c.log.Error("could not accept the channel", zap.Error(err))
		return
	}

	handler, err := c.newSessionHandler(c.log, channel, requests, c.Shared)
	if err != nil {
		c.log.Error("could not create session handler", zap.Error(err))
		return
	}
	c.log.Debug("starting session handler goroutine")
	go handler.Serve(ctx)
	handler.Wait()
}

// handleChannelTypeDirectTCPIP handles the DirectTCPIP request from the client. We get a channel and should connect it to the
// Address and Port requested in the ExtraData from the channel.
// Note that the lifetime of the portForwarding is bound to the channel.
func (c *Handler) handleChannelTypeDirectTCPIP(ctx context.Context, newChannel ssh.NewChannel) {
	c.log.Debug("handling new direct-tcpip channel request")
	var tcpipData payload.ForwardTCPChannelOpen
	err := ssh.Unmarshal(newChannel.ExtraData(), &tcpipData)
	if err != nil {
		c.log.Error("could not unmarshal payload", zap.Error(err))
		err := newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("could not unmarshel extradata in channel: %s", newChannel.ChannelType()))
		if err != nil {
			c.log.Error("failed to reject channel", zap.Error(err))
		}
		return
	}
	c.log.Debug("payload unmarshal successful", zap.Any("payload", tcpipData))
	channel, requests, err := newChannel.Accept()
	if err != nil {
		c.log.Error("could not accept the channel", zap.Error(err))
		return
	}
	handler, err := c.newDirectTCPIPHandler(c.log, channel, requests, c.Shared, &tcpipData)
	if err != nil {
		c.log.Error("could not create directtcpip handler", zap.Error(err))
		return
	}
	c.log.Debug("starting directtcpip handler goroutine")
	go handler.Serve(ctx)
	handler.Wait()
}

// keepAlive sends keep alive requests to the client, if the client is not respong 4 times, deallocate all server ressources.
func (c *Handler) keepAlive(ctx context.Context, sshConn *ssh.ServerConn, done chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	c.log.Debug("starting keepAlive")
	go func() {
		t := time.NewTicker(c.keepAliveInterval)
		defer func() {
			c.log.Debug("stopping keepAlive")
			t.Stop()
			done <- struct{}{}
		}()
		retries := 0
		for {
			select {
			case <-t.C:
				if _, _, err := sshConn.SendRequest("keepalive@golang.org", true, nil); err != nil {
					c.log.Info("keepAlive did not received a response", zap.Error(err))
					retries++
				} else {
					retries = 0
				}

				if retries > c.maxKeepAliveRetries {
					c.log.Info("keepAlive failed; closing connection", zap.Int("retries", retries))
					cancel()
					return
				}
			case <-ctx.Done():
				c.log.Info("keepAlive context canceled", zap.Error(context.Cause(ctx)))
				return
			}
		}
	}()
	return ctx, func() {
		cancel()
		<-done
	}
}

func (c *Handler) handleGlobalRequests(ctx context.Context, done chan struct{}) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	c.log.Debug("starting handleGlobalRequests")
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		for {
			select {
			case req, ok := <-c.globalRequests:
				if !ok {
					c.log.Debug("handleGlobalRequests stopped by closed chan")
					return
				}
				if req.WantReply {
					if err := req.Reply(false, nil); err != nil {
						c.log.Error("failed to reply to request", zap.Error(err))
					}
				}
				c.log.Info("discared global request")

			case <-ctx.Done():
				c.log.Debug("handleGlobalRequests stopped by context", zap.Error(context.Cause(ctx)))
				return
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}
