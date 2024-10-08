/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/store"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"github.com/benschlueter/delegatio/ssh/connection"
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"github.com/benschlueter/delegatio/ssh/ldap"
	"github.com/benschlueter/delegatio/ssh/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// TODO: Add support for multiple users

// Server is a ssh server.
type Server struct {
	log                *zap.Logger
	k8sHelper          kubernetes.K8sAPI
	handleConnWG       *sync.WaitGroup
	currentConnections int64
	backingStore       store.Store
	ldap               *ldap.Ldap
	privateKey         []byte
}

// NewServer returns a sshServer.
func NewServer(client kubernetes.K8sAPI, log *zap.Logger, storage store.Store, privKey []byte, ldap *ldap.Ldap) *Server {
	return &Server{
		k8sHelper:          client,
		log:                log,
		handleConnWG:       &sync.WaitGroup{},
		currentConnections: 0,
		backingStore:       storage,
		privateKey:         privKey,
		ldap:               ldap,
	}
}

// Start starts the ssh server.
func (s *Server) Start(ctx context.Context) {
	config := &ssh.ServerConfig{
		// Function is called to determine if the user is allowed to connect with the ssh server
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			var userData config.UserInformation
			encodeKey := base64.StdEncoding.EncodeToString(key.Marshal())
			s.log.Debug("publickeycallback called", zap.String("user", conn.User()), zap.Binary("session", conn.SessionID()), zap.String("key", encodeKey))

			err := s.data().GetPublicKeyData(string(ssh.MarshalAuthorizedKey(key)), &userData)
			if err != nil {
				s.log.Error("failed to obtain user data", zap.Error(err))
				return nil, fmt.Errorf("failed to obtain user data: %w", err)
			}
			return &ssh.Permissions{
				Extensions: map[string]string{
					config.AuthenticationType:  "pk",
					config.AuthenticatedUserID: userData.UUID,
				},
			}, nil
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			s.log.Debug("passwordcallback called", zap.String("user", conn.User()), zap.Binary("session", conn.SessionID()))
			userData, err := s.ldap.Search(conn.User(), string(password))
			if err != nil {
				return nil, fmt.Errorf("ldap search for user %s failed: %w", conn.User(), err)
			}
			exists, err := s.data().UUIDExists(userData.UUID)
			if err != nil {
				s.log.Error("error checking if uuid exists; likely due to etcd", zap.Error(err))
				return nil, fmt.Errorf("error checking if uuid %s exists: %w", userData.UUID, err)
			}
			// We assume that when an entry exists in the store it ALWAYS has a public/private key pair
			if !exists {
				privKey, pubKey, err := util.CreateSSHKeypair()
				if err != nil {
					return nil, fmt.Errorf("failed to create ssh keypair: %w", err)
				}
				userData.PrivKey = privKey
				userData.PubKey = pubKey
				err = s.data().PutDataIdxByUUID(userData.UUID, userData)
				if err != nil {
					return nil, fmt.Errorf("failed to put data into store: %w", err)
				}
				err = s.data().PutDataIdxByPubKey(string(userData.PubKey), userData)
				if err != nil {
					return nil, fmt.Errorf("failed to put data into store: %w", err)
				}
				s.log.Debug("public key created and stored", zap.String("key", string(userData.PubKey)))
			}
			s.log.Debug("private key", zap.String("key", string(userData.PrivKey)))
			return &ssh.Permissions{
				Extensions: map[string]string{
					config.AuthenticationType:   "pw",
					config.AuthenticatedPrivKey: string(userData.PrivKey),
					config.AuthenticatedUserID:  userData.UUID,
				},
			}, nil
		},
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return fmt.Sprintf("delegatio ssh server version %s\ncommit %s\nsession ID %s\n", config.Version, config.Commit, base64.StdEncoding.EncodeToString(conn.SessionID()))
		},
	}
	// routine currently leaks
	periodicLogsDone := make(chan struct{})
	go s.periodicLogs(ctx, periodicLogsDone)

	private, err := ssh.ParsePrivateKey(s.privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config.AddHostKey(private)
	listener, err := net.Listen("tcp", "0.0.0.0:2200")
	if err != nil {
		log.Fatalf("Failed to listen on 2200 (%s)", err)
	}
	defer listener.Close()

	s.log.Info("Listening on  \"0.0.0.0:2200\"")
	go func(ctx context.Context) {
		for {
			tcpConn, err := listener.Accept()
			if errors.Is(err, net.ErrClosed) {
				// s.log.Error("failed to accept incoming connection", zap.Error(err))
				return
			}
			if err != nil {
				s.log.Error("failed to accept incoming connection", zap.Error(err))
				continue
			}
			s.log.Info("received data on connection", zap.String("addr", tcpConn.RemoteAddr().String()))
			s.handleConnWG.Add(1)
			atomic.AddInt64(&s.currentConnections, 1)
			go s.validateAndProcessConnection(ctx, tcpConn, config)
		}
	}(ctx)
	<-ctx.Done()
	if err := listener.Close(); err != nil {
		s.log.Error("failed to close listener", zap.Error(err))
	}
	s.log.Info("waiting for periodicLogs to stop")
	<-periodicLogsDone
	s.log.Info("waiting for all connections to terminate gracefully")
	s.handleConnWG.Wait()
	s.log.Info("closing program")
}

func (s *Server) validateAndProcessConnection(ctx context.Context, tcpConn net.Conn, config *ssh.ServerConfig) {
	defer func() {
		s.handleConnWG.Done()
		atomic.AddInt64(&s.currentConnections, -1)
	}()

	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		s.log.Info("failed to handshake", zap.Error(err))
		return
	}
	if sshConn.Permissions == nil || sshConn.Permissions.Extensions == nil {
		s.log.Error("no permissions found in ssh connection")
		return
	}
	s.log.Info("authentication of connection successful", zap.Binary("session", sshConn.SessionID()))
	builder := connection.NewBuilder()
	builder.SetK8sHelper(s.k8sHelper)
	builder.SetChannel(chans)
	builder.SetGlobalRequests(reqs)
	builder.SetConnection(sshConn)
	builder.SetLogger(s.log)
	sshConnHandler, err := builder.Build()
	if err != nil {
		s.log.Info("failed to build sshConnHandler", zap.Error(err))
		return
	}
	sshConnHandler.HandleGlobalConnection(ctx)
}

func (s *Server) periodicLogs(ctx context.Context, done chan<- struct{}) {
	t := time.NewTicker(10 * time.Second)
	defer func() {
		t.Stop()
		done <- struct{}{}
	}()
	s.log.Debug("starting periodicLogs")
	for {
		select {
		case <-t.C:
			s.log.Info("active connections", zap.Int64("conn", s.currentConnections))
			s.log.Info("active goroutines", zap.Int("goroutines", runtime.NumGoroutine()))
		case <-ctx.Done():
			s.log.Debug("stopping periodicLogs")
			return
		}
	}
}

func (s *Server) data() storewrapper.StoreWrapper {
	return storewrapper.StoreWrapper{Store: s.backingStore}
}
