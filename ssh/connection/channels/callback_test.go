/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func TestHandleShell(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		closeByCtx     bool
		closeByChannel bool
		closeByServer  bool
		serverErr      error
		requests       []*ssh.Request
	}{
		"close by context ": {
			closeByCtx: true,
		},

		"close by channel ": {
			closeByChannel: true,
		},

		"close by server ": {
			closeByChannel: true,
			serverErr:      errors.New("server error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)
			var execFunc func(ctx context.Context, kec *config.KubeExecConfig) error

			if tc.closeByServer {
				execFunc = func(ctx context.Context, kec *config.KubeExecConfig) error {
					kec.Communication.Close()
					return tc.serverErr
				}
			} else {
				execFunc = func(ctx context.Context, kec *config.KubeExecConfig) error {
					for {
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
							if _, err := kec.Communication.Write([]byte("hello")); err != nil {
								return err
							}
						}
					}
				}
			}

			requests := make(chan *ssh.Request, len(tc.requests)+1)
			stubChannel := &stubChannel{reqChan: requests}
			rd := &callbackData{
				channel:         stubChannel,
				wg:              &sync.WaitGroup{},
				log:             zap.NewNop(),
				terminalResizer: NewTerminalSizeHandler(10),
				Shared: &Shared{
					Namespace:           "ns-test",
					AuthenticatedUserID: "user-test",
					ExecFunc:            execFunc,
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			rd.wg.Add(1)
			go rd.handleShell(ctx)

			if tc.closeByCtx {
				cancel()
				rd.wg.Wait()
				assert.Error(stubChannel.Close())
			}
			if tc.closeByChannel {
				assert.NoError(stubChannel.Close())
				rd.wg.Wait()
			}
			if tc.closeByServer {
				rd.wg.Wait()
			}
			_, err := stubChannel.Write([]byte("hello"))
			assert.Error(err)
			cancel()
		})
	}
}

func TestHandleSubsystem(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		closeByCtx     bool
		closeByChannel bool
		closeByServer  bool
		serverErr      error
		requests       []*ssh.Request
	}{
		"close by context ": {
			closeByCtx: true,
		},

		"close by channel ": {
			closeByChannel: true,
		},

		"close by server ": {
			closeByChannel: true,
			serverErr:      errors.New("server error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)
			var execFunc func(ctx context.Context, kec *config.KubeExecConfig) error

			if tc.closeByServer {
				execFunc = func(ctx context.Context, kec *config.KubeExecConfig) error {
					kec.Communication.Close()
					return tc.serverErr
				}
			} else {
				execFunc = func(ctx context.Context, kec *config.KubeExecConfig) error {
					for {
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
							if _, err := kec.Communication.Write([]byte("hello")); err != nil {
								return err
							}
						}
					}
				}
			}

			requests := make(chan *ssh.Request, len(tc.requests)+1)
			stubChannel := &stubChannel{reqChan: requests}
			rd := &callbackData{
				channel:         stubChannel,
				wg:              &sync.WaitGroup{},
				log:             zap.NewNop(),
				terminalResizer: NewTerminalSizeHandler(10),
				Shared: &Shared{
					Namespace:           "ns-test",
					AuthenticatedUserID: "user-test",
					ExecFunc:            execFunc,
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			rd.wg.Add(1)
			go rd.handleSubsystem(ctx, "sftp")

			if tc.closeByCtx {
				cancel()
				rd.wg.Wait()
				assert.Error(stubChannel.Close())
			}
			if tc.closeByChannel {
				assert.NoError(stubChannel.Close())
				rd.wg.Wait()
			}
			if tc.closeByServer {
				rd.wg.Wait()
			}
			_, err := stubChannel.Write([]byte("hello"))
			assert.Error(err)
			cancel()
		})
	}
}

func TestHandlePortForward(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		closeByCtx     bool
		closeByChannel bool
		closeByServer  bool
		serverErr      error
		requests       []*ssh.Request
	}{
		"close by context ": {
			closeByCtx: true,
		},

		"close by channel ": {
			closeByChannel: true,
		},

		"close by server ": {
			closeByChannel: true,
			serverErr:      errors.New("server error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)
			var forwardFunc func(ctx context.Context, kec *config.KubeForwardConfig) error

			if tc.closeByServer {
				forwardFunc = func(ctx context.Context, kec *config.KubeForwardConfig) error {
					kec.Communication.Close()
					return tc.serverErr
				}
			} else {
				forwardFunc = func(ctx context.Context, kec *config.KubeForwardConfig) error {
					for {
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
							if _, err := kec.Communication.Write([]byte("hello")); err != nil {
								return err
							}
						}
					}
				}
			}

			requests := make(chan *ssh.Request, len(tc.requests)+1)
			stubChannel := &stubChannel{reqChan: requests}
			rd := &callbackData{
				channel:         stubChannel,
				wg:              &sync.WaitGroup{},
				log:             zap.NewNop(),
				directTCPIPData: &payload.ForwardTCPChannelOpen{},
				Shared: &Shared{
					Namespace:           "ns-test",
					AuthenticatedUserID: "user-test",
					ForwardFunc:         forwardFunc,
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			rd.wg.Add(1)
			go rd.handlePortForward(ctx)

			if tc.closeByCtx {
				cancel()
				rd.wg.Wait()
				assert.Error(stubChannel.Close())
			}
			if tc.closeByChannel {
				assert.NoError(stubChannel.Close())
				rd.wg.Wait()
			}
			if tc.closeByServer {
				rd.wg.Wait()
			}
			_, err := stubChannel.Write([]byte("hello"))
			assert.Error(err)
			cancel()
		})
	}
}
