/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

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
	"github.com/benschlueter/delegatio/ssh/kubernetes"
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
	k8sHelper      kubernetes.K8sAPI
}

// NewBuilder returns a sshConnection.
func NewBuilder() *Builder {
	return &Builder{}
}

// SetK8sHelper sets the k8shelper interface.
func (s *Builder) SetK8sHelper(helper kubernetes.K8sAPI) {
	s.k8sHelper = helper
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
	if s.k8sHelper == nil {
		return nil, errors.New("no k8s helper provided is nil")
	}
	if s.log == nil {
		return nil, errors.New("no logger provided")
	}

	userK8SAPI := kubernetes.NewK8sAPIUserWrapper(s.k8sHelper, &config.KubeRessourceIdentifier{
		// Namespace will define the challenge / container we're using
		Namespace:      config.UserNamespace,
		UserIdentifier: userID,
		// Currently unused, but required for later.
		ContainerIdentifier: s.connection.User(),
	})

	return &Handler{
		wg:                  &sync.WaitGroup{},
		maxKeepAliveRetries: 3,
		keepAliveInterval:   10 * time.Second,
		connection:          s.connection,
		channel:             s.channel,
		globalRequests:      s.globalRequests,
		log:                 s.log.Named("connection").Named(logIdentifier),
		K8sAPIUser:          userK8SAPI,

		newSessionHandler:     newSession,
		newDirectTCPIPHandler: newDirectTCPIP,
		writeFileToContainer:  writeFileToContainer,
	}, nil
}

func writeFileToContainer(ctx context.Context, conn *ssh.ServerConn, api kubernetes.K8sAPIUser) error {
	if conn.Permissions.Extensions[config.AuthenticationType] != "pw" {
		return nil
	}
	return api.WriteFileInPod(ctx, &config.KubeFileWriteConfig{
		Namespace:      config.UserNamespace,
		UserIdentifier: conn.Permissions.Extensions[config.AuthenticatedUserID],
		FileName:       "delegatio_priv_key",
		FileData:       []byte(conn.Permissions.Extensions[config.AuthenticatedPrivKey]),
		FilePath:       "/root/.ssh",
	})
}

func newSession(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, api kubernetes.K8sAPIUser) (channels.Channel, error) {
	builder := channels.SessionBuilderSkeleton()
	builder.SetRequests(requests)
	builder.SetChannel(channel)
	builder.SetLog(log)
	builder.SetK8sUserAPI(api)
	return builder.Build()
}

func newDirectTCPIP(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, api kubernetes.K8sAPIUser, data *payload.ForwardTCPChannelOpen) (channels.Channel, error) {
	builder := channels.DirectTCPIPBuilderSkeleton()
	builder.SetRequests(requests)
	builder.SetChannel(channel)
	builder.SetLog(log)
	builder.SetK8sUserAPI(api)
	builder.SetDirectTCPIPData(data)
	return builder.Build()
}
