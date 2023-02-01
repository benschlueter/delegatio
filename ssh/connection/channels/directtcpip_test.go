/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/local"
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
		"unexcepted requests": {
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

			requests := make(chan *ssh.Request, len(tc.requests)+1)
			stubChannel := &stubChannel{reqChan: requests}
			log := zap.NewNop()
			builder := DirectTCPIPBuilderSkeleton()
			builder.SetRequests(requests)
			builder.SetChannel(stubChannel)
			builder.SetLog(log)
			builder.SetSharedData(&local.Shared{
				ForwardFunc:         func(ctx context.Context, kec *config.KubeForwardConfig) error { return nil },
				AuthenticatedUserID: "test-user",
				Namespace:           "test-ns",
			},
			)
			builder.SetDirectTCPIPData(&payload.ForwardTCPChannelOpen{})

			for _, v := range tc.requests {
				requests <- v
			}
			reqMux := sync.Mutex{}
			reqCnt := 0
			builder.SetOnRequest(
				func(ctx context.Context, r *ssh.Request, rq *callbackData) {
					reqMux.Lock()
					reqCnt++
					reqMux.Unlock()
				},
			)
			reqDefaultCnt := 0
			builder.SetOnReqDefault(
				func(ctx context.Context, r *ssh.Request, rq *callbackData) {
					reqDefaultCnt++
				},
			)
			reqStartupCnt := 0
			builder.SetOnStartup(
				func(ctx context.Context, rd *callbackData) {
					reqStartupCnt++
				},
			)

			handler, err := builder.Build()
			require.NoError(err)
			go handler.Serve(context.Background())
			timeout := time.After(100 * time.Millisecond)
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
				}
			}
			// Wait for all goroutines to finish
			handler.reqData.wg.Wait()
			if tc.expectCloseErr {
				assert.Error(stubChannel.Close())
			} else {
				stubChannel.Close()
			}
			// wait for termination of the go routine
			handler.Wait()
			assert.Equal(tc.onReqCnt, reqCnt)
			assert.Equal(tc.onReqDefaultCnt, reqDefaultCnt)
			assert.Equal(tc.onStartupCnt, reqStartupCnt)
		})
	}
}
