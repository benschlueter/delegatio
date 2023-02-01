/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package connection

import (
	"context"
	"errors"
	"io"
	"reflect"
	"runtime"
	"sync"
	"testing"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/local"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func TestBuilder(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		expectErr            bool
		skipSetConn          bool
		skipSetFunc          bool
		skipSetPermissions   bool
		skipSetPermissionKey bool
		skipSetLogger        bool
		compareEverything    bool
	}{
		"no error": {
			compareEverything: true,
		},
		"no functions set": {
			expectErr:   true,
			skipSetFunc: true,
		},
		"no connection set": {
			expectErr:   true,
			skipSetConn: true,
		},
		"no permissions set": {
			expectErr:          true,
			skipSetPermissions: true,
		},
		"no permission key set": {
			expectErr:            true,
			skipSetPermissionKey: true,
		},
		"no logger set": {
			expectErr:     true,
			skipSetLogger: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			builder := NewBuilder()
			globalRequests := make(<-chan *ssh.Request)
			channel := make(<-chan ssh.NewChannel)
			connection := &ssh.ServerConn{
				Conn: &stubConn{},
			}
			log := zap.NewNop()
			forwardFunc := func(context.Context, *config.KubeForwardConfig) error { return nil }
			execFunc := func(context.Context, *config.KubeExecConfig) error { return nil }
			createWaitFunc := func(context.Context, *config.KubeRessourceIdentifier) error { return nil }

			if !tc.skipSetPermissions {
				connection.Permissions = &ssh.Permissions{
					Extensions: map[string]string{},
				}
			}
			if !tc.skipSetPermissionKey && !tc.skipSetPermissions {
				connection.Permissions.Extensions[config.AuthenticatedUserID] = "test"
			}

			if !tc.skipSetFunc {
				builder.SetExecFunc(execFunc)
				builder.SetForwardFunc(forwardFunc)
				builder.SetRessourceFunc(createWaitFunc)
			}

			if !tc.skipSetConn {
				builder.SetConnection(connection)
			}
			if !tc.skipSetLogger {
				builder.SetLogger(log)
			}
			builder.SetChannel(channel)
			builder.SetGlobalRequests(globalRequests)

			if tc.compareEverything {
				assert.Equal(builder.channel, channel)
				assert.Equal(builder.connection, connection)
				assert.Equal(builder.log, log)
				assert.Equal(builder.globalRequests, globalRequests)
				funcName1 := runtime.FuncForPC(reflect.ValueOf(execFunc).Pointer()).Name()
				funcName2 := runtime.FuncForPC(reflect.ValueOf(builder.execFunc).Pointer()).Name()
				assert.Equal(funcName1, funcName2)
				funcName1 = runtime.FuncForPC(reflect.ValueOf(forwardFunc).Pointer()).Name()
				funcName2 = runtime.FuncForPC(reflect.ValueOf(builder.forwardFunc).Pointer()).Name()
				assert.Equal(funcName1, funcName2)
				funcName1 = runtime.FuncForPC(reflect.ValueOf(createWaitFunc).Pointer()).Name()
				funcName2 = runtime.FuncForPC(reflect.ValueOf(createWaitFunc).Pointer()).Name()
				assert.Equal(funcName1, funcName2)
			}
			_, err := builder.Build()

			if tc.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		expectErr  bool
		setLog     bool
		setChannel bool
		setRequest bool
		setShared  bool
	}{
		"no error": {
			setLog:     true,
			setChannel: true,
			setRequest: true,
			setShared:  true,
		},
		"no log set": {
			expectErr:  true,
			setChannel: true,
			setRequest: true,
			setShared:  true,
		},
		"no shared set": {
			expectErr:  true,
			setChannel: true,
			setRequest: true,
			setLog:     true,
		},
		"no channel set": {
			expectErr:  true,
			setLog:     true,
			setRequest: true,
			setShared:  true,
		},
		"no requests set": {
			expectErr:  true,
			setChannel: true,
			setLog:     true,
			setShared:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var log *zap.Logger
			var channel ssh.Channel
			var request <-chan *ssh.Request
			var shared *local.Shared

			if tc.setLog {
				log = zap.NewNop()
			}
			if tc.setChannel {
				channel = &stubChannel{}
			}
			if tc.setRequest {
				request = make(<-chan *ssh.Request)
			}
			if tc.setShared {
				shared = &local.Shared{}
			}

			_, err := newSession(log, channel, request, shared)

			if tc.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestNewDirectTCPIP(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		expectErr    bool
		setLog       bool
		setChannel   bool
		setRequest   bool
		setShared    bool
		setTCPIPData bool
	}{
		"no error": {
			setLog:       true,
			setChannel:   true,
			setRequest:   true,
			setShared:    true,
			setTCPIPData: true,
		},
		"no log set": {
			expectErr:    true,
			setChannel:   true,
			setRequest:   true,
			setShared:    true,
			setTCPIPData: true,
		},
		"no shared set": {
			expectErr:    true,
			setChannel:   true,
			setRequest:   true,
			setLog:       true,
			setTCPIPData: true,
		},
		"no channel set": {
			expectErr:    true,
			setLog:       true,
			setRequest:   true,
			setShared:    true,
			setTCPIPData: true,
		},
		"no requests set": {
			expectErr:    true,
			setChannel:   true,
			setLog:       true,
			setShared:    true,
			setTCPIPData: true,
		},
		"no tcpIPData set": {
			expectErr:  true,
			setChannel: true,
			setLog:     true,
			setShared:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var log *zap.Logger
			var channel ssh.Channel
			var request <-chan *ssh.Request
			var shared *local.Shared
			var tcpIPData *payload.ForwardTCPChannelOpen

			if tc.setLog {
				log = zap.NewNop()
			}
			if tc.setChannel {
				channel = &stubChannel{}
			}
			if tc.setRequest {
				request = make(<-chan *ssh.Request)
			}
			if tc.setShared {
				shared = &local.Shared{}
			}
			if tc.setTCPIPData {
				tcpIPData = &payload.ForwardTCPChannelOpen{}
			}

			_, err := newDirectTCPIP(log, channel, request, shared, tcpIPData)

			if tc.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

type stubChannel struct {
	reqChan chan *ssh.Request
	closed  bool
	mux     sync.Mutex
}

func (cs *stubChannel) Read(data []byte) (int, error) {
	return 0, nil
}

func (cs *stubChannel) Write(data []byte) (int, error) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if !cs.closed {
		return len(data), nil
	}
	return 0, errors.New("already closed")
}

func (cs *stubChannel) Close() error {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if !cs.closed {
		cs.closed = true
		close(cs.reqChan)
		return nil
	}
	return errors.New("already closed")
}

func (cs *stubChannel) CloseWrite() error {
	return nil
}

func (cs *stubChannel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return true, nil
}

func (cs *stubChannel) Stderr() io.ReadWriter {
	return nil
}
