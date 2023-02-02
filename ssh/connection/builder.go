/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

// code based on https://gist.github.com/protosam/53cf7970e17e06135f1622fa9955415f#file-basic-sshd-go
package connection

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/channels"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Builder is a builder for ssh connections.
// After the server handshake is complete, the server builds a connection handler with this builder.
type Builder struct {
	globalRequests <-chan *ssh.Request
	channel        <-chan ssh.NewChannel
	connection     *ssh.ServerConn
	log            *zap.Logger
	forwardFunc    func(context.Context, *config.KubeForwardConfig) error
	execFunc       func(context.Context, *config.KubeExecConfig) error
	createWaitFunc func(context.Context, *config.KubeRessourceIdentifier) error
}

// NewBuilder returns a sshConnection.
func NewBuilder() *Builder {
	return &Builder{}
}

// SetExecFunc sets the exec function.
func (s *Builder) SetExecFunc(execFunc func(context.Context, *config.KubeExecConfig) error) {
	s.execFunc = execFunc
}

// SetForwardFunc sets the forward function.
func (s *Builder) SetForwardFunc(forwardFunc func(context.Context, *config.KubeForwardConfig) error) {
	s.forwardFunc = forwardFunc
}

// SetRessourceFunc sets the ressource function.
func (s *Builder) SetRessourceFunc(createWaitFunc func(context.Context, *config.KubeRessourceIdentifier) error) {
	s.createWaitFunc = createWaitFunc
}

// SetConnection sets the connection.
func (s *Builder) SetConnection(connection *ssh.ServerConn) {
	s.connection = connection
}

// SetChannel sets the channel.
func (s *Builder) SetChannel(channel <-chan ssh.NewChannel) {
	s.channel = channel
}

// SetGlobalRequests sets the global requests.
func (s *Builder) SetGlobalRequests(reqs <-chan *ssh.Request) {
	s.globalRequests = reqs
}

// SetLogger sets the logger.
func (s *Builder) SetLogger(log *zap.Logger) {
	s.log = log
}

// Build builds the sshConnection. All fields must be set otherwise an error is returned.
func (s *Builder) Build() (*Handler, error) {
	if s.connection == nil || s.channel == nil || s.globalRequests == nil {
		return nil, errors.New("connection, channel or globalRequests is nil")
	}
	if s.connection.Permissions == nil || s.connection.Permissions.Extensions == nil {
		return nil, errors.New("connection malformed, permissions or extensions is nil")
	}
	userID, ok := s.connection.Permissions.Extensions[config.AuthenticatedUserID]
	if !ok {
		return nil, errors.New("no authenticated user id found")
	}
	logIdentifier := base64.StdEncoding.EncodeToString(s.connection.SessionID())
	if s.execFunc == nil || s.forwardFunc == nil || s.createWaitFunc == nil {
		return nil, errors.New("execFunc, forwardFunc or createWaitFunc is nil")
	}
	if s.log == nil {
		return nil, errors.New("no logger provided")
	}

	return &Handler{
		wg:                  &sync.WaitGroup{},
		maxKeepAliveRetries: 3,
		keepAliveInterval:   10 * time.Second,
		connection:          s.connection,
		channel:             s.channel,
		globalRequests:      s.globalRequests,
		createWaitFunc:      s.createWaitFunc,
		log:                 s.log.Named("connection").Named(logIdentifier),
		Shared: &channels.Shared{
			ForwardFunc:         s.forwardFunc,
			ExecFunc:            s.execFunc,
			Namespace:           s.connection.User(),
			AuthenticatedUserID: userID,
		},

		newSessionHandler:     newSession,
		newDirectTCPIPHandler: newDirectTCPIP,
	}, nil
}

func newSession(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, shared *channels.Shared) (channels.Channel, error) {
	builder := channels.SessionBuilderSkeleton()
	builder.SetRequests(requests)
	builder.SetChannel(channel)
	builder.SetLog(log)
	builder.SetSharedData(shared)
	return builder.Build()
}

func newDirectTCPIP(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, shared *channels.Shared, data *payload.ForwardTCPChannelOpen) (channels.Channel, error) {
	builder := channels.DirectTCPIPBuilderSkeleton()
	builder.SetRequests(requests)
	builder.SetChannel(channel)
	builder.SetLog(log)
	builder.SetSharedData(shared)
	builder.SetDirectTCPIPData(data)
	return builder.Build()
}
