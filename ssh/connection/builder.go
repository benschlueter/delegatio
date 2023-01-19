/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

// code based on https://gist.github.com/protosam/53cf7970e17e06135f1622fa9955415f#file-basic-sshd-go
package connection

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type sshConnectionBuilder struct {
	globalRequests <-chan *ssh.Request
	channel        <-chan ssh.NewChannel
	connection     *ssh.ServerConn
	log            *zap.Logger
	forwardFunc    func(context.Context, *config.KubeForwardConfig) error
	execFunc       func(context.Context, *config.KubeExecConfig) error
	createWaitFunc func(context.Context, *config.KubeRessourceIdentifier) error
}

// NewSSHConnectionHandlerBuilder returns a sshConnection.
func NewSSHConnectionHandlerBuilder(logger *zap.Logger, connection *ssh.ServerConn, channel <-chan ssh.NewChannel, reqs <-chan *ssh.Request) *sshConnectionBuilder {
	return &sshConnectionBuilder{}
}

// SetExecFunc sets the exec function.
func (s *sshConnectionBuilder) SetExecFunc(execFunc func(context.Context, *config.KubeExecConfig) error) {
	s.execFunc = execFunc
}

// SetForwardFunc sets the forward function.
func (s *sshConnectionBuilder) SetForwardFunc(forwardFunc func(context.Context, *config.KubeForwardConfig) error) {
	s.forwardFunc = forwardFunc
}

// SetRessourceFunc sets the ressource function.
func (s *sshConnectionBuilder) SetRessourceFunc(createWaitFunc func(context.Context, *config.KubeRessourceIdentifier) error) {
	s.createWaitFunc = createWaitFunc
}

// SetConnection sets the connection.
func (s *sshConnectionBuilder) SetConnection(connection *ssh.ServerConn) {
	s.connection = connection
}

// SetChannel sets the channel.
func (s *sshConnectionBuilder) SetChannel(channel <-chan ssh.NewChannel) {
	s.channel = channel
}

// SetGlobalRequests sets the global requests.
func (s *sshConnectionBuilder) SetGlobalRequests(reqs <-chan *ssh.Request) {
	s.globalRequests = reqs
}

// SetLogger sets the logger.
func (s *sshConnectionBuilder) SetLogger(log *zap.Logger) {
	s.log = log
}

// Build builds the sshConnection. All fields must be set otherwise an error is returned.
func (s *sshConnectionBuilder) Build() (*sshConnectionHandler, error) {
	userID, ok := s.connection.Permissions.Extensions[config.AuthenticatedUserID]
	if !ok {
		return nil, errors.New("no authenticated user id found")
	}
	logIdentifier := base64.StdEncoding.EncodeToString(s.connection.SessionID())
	if s.connection == nil || s.channel == nil || s.globalRequests == nil {
		return nil, errors.New("connection, channel or globalRequests is nil")
	}
	if s.execFunc == nil || s.forwardFunc == nil || s.createWaitFunc == nil {
		return nil, errors.New("execFunc, forwardFunc or createWaitFunc is nil")
	}
	if s.log == nil {
		return nil, errors.New("no logger provided")
	}

	return &sshConnectionHandler{
		connection:          s.connection,
		channel:             s.channel,
		globalRequests:      s.globalRequests,
		log:                 s.log.Named(logIdentifier),
		forwardFunc:         s.forwardFunc,
		execFunc:            s.execFunc,
		createWaitFunc:      s.createWaitFunc,
		namespace:           s.connection.User(),
		authenticatedUserID: userID,
		maxKeepAliveRetries: 3,
	}, nil
}
