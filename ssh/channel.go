/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// SSHChannelHandler is a wrapper around an ssh.Channel and ssh.Requests.
type SSHChannelHandler struct {
	channel  ssh.Channel
	requests <-chan *ssh.Request
	ptyReq   *PtyRequestPayload
	wg       *sync.WaitGroup
	log      *zap.Logger
	window   *Winsize
	parent   *sshConnectionHandler
}

// NewSSHChannelHandler returns a new SSHChannelServer.
func NewSSHChannelHandler(parent *sshConnectionHandler, channel ssh.Channel, requests <-chan *ssh.Request) *SSHChannelHandler {
	return &SSHChannelHandler{
		log:      parent.log.Named("channelHandler"),
		channel:  channel,
		requests: requests,
		wg:       &sync.WaitGroup{},
		window:   &Winsize{Queue: make(chan *remotecommand.TerminalSize)},
		parent:   parent,
	}
}

// Serve starts the server. It will block until the context is canceled.
func (s *SSHChannelHandler) Serve(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		s.wg.Wait()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-s.requests:
			if !ok {
				s.log.Debug("request channel closed")
				return
			}
			s.log.Debug("received data over request channel", zap.Any("req", req))
			switch req.Type {
			case "shell":
				s.log.Info("shell request", zap.Any("data", req.Payload))
				s.wg.Add(1)
				go s.handleShell(ctx)
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to reply to \"shell\" request", zap.Error(err))
				}
			case "pty-req":
				ptyReq := PtyRequestPayload{}
				if err := ssh.Unmarshal(req.Payload, &ptyReq); err != nil {
					s.log.Error("failled to unmarshal pty request", zap.Error(err))
					continue
				}
				s.log.Info("pty request", zap.Any("data", ptyReq))
				s.ptyReq = &ptyReq
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to respond to \"pty-req\" request", zap.Error(err))
				}
			case "window-change":
				windowChange := WindowChangeRequestPayload{}
				if err := ssh.Unmarshal(req.Payload, &windowChange); err != nil {
					s.log.Error("failled to unmarshal window-change request", zap.Error(err))
					continue
				}
				s.log.Info("window-change request", zap.Any("data", windowChange))
				s.window.Queue <- &remotecommand.TerminalSize{Width: uint16(windowChange.WidthColumns), Height: uint16(windowChange.HeightRows)}
			case "subsystem":
				subSys := SubsystemRequestPayload{}
				if err := ssh.Unmarshal(req.Payload, &subSys); err != nil {
					s.log.Error("failled to unmarshal window-change request", zap.Error(err))
					continue
				}
				s.log.Info("subsystem request", zap.Any("data", subSys))
				s.wg.Add(1)
				go s.handleSubsystem(ctx, subSys.Subsystem)
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to respond to \"subsystem\" request", zap.Error(err))
				}
			default:
				if req.WantReply {
					if err := req.Reply(false, nil); err != nil {
						s.log.Error("failled to respond to request", zap.Any("request", req), zap.Error(err))
					}
				}
				s.log.Info("unimplemented request", zap.Any("request", req))
			}
		}
	}
}

// Close closes all remaining resources associated with the server.
func (s *SSHChannelHandler) Close() {
	close(s.window.Queue)
}

func (s *SSHChannelHandler) handleShell(ctx context.Context) {
	defer s.wg.Done()
	defer func() {
		if err := s.channel.Close(); err != nil {
			s.log.Error("failed to close channel", zap.Error(err))
		}
	}()
	// Fire up "kubectl exec" for this session
	tty := false
	if s.ptyReq != nil {
		// Be safe and feed the queue in a goroutine. If somehow another window-change request is pending the connecton
		// will deadlock.
		go func() {
			s.window.Queue <- &remotecommand.TerminalSize{Width: uint16(s.ptyReq.WidthColumns), Height: uint16(s.ptyReq.HeightRows)}
		}()
		tty = true
	}

	err := s.parent.parent.client.CreatePodShell(ctx, s.parent.namespace, fmt.Sprintf("%s-statefulset-0", s.parent.authenticatedUserID), s.channel, s.window, tty)
	if err != nil {
		s.log.Error("createPodShell exited", zap.Error(err))
		_, _ = s.channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v", err)))
		return
	}
	_, _ = s.channel.Write([]byte("graceful termination"))
}

// handleSubsystem handles the "subsystem" request. This is used for SFTP.
// This is used by "scp" to copy files from the localhost to the pod or vice versa.
func (s *SSHChannelHandler) handleSubsystem(ctx context.Context, cmd string) {
	defer s.wg.Done()
	subSysMap := map[string]string{
		"sftp": "/usr/lib/ssh/sftp-server",
	}
	parsedSubsystem, ok := subSysMap[cmd]
	if !ok {
		s.log.Error("unknown subsystem", zap.String("subsystem", cmd))
		_, _ = s.channel.Write([]byte(fmt.Sprintf("unknown subsystem: %s", cmd)))
		return
	}

	tty := false
	if s.ptyReq != nil {
		// Be safe and feed the queue in a goroutine. If somehow another window-change request is pending the connecton
		// will deadlock.
		go func() {
			s.window.Queue <- &remotecommand.TerminalSize{Width: uint16(s.ptyReq.WidthColumns), Height: uint16(s.ptyReq.HeightRows)}
		}()
		tty = true
	}

	err := s.parent.parent.client.ExecuteCommandInPod(ctx, s.parent.namespace, fmt.Sprintf("%s-statefulset-0", s.parent.authenticatedUserID), parsedSubsystem, s.channel, s.window, tty)
	if err != nil {
		s.log.Error("ExecuteCommandInPod exited", zap.Error(err))
		_, _ = s.channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v", err)))
		return
	}
}

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
	Queue chan *remotecommand.TerminalSize
}

// Next returns the size. The chanel must be served. Otherwise the connection will hang.
func (w *Winsize) Next() *remotecommand.TerminalSize {
	return <-w.Queue
}
