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
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// TODO: test for zap messages
/*
   observedZapCore, observedLogs := observer.New(zap.InfoLevel)
   observedLogger := zap.New(observedZapCore)

   myFunction(observedLogger)

   require.Equal(t, 2, observedLogs.Len())
   allLogs := observedLogs.All()
   assert.Equal(t, "log myFunction", allLogs[0].Message)
   assert.Equal(t, "log with fields", allLogs[1].Message)
   assert.ElementsMatch(t, []zap.Field{
       {Key: "keyOne", String: "valueOne"},
       {Key: "keyTwo", St
*/

func TestKeepAlive(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test errr")
	testCases := map[string]struct {
		closeByCtx bool
		interval   time.Duration
		serverConn *ssh.ServerConn
	}{
		"close by context ": {
			closeByCtx: true,
			serverConn: &ssh.ServerConn{
				Conn: &stubConn{},
			},
			interval: time.Second,
		},
		"close by timeout ": {
			serverConn: &ssh.ServerConn{
				Conn: &stubConn{
					sendRequestErr: testErr,
				},
			},
			interval: time.Nanosecond,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			handler := connection{keepAliveInterval: tc.interval, log: zap.NewNop()}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan struct{})
			_, cancelHandler := handler.keepAlive(ctx, tc.serverConn, done)

			if tc.closeByCtx {
				cancelHandler()
			} else {
				<-done
			}
		})
	}
}

func TestHandleChannel(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test errr")
	testCases := map[string]struct {
		channel                ssh.NewChannel
		expectFinish           bool
		sessionHandlerFunc     func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared) (channels.Channel, error)
		directtcpIPHandlerFunc func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared, *payload.ForwardTCPChannelOpen) (channels.Channel, error)
	}{
		"session accept error": {
			channel: &stubNewChannel{
				channelType: "session",
				acceptErr:   testErr,
			},
			expectFinish: true,
		},
		"new session handler error": {
			channel: &stubNewChannel{
				channelType: "session",
			},
			expectFinish: true,
			sessionHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared) (channels.Channel, error) {
				return nil, testErr
			},
		},
		"session cancelled by context": {
			channel: &stubNewChannel{
				channelType: "session",
			},
			expectFinish: false,
			sessionHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared) (channels.Channel, error) {
				return &stubHandler{done: make(chan struct{})}, nil
			},
		},
		"unknows channel type": {
			channel: &stubNewChannel{
				channelType: "unknown stuff",
			},
			expectFinish: true,
		},
		"unknows channel type and reject error": {
			channel: &stubNewChannel{
				channelType: "unknown stuff",
				rejectErr:   testErr,
			},
			expectFinish: true,
		},
		"direct-tcpip unmarshal error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        []byte("invalid data"),
			},
			expectFinish: true,
		},
		"direct-tcpip unmarshal and reject error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        []byte("invalid data"),
				rejectErr:   testErr,
			},
			expectFinish: true,
		},
		"direct-tcpip accept error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				acceptErr:   testErr,
				data:        ssh.Marshal(payload.ForwardTCPChannelOpen{}),
			},
			expectFinish: true,
		},
		"new direct-tcpip handler error": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        ssh.Marshal(payload.ForwardTCPChannelOpen{}),
			},
			directtcpIPHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared, *payload.ForwardTCPChannelOpen) (channels.Channel, error) {
				return nil, testErr
			},
			expectFinish: true,
		},
		"direct-tcpip closed by ctx": {
			channel: &stubNewChannel{
				channelType: "direct-tcpip",
				data:        ssh.Marshal(payload.ForwardTCPChannelOpen{}),
			},
			directtcpIPHandlerFunc: func(*zap.Logger, ssh.Channel, <-chan *ssh.Request, *channels.Shared, *payload.ForwardTCPChannelOpen) (channels.Channel, error) {
				return &stubHandler{done: make(chan struct{})}, nil
			},
			expectFinish: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// assert := assert.New(t)
			// require := require.New(t)

			handler := connection{
				log:                   zap.NewNop(),
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
		})
	}
}

func TestHandleChannels(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		closeChannel    bool
		cancelCtx       bool
		channelElements []ssh.NewChannel
	}{
		"closed channel": {
			closeChannel: true,
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
		},
		"cancel ctx": {
			cancelCtx: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// assert := assert.New(t)
			// require := require.New(t)

			channelChan := make(chan ssh.NewChannel)
			handler := connection{
				log:     zap.NewNop(),
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
		})
	}
}

func TestHandleGlobalConnection(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test errr")
	testCases := map[string]struct {
		createWfunc   func(context.Context, *config.KubeRessourceIdentifier) error
		sshConnection *ssh.ServerConn
	}{
		"works": {
			createWfunc: func(context.Context, *config.KubeRessourceIdentifier) error { return nil },
			sshConnection: &ssh.ServerConn{
				Conn: &stubConn{},
			},
		},
		"createWaitFunc error": {
			createWfunc: func(context.Context, *config.KubeRessourceIdentifier) error { return testErr },
			sshConnection: &ssh.ServerConn{
				Conn: &stubConn{},
			},
		},
		"close error": {
			createWfunc: func(context.Context, *config.KubeRessourceIdentifier) error { return nil },
			sshConnection: &ssh.ServerConn{
				Conn: &stubConn{
					closeErr: testErr,
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// assert := assert.New(t)
			// require := require.New(t)

			channelChan := make(chan ssh.NewChannel)
			handler := connection{
				log:               zap.NewNop(),
				wg:                &sync.WaitGroup{},
				keepAliveInterval: time.Second,
				channel:           channelChan,
				Shared: &channels.Shared{
					Namespace:           "test-ns",
					AuthenticatedUserID: "test-uid",
				},
				createWaitFunc: tc.createWfunc,
				connection:     tc.sshConnection,
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			handler.HandleGlobalConnection(ctx)
		})
	}
}

type stubConn struct {
	sendRequestErr error
	openChannelErr error
	closeErr       error
	waitErr        error
}

func (c *stubConn) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	return true, nil, c.sendRequestErr
}

func (c *stubConn) OpenChannel(name string, data []byte) (ssh.Channel, <-chan *ssh.Request, error) {
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

func (s *stubNewChannel) Reject(reason ssh.RejectionReason, message string) error {
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
