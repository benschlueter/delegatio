/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func TestDirectTCPIP(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		onReqDefaultCnt int
		onReqCnt        int
		onStartupCnt    int
		expectCloseErr  bool
		requests        []*ssh.Request
	}{
		"no requests": {
			expectCloseErr: false,
			onReqCnt:       0,
			onStartupCnt:   1,
		},
		"unexpected requests": {
			expectCloseErr:  false,
			onReqCnt:        2,
			onReqDefaultCnt: 2,
			onStartupCnt:    1,
			requests: []*ssh.Request{
				{Type: "unknown stuff 1", WantReply: false},
				{Type: "unknown stuff 2", WantReply: false},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			requests := make(chan *ssh.Request, len(tc.requests))
			stubChannel := &stubChannel{reqChan: requests}
			log := zap.NewNop()
			// Attention the default callbacks are also registered
			builder := DirectTCPIPBuilderSkeleton()
			builder.SetRequests(requests)
			builder.SetChannel(stubChannel)
			builder.SetLog(log)
			builder.SetK8sUserAPI(
				&kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						forwardFunc: func(ctx context.Context, _ *config.KubeForwardConfig) error {
							<-ctx.Done()
							return nil
						},
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "test-ns",
						UserIdentifier: "test-user",
					},
				},
			)
			builder.SetDirectTCPIPData(&payload.ForwardTCPChannelOpen{})

			for _, v := range tc.requests {
				requests <- v
			}
			reqMux := sync.Mutex{}
			reqCnt := 0
			builder.SetOnRequest(
				func(context.Context, *ssh.Request, *callbackData) {
					reqMux.Lock()
					reqCnt++
					reqMux.Unlock()
				},
			)
			reqDefaultCnt := 0
			builder.SetOnReqDefault(
				func(context.Context, *ssh.Request, *callbackData) {
					reqDefaultCnt++
				},
			)
			reqStartupCnt := 0
			builder.SetOnStartup(
				func(context.Context, *callbackData) {
					reqStartupCnt++
				},
			)

			handler, err := builder.Build()
			require.NoError(err)
			ctx, cancel := context.WithCancel(context.Background())
			go handler.Serve(ctx)
			timeout := time.After(1000 * time.Millisecond)
		O:
			for {
				select {
				case <-timeout:
					break O
				default:
					reqMux.Lock()
					// This ensures that all requests are processed and all goroutines (if any) are started
					if len(requests) == 0 && reqCnt == len(tc.requests) {
						reqMux.Unlock()
						break O
					}
					reqMux.Unlock()
					runtime.Gosched()
				}
			}

			// Wait for all goroutines to finish
			cancel()
			// wait for termination of the go routine
			handler.Wait()
			assert.Equal(tc.onReqCnt, reqCnt)
			assert.Equal(tc.onReqDefaultCnt, reqDefaultCnt)
			assert.Equal(tc.onStartupCnt, reqStartupCnt)
		})
	}
}
