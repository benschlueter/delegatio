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
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/benschlueter/delegatio/internal/kubernetes"
	"github.com/benschlueter/delegatio/internal/store"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"github.com/benschlueter/delegatio/ssh/connection"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

const (
	authenticatedUserID = "sha256Fingerprint"
)

// TODO: Add support for multiple users

type sshServer struct {
	log                *zap.Logger
	client             *kubernetes.Client
	handleConnWG       *sync.WaitGroup
	currentConnections int64
	backingStore       store.Store
	privateKey         []byte
}

// NewSSHServer returns a sshServer.
func NewSSHServer(client *kubernetes.Client, log *zap.Logger, storage store.Store, privKey []byte) *sshServer {
	return &sshServer{
		client:             client,
		log:                log,
		handleConnWG:       &sync.WaitGroup{},
		currentConnections: 0,
		backingStore:       storage,
		privateKey:         privKey,
	}
}

func (s *sshServer) StartServer(ctx context.Context) {
	// In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed
	// in favour of an SSH connection type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in effect, upgrading the net.Conn
	// into an ssh.ServerConn
	config := &ssh.ServerConfig{
		// Function is called to determine if the user is allowed to connect with the ssh server
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			s.log.Info("publickeycallback called", zap.String("user", conn.User()), zap.Binary("session", conn.SessionID()))
			if ok, err := s.data().ChallengeExists(conn.User()); err != nil || !ok {
				return nil, fmt.Errorf("user %s not in database or internal store error %w", conn.User(), err)
			}
			encodeKey := base64.StdEncoding.EncodeToString(key.Marshal())
			compareKey := fmt.Sprintf("%s %s", key.Type(), encodeKey)
			if ok, err := s.data().PublicKeyExists(compareKey); err != nil || !ok {
				return nil, fmt.Errorf("pubkey %v not in database or internal store error %w", compareKey, err)
			}
			return &ssh.Permissions{
				Extensions: map[string]string{
					"authType":          "pk",
					authenticatedUserID: strings.ToLower(ssh.FingerprintSHA256(key)[7:47]),
				},
			}, nil
		},
	}
	// routine currently leaks
	go s.periodicLogs(ctx)

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
	s.log.Info("waiting for all connections to terminate gracefully")
	s.handleConnWG.Wait()
	s.log.Info("closing program")
}

func (s *sshServer) validateAndProcessConnection(ctx context.Context, tcpConn net.Conn, config *ssh.ServerConfig) {
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
	builder.SetExecFunc(s.client.ExecuteCommandInPod)
	builder.SetForwardFunc(s.client.CreatePodPortForward)
	builder.SetRessourceFunc(s.client.CreateAndWaitForRessources)
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

func (s *sshServer) periodicLogs(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
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

func (s *sshServer) data() storewrapper.StoreWrapper {
	return storewrapper.StoreWrapper{Store: s.backingStore}
}

func registerSignalHandler(cancelContext context.CancelFunc, done chan<- struct{}, log *zap.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Info("cancellation signal received")
	cancelContext()
	signal.Stop(sigs)
	close(sigs)
	done <- struct{}{}
}
