/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"errors"
	"io"
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

func TestSession(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		onReqDefaultCnt  int
		onReqCnt         int
		onReqSubSysCnt   int
		onReqPtyCnt      int
		onReqWindowChCnt int
		onReqShellCnt    int
		expectCloseErr   bool
		requests         []*ssh.Request
	}{
		"no requests": {
			expectCloseErr: false,
			onReqCnt:       0,
		},
		"unknown request type": {
			expectCloseErr:  false,
			onReqCnt:        1,
			onReqDefaultCnt: 1,
			requests: []*ssh.Request{
				{Type: "unknown stuff", WantReply: false},
			},
		},
		"unknown subsystem request": {
			expectCloseErr: false,
			onReqCnt:       1,
			onReqSubSysCnt: 1,
			requests: []*ssh.Request{
				{Type: "subsystem", WantReply: false, Payload: ssh.Marshal(payload.SubsystemRequest{Subsystem: "some-non-existent"})},
			},
		},
		"pty-req": {
			expectCloseErr: false,
			onReqCnt:       1,
			onReqPtyCnt:    1,
			requests: []*ssh.Request{
				{Type: "pty-req", WantReply: false, Payload: ssh.Marshal(payload.PtyRequest{})},
			},
		},
		"window change request": {
			expectCloseErr:   false,
			onReqCnt:         1,
			onReqWindowChCnt: 1,
			requests: []*ssh.Request{
				{Type: "window-change", WantReply: false, Payload: ssh.Marshal(payload.WindowChangeRequest{})},
			},
		},
		"multiple window change requests": {
			expectCloseErr:   false,
			onReqCnt:         3,
			onReqWindowChCnt: 3,
			requests: []*ssh.Request{
				{Type: "window-change", WantReply: false, Payload: ssh.Marshal(payload.WindowChangeRequest{})},
				{Type: "window-change", WantReply: false, Payload: ssh.Marshal(payload.WindowChangeRequest{})},
				{Type: "window-change", WantReply: false, Payload: ssh.Marshal(payload.WindowChangeRequest{})},
			},
		},
		"shell request": {
			expectCloseErr: true,
			onReqCnt:       1,
			onReqShellCnt:  1,
			requests: []*ssh.Request{
				{Type: "shell", WantReply: false},
			},
		},
		"sftp subsystem request": {
			expectCloseErr: false,
			onReqCnt:       1,
			onReqSubSysCnt: 1,
			requests: []*ssh.Request{
				{Type: "subsystem", WantReply: false, Payload: ssh.Marshal(payload.SubsystemRequest{Subsystem: "sftp"})},
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
			builder := SessionBuilderSkeleton()
			builder.SetRequests(requests)
			builder.SetChannel(stubChannel)
			builder.SetLog(log)
			builder.SetK8sUserAPI(
				&kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						execFunc: func(context.Context, *config.KubeExecConfig) error { return nil },
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "test-ns",
						UserIdentifier: "test-user",
					},
				},
			)

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
			reqSubSysCnt := 0
			builder.SetOnReqSubSys(
				func(context.Context, *ssh.Request, *callbackData) {
					reqSubSysCnt++
				},
			)
			reqPtyCnt := 0
			builder.SetOnReqPty(
				func(context.Context, *ssh.Request, *callbackData) {
					reqPtyCnt++
				},
			)
			reqWinChCnt := 0
			builder.SetOnReqWinCh(
				func(context.Context, *ssh.Request, *callbackData) {
					reqWinChCnt++
				},
			)
			reqShellCnt := 0
			builder.SetOnReqShell(
				func(context.Context, *ssh.Request, *callbackData) {
					reqShellCnt++
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
			assert.Equal(tc.onReqSubSysCnt, reqSubSysCnt)
			assert.Equal(tc.onReqPtyCnt, reqPtyCnt)
			assert.Equal(tc.onReqWindowChCnt, reqWinChCnt)
			assert.Equal(tc.onReqShellCnt, reqShellCnt)
		})
	}
}

func TestSessionBlockingExec(t *testing.T) {
	defer goleak.VerifyNone(t)
	testCases := map[string]struct {
		onReqCnt       int
		onReqSubSysCnt int
		onReqShellCnt  int
		closeByCtx     bool
		requests       []*ssh.Request
	}{
		"shell request context cancel": {
			closeByCtx:    true,
			onReqCnt:      1,
			onReqShellCnt: 1,
			requests: []*ssh.Request{
				{Type: "shell", WantReply: false},
			},
		},
		"shell request channel close": {
			closeByCtx:    false,
			onReqCnt:      1,
			onReqShellCnt: 1,
			requests: []*ssh.Request{
				{Type: "shell", WantReply: false},
			},
		},
		"sftp subsystem request context cancel": {
			closeByCtx:     true,
			onReqCnt:       1,
			onReqSubSysCnt: 1,
			requests: []*ssh.Request{
				{Type: "subsystem", WantReply: false, Payload: ssh.Marshal(payload.SubsystemRequest{Subsystem: "sftp"})},
			},
		},
		"sftp subsystem request channel close": {
			closeByCtx:     false,
			onReqCnt:       1,
			onReqSubSysCnt: 1,
			requests: []*ssh.Request{
				{Type: "subsystem", WantReply: false, Payload: ssh.Marshal(payload.SubsystemRequest{Subsystem: "sftp"})},
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
			builder := SessionBuilderSkeleton()
			builder.SetRequests(requests)
			builder.SetChannel(stubChannel)
			builder.SetLog(log)
			builder.SetK8sUserAPI(
				&kubernetes.K8sAPIUserWrapper{
					K8sAPI: &stubK8sAPIWrapper{
						execFunc: func(ctx context.Context, kec *config.KubeExecConfig) error {
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								if _, err := kec.Communication.Write([]byte("hello")); err != nil {
									return err
								}
							}
							return nil
						},
					},
					UserInformation: &config.KubeRessourceIdentifier{
						Namespace:      "test-ns",
						UserIdentifier: "test-user",
					},
				},
			)

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
			reqSubSysCnt := 0
			builder.SetOnReqSubSys(
				func(context.Context, *ssh.Request, *callbackData) {
					reqSubSysCnt++
				},
			)
			reqPtyCnt := 0
			builder.SetOnReqPty(
				func(context.Context, *ssh.Request, *callbackData) {
					reqPtyCnt++
				},
			)
			reqWinChCnt := 0
			builder.SetOnReqWinCh(
				func(context.Context, *ssh.Request, *callbackData) {
					reqWinChCnt++
				},
			)
			reqShellCnt := 0
			builder.SetOnReqShell(
				func(context.Context, *ssh.Request, *callbackData) {
					reqShellCnt++
				},
			)

			handler, err := builder.Build()
			require.NoError(err)

			ctx, cancel := context.WithCancel(context.Background())
			go handler.Serve(ctx)
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
			// close connection either by context or by channel close
			if tc.closeByCtx {
				cancel()
			} else {
				stubChannel.Close()
			}
			// wait for termination of the go routine
			handler.Wait()
			assert.Equal(tc.onReqCnt, reqCnt)
			assert.Equal(tc.onReqSubSysCnt, reqSubSysCnt)
			assert.Equal(tc.onReqShellCnt, reqShellCnt)
			cancel()
		})
	}
}

type stubChannel struct {
	reqChan chan *ssh.Request
	closed  bool
	mux     sync.Mutex
}

func (cs *stubChannel) Read(_ []byte) (int, error) {
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

func (cs *stubChannel) SendRequest(_ string, _ bool, _ []byte) (bool, error) {
	return true, nil
}

func (cs *stubChannel) Stderr() io.ReadWriter {
	return nil
}
