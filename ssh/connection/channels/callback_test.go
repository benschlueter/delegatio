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
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
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
				execFunc = func(_ context.Context, kec *config.KubeExecConfig) error {
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
				K8sAPIUser: &kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						execFunc: execFunc,
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "ns-test",
						UserIdentifier: "user-test",
					},
				},
				cancel: func() {},
			}
			ctx, cancel := context.WithCancel(context.Background())
			rd.wg.Add(1)
			go rd.handleShell(ctx)

			if tc.closeByCtx {
				cancel()
				rd.wg.Wait()
				assert.NoError(stubChannel.Close())
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
				execFunc = func(_ context.Context, kec *config.KubeExecConfig) error {
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
				K8sAPIUser: &kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						execFunc: execFunc,
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "ns-test",
						UserIdentifier: "user-test",
					},
				},
				cancel: func() {},
			}
			ctx, cancel := context.WithCancel(context.Background())
			rd.wg.Add(1)
			go rd.handleSubsystem(ctx, "sftp")

			if tc.closeByCtx {
				cancel()
				rd.wg.Wait()
				assert.NoError(stubChannel.Close())
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
				forwardFunc = func(_ context.Context, kec *config.KubeForwardConfig) error {
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
				log:             zaptest.NewLogger(t),
				directTCPIPData: &payload.ForwardTCPChannelOpen{},
				K8sAPIUser: &kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						forwardFunc: forwardFunc,
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "ns-test",
						UserIdentifier: "user-test",
					},
				},
				cancel: func() {},
			}
			ctx, cancel := context.WithCancel(context.Background())
			rd.wg.Add(1)
			go rd.handlePortForward(ctx)

			if tc.closeByCtx {
				cancel()
				rd.wg.Wait()
				assert.NoError(stubChannel.Close())
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

type stubK8sAPIWrapper struct {
	CreateAndWaitForRessourcesErr error
	execFunc                      func(ctx context.Context, kec *config.KubeExecConfig) error
	forwardFunc                   func(ctx context.Context, kec *config.KubeForwardConfig) error
	writeFunc                     func(ctx context.Context, kec *config.KubeFileWriteConfig) error
}

func (k *stubK8sAPIWrapper) CreateAndWaitForRessources(_ context.Context, _ *config.KubeRessourceIdentifier) error {
	return nil
}

func (k *stubK8sAPIWrapper) ExecuteCommandInPod(ctx context.Context, conf *config.KubeExecConfig) error {
	return k.execFunc(ctx, conf)
}

func (k *stubK8sAPIWrapper) CreatePodPortForward(ctx context.Context, conf *config.KubeForwardConfig) error {
	return k.forwardFunc(ctx, conf)
}

func (k *stubK8sAPIWrapper) WriteFileInPod(ctx context.Context, conf *config.KubeFileWriteConfig) error {
	return k.writeFunc(ctx, conf)
}
