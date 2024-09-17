/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package connection

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/channels"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/ssh"
)

func TestHandleChannel(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test errr")
	testCases := map[string]struct {
		channel                ssh.NewChannel
		expectFinish           bool
		sessionHandlerFunc     func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, kubernetes.K8sAPIUser) (channels.Channel, error)
		directtcpIPHandlerFunc func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, kubernetes.K8sAPIUser, *payload.ForwardTCPChannelOpen) (channels.Channel, error)
		logMessages            []string
		nonLogMessages         []string
	}{
		"session accept error": {
			channel: &stubNewChannel{
				channelType: "session",
				acceptErr:   testErr,
			},
			expectFinish: true,
			logMessages: []string{
				"could not accept the channel",
			},
		},
		"new session handler error": {
			channel: &stubNewChannel{
				channelType: "session",
			},
			expectFinish: true,
			sessionHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, kubernetes.K8sAPIUser) (channels.Channel, error) {
				return nil, testErr
			},
			logMessages: []string{
				"could not create session handler",
			},
		},
		"session cancelled by context": {
			channel: &stubNewChannel{
				channelType: "session",
			},
			expectFinish: false,
			sessionHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, kubernetes.K8sAPIUser) (channels.Channel, error) {
				return &stubHandler{done: make(chan struct{})}, nil
			},
			nonLogMessages: []string{
				"could not create session handler",
				"could not accept the channel",
			},
			logMessages: []string{
				"starting session handler goroutine",
			},
		},
		"unknows channel type": {
			channel: &stubNewChannel{
				channelType: "unknown stuff",
			},
			expectFinish: true,
			logMessages: []string{
				"unknown channel type",
			},
		},
		"unknows channel type and reject error": {
			channel: &stubNewChannel{
				channelType: "unknown stuff",
				rejectErr:   testErr,
			},
			expectFinish: true,
			logMessages: []string{
				"unknown channel type",
				"failed to reject channel",
			},
		},
		"direct-tcpip unmarshal error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        []byte("invalid data"),
			},
			expectFinish: true,
			logMessages: []string{
				"could not unmarshal payload",
			},
		},
		"direct-tcpip unmarshal and reject error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        []byte("invalid data"),
				rejectErr:   testErr,
			},
			expectFinish: true,
			logMessages: []string{
				"could not unmarshal payload",
				"failed to reject channel",
			},
		},
		"direct-tcpip accept error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				acceptErr:   testErr,
				data:        ssh.Marshal(payload.ForwardTCPChannelOpen{}),
			},
			expectFinish: true,
			logMessages: []string{
				"could not accept the channel",
			},
			nonLogMessages: []string{
				"could not unmarshal payload",
				"failed to reject channel",
			},
		},
		"new direct-tcpip handler error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        ssh.Marshal(payload.ForwardTCPChannelOpen{}),
			},
			directtcpIPHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, kubernetes.K8sAPIUser, *payload.ForwardTCPChannelOpen) (channels.Channel, error) {
				return nil, testErr
			},
			expectFinish: true,
			logMessages: []string{
				"could not create directtcpip handler",
			},
			nonLogMessages: []string{
				"could not unmarshal payload",
				"failed to reject channel",
				"could not accept the channel",
			},
		},
		"direct-tcpip closed by ctx": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        ssh.Marshal(payload.ForwardTCPChannelOpen{}),
			},
			directtcpIPHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, kubernetes.K8sAPIUser, *payload.ForwardTCPChannelOpen) (channels.Channel, error) {
				return &stubHandler{done: make(chan struct{})}, nil
			},
			expectFinish: false,
			nonLogMessages: []string{
				"could not unmarshal payload",
				"failed to reject channel",
				"could not accept the channel",
				"could not create directtcpip handler",
			},
			logMessages: []string{
				"starting directtcpip handler goroutine",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			observedLogger := zap.New(observedZapCore)

			handler := Handler{
				log:                   observedLogger,
				newSessionHandler:     tc.sessionHandlerFunc,
				newDirectTCPIPHandler: tc.directtcpIPHandlerFunc,
				wg:                    &sync.WaitGroup{},
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			handler.wg.Add(1)
			go handler.handleChannel(ctx, tc.channel)
			if !tc.expectFinish {
				cancel()
			}
			handler.wg.Wait()
			for _, v := range tc.logMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.GreaterOrEqual(len(logs), 1)
			}
			for _, v := range tc.nonLogMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.Zero(len(logs))
			}
		})
	}
}

func TestHandleChannels(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		closeChannel    bool
		cancelCtx       bool
		channelElements []ssh.NewChannel
		logMessages     []string
		nonLogMessages  []string
	}{
		"closed channel": {
			closeChannel: true,
			logMessages: []string{
				"global channel closed",
			},
		},
		"closed channel after starting at least one go routine": {
			closeChannel: true,
			channelElements: []ssh.NewChannel{
				&stubNewChannel{
					channelType: "non-existing",
				},
				&stubNewChannel{
					channelType: "non-existing",
				},
			},
			logMessages: []string{
				"handling new global channel request",
			},
		},
		"cancel ctx": {
			cancelCtx: true,
			logMessages: []string{
				"context cancelled",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			observedLogger := zap.New(observedZapCore)

			channelChan := make(chan ssh.NewChannel)
			handler := Handler{
				log:     observedLogger,
				wg:      &sync.WaitGroup{},
				channel: channelChan,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan struct{})

			go func() {
				for _, v := range tc.channelElements {
					channelChan <- v
				}

				if tc.closeChannel {
					close(channelChan)
				}
				if tc.cancelCtx {
					cancel()
				}
				done <- struct{}{}
			}()

			handler.handleChannels(ctx)
			<-done
			for _, v := range tc.logMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.GreaterOrEqual(len(logs), 1)
			}
			for _, v := range tc.nonLogMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.Zero(len(logs))
			}
		})
	}
}

func TestHandleGlobalConnection(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test errr")
	testCases := map[string]struct {
		createWfuncErr error
		sshConnection  *ssh.ServerConn
		logMessages    []string
		nonLogMessages []string
	}{
		"works": {
			createWfuncErr: nil,
			sshConnection: &ssh.ServerConn{
				Conn: &stubConn{},
			},
			logMessages: []string{
				"closed handleGlobalConnection gracefully",
			},
		},
		"createWaitFunc error": {
			createWfuncErr: testErr,
			sshConnection: &ssh.ServerConn{
				Conn: &stubConn{},
			},
			logMessages: []string{
				"creating/waiting for kubernetes ressources",
			},
		},
		"close error": {
			createWfuncErr: nil,
			sshConnection: &ssh.ServerConn{
				Conn: &stubConn{
					closeErr: testErr,
				},
			},
			logMessages: []string{
				"failed to close connection",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			observedLogger := zap.New(observedZapCore)

			channelChan := make(chan ssh.NewChannel)
			handler := Handler{
				log:               observedLogger,
				wg:                &sync.WaitGroup{},
				keepAliveInterval: time.Second,
				channel:           channelChan,
				K8sAPIUser: &kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						CreateAndWaitForRessourcesErr: tc.createWfuncErr,
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "test-ns",
						UserIdentifier: "test-user",
					},
				},
				connection: tc.sshConnection,
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			handler.HandleGlobalConnection(ctx)
			for _, v := range tc.logMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.GreaterOrEqual(len(logs), 1)
			}
			for _, v := range tc.nonLogMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.Zero(len(logs))
			}
		})
	}
}

func TestKeepAlive(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test errr")
	testCases := map[string]struct {
		closeByCtx     bool
		interval       time.Duration
		serverConn     *ssh.ServerConn
		logMessages    []string
		nonLogMessages []string
	}{
		"close by context ": {
			closeByCtx: true,
			serverConn: &ssh.ServerConn{
				Conn: &stubConn{},
			},
			interval: time.Second,
			logMessages: []string{
				"stopping keepAlive",
				"starting keepAlive",
				"keepAlive context canceled",
			},
			nonLogMessages: []string{
				"keepAlive failed; closing connection",
				"keepAlive did not received a response",
			},
		},
		"close by timeout ": {
			serverConn: &ssh.ServerConn{
				Conn: &stubConn{
					sendRequestErr: testErr,
				},
			},
			interval: time.Nanosecond,
			logMessages: []string{
				"keepAlive failed; closing connection",
				"stopping keepAlive",
				"starting keepAlive",
				"keepAlive did not received a response",
			},
			nonLogMessages: []string{
				"keepAlive context canceled",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			observedLogger := zap.New(observedZapCore)
			handler := Handler{keepAliveInterval: tc.interval, log: observedLogger}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan struct{})
			_, cancelHandler := handler.keepAlive(ctx, tc.serverConn, done)

			if tc.closeByCtx {
				cancelHandler()
			} else {
				<-done
			}
			for _, v := range tc.logMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.GreaterOrEqual(len(logs), 1)
			}
			for _, v := range tc.nonLogMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.Zero(len(logs))
			}
		})
	}
}

func TestGlobalRequests(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		closeByCtx     bool
		logMessages    []string
		nonLogMessages []string
	}{
		"close by context ": {
			closeByCtx: true,
			logMessages: []string{
				"handleGlobalRequests stopped by context",
			},
		},
		"close by channelClose ": {
			logMessages: []string{
				"handleGlobalRequests stopped by closed chan",
				"discared global request",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			observedLogger := zap.New(observedZapCore)
			requests := make(chan *ssh.Request, 1)
			requests <- &ssh.Request{
				WantReply: false,
			}
			handler := Handler{log: observedLogger, globalRequests: requests}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan struct{})
			_ = handler.handleGlobalRequests(ctx, done)

			if tc.closeByCtx {
				cancel()
			} else {
				close(requests)
			}
			<-done
			for _, v := range tc.logMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.GreaterOrEqual(len(logs), 1)
			}
			for _, v := range tc.nonLogMessages {
				logs := observedLogs.FilterMessage(v).TakeAll()
				assert.Zero(len(logs))
			}
		})
	}
}

type stubConn struct {
	sendRequestErr error
	openChannelErr error
	closeErr       error
	waitErr        error
}

func (c *stubConn) SendRequest(_ string, _ bool, _ []byte) (bool, []byte, error) {
	return true, nil, c.sendRequestErr
}

func (c *stubConn) OpenChannel(_ string, _ []byte) (ssh.Channel, <-chan *ssh.Request, error) {
	return nil, nil, c.openChannelErr
}

func (c *stubConn) Close() error {
	return c.closeErr
}

func (c *stubConn) Wait() error {
	return c.waitErr
}

func (c *stubConn) User() string {
	return "test"
}

func (c *stubConn) SessionID() []byte {
	return []byte("test")
}

func (c *stubConn) ClientVersion() []byte {
	return []byte("test")
}

func (c *stubConn) ServerVersion() []byte {
	return []byte("test")
}

func (c *stubConn) RemoteAddr() net.Addr {
	return nil
}

func (c *stubConn) LocalAddr() net.Addr {
	return nil
}

type stubNewChannel struct {
	acceptErr   error
	rejectErr   error
	channelType string
	data        []byte
}

func (s *stubNewChannel) Accept() (ssh.Channel, <-chan *ssh.Request, error) {
	return nil, nil, s.acceptErr
}

func (s *stubNewChannel) Reject(_ ssh.RejectionReason, _ string) error {
	return s.rejectErr
}

func (s *stubNewChannel) ChannelType() string {
	return s.channelType
}

func (s *stubNewChannel) ExtraData() []byte {
	return s.data
}

type stubHandler struct {
	done chan struct{}
}

func (s *stubHandler) Serve(ctx context.Context) {
	<-ctx.Done()
	s.done <- struct{}{}
}

func (s *stubHandler) Wait() {
	<-s.done
}

type stubK8sAPIWrapper struct {
	CreateAndWaitForRessourcesErr error
	ExecuteCommandInPodErr        error
	CreatePodPortForwardErr       error
	WriteFileInPodErr             error
}

func (k *stubK8sAPIWrapper) CreateAndWaitForRessources(_ context.Context, _ *config.KubeRessourceIdentifier) error {
	return k.CreateAndWaitForRessourcesErr
}

func (k *stubK8sAPIWrapper) ExecuteCommandInPod(_ context.Context, _ *config.KubeExecConfig) error {
	return k.ExecuteCommandInPodErr
}

func (k *stubK8sAPIWrapper) CreatePodPortForward(_ context.Context, _ *config.KubeForwardConfig) error {
	return k.CreatePodPortForwardErr
}

func (k *stubK8sAPIWrapper) WriteFileInPod(_ context.Context, _ *config.KubeFileWriteConfig) error {
	return k.WriteFileInPodErr
}
