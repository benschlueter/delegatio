/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// ChannelHandler handles incoming requests on a channel.
type ChannelHandler struct {
	requests       <-chan *ssh.Request
	log            *zap.Logger
	serveCloseDone chan struct{}
	reqData        *callbackData

	funcMap            map[string][]func(context.Context, *ssh.Request, *callbackData)
	onEveryReqCallback []func(context.Context, *ssh.Request, *callbackData)
	onDefaultCallback  []func(context.Context, *ssh.Request, *callbackData)
	onStartupCallback  []func(context.Context, *callbackData)
}

// Serve starts the server. It will block until the context is canceled or s.requests is closed.
func (h *ChannelHandler) Serve(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		h.log.Info("before wg wait")
		h.reqData.wg.Wait()
		h.log.Info("serverclosedone")
		h.serveCloseDone <- struct{}{}
	}()

	for _, funct := range h.onStartupCallback {
		funct(ctx, h.reqData)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-h.requests:
			if !ok {
				h.log.Debug("request channel closed")
				return
			}
			// h.log.Debug("received data over request channel", zap.Any("req", req))
			if callbackFuncions, ok := h.funcMap[req.Type]; ok {
				for _, callbackFuncion := range callbackFuncions {
					callbackFuncion(ctx, req, h.reqData)
				}
			} else {
				for _, funct := range h.onDefaultCallback {
					funct(ctx, req, h.reqData)
				}
			}
			for _, funct := range h.onEveryReqCallback {
				funct(ctx, req, h.reqData)
			}
		}
	}
}

// Wait waits until serve has finished (including all goroutines started by it).
func (h *ChannelHandler) Wait() {
	<-h.serveCloseDone
}
