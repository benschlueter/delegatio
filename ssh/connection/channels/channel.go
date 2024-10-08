/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Channel is the interface that wraps the Serve and Wait methods.
type Channel interface {
	Serve(ctx context.Context)
	Wait()
}

// Serve starts the server. It will block until the context is canceled or s.requests is closed.
func (h *Handler) Serve(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	// if one of the callbacks cancels the context, we want to stop the server
	// this will cancel all callbacks and wait for each of them to finish
	h.reqData.cancel = cancel
	defer func() {
		cancel()
		h.reqData.wg.Wait()
		if err := h.reqData.channel.Close(); err != nil {
			h.log.Error("failed to close channel", zap.Error(err))
		}
		h.log.Debug("stopped channel serve")
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
func (h *Handler) Wait() {
	<-h.serveCloseDone
}

// Handler handles incoming requests on a Handler.
type Handler struct {
	requests       <-chan *ssh.Request
	log            *zap.Logger
	serveCloseDone chan struct{}
	reqData        *callbackData

	funcMap            map[string][]func(context.Context, *ssh.Request, *callbackData)
	onEveryReqCallback []func(context.Context, *ssh.Request, *callbackData)
	onDefaultCallback  []func(context.Context, *ssh.Request, *callbackData)
	onStartupCallback  []func(context.Context, *callbackData)
}
