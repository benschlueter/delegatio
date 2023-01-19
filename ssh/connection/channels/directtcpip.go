/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// NewDirectTCPIPHandler returns a new DirectTCPIPHandler.
func NewDirectTCPIPHandler(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, namespace, userID string, forwardFunc func(context.Context, *config.KubeForwardConfig) error, data *payload.ForwardTCPChannelOpen) (*ChannelHandler, error) {
	return NewDirectTCPIPBuilder(log, channel, requests, namespace, userID, forwardFunc, data).Build()
}

// NewDirectTCPIPBuilder returns a new ChannelHandlerBuilder, which has the corresponding field set to handle directTCPIP.
func NewDirectTCPIPBuilder(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, namespace, userID string, forwardFunc func(context.Context, *config.KubeForwardConfig) error, data *payload.ForwardTCPChannelOpen) *ChannelHandlerBuilder {
	builder := NewChannelBuilder().WithChannelType("direct-tcpip")
	builder.SetRequests(requests)
	builder.SetChannel(channel)
	builder.SetLog(log)
	builder.SetNamespace(namespace)
	builder.SetUserID(userID)
	builder.SetOnKubeForward(forwardFunc)
	builder.SetDirectTCPIPData(data)

	builder.SetOnReqDefault(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		if err := req.Reply(false, nil); err != nil {
			rd.log.Error("failled to respond to request", zap.Any("request", req), zap.Error(err))
		}
		rd.log.Info("unimplemented request", zap.Any("request", req))
	})

	builder.SetOnRequest(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		rd.log.Debug("request", zap.Any("data", req))
	})

	builder.SetOnStartup(func(ctx context.Context, rd *callbackData) {
		rd.log.Info("starting direct-tcpip handler")
		rd.wg.Add(1)
		go rd.handlePortForward(ctx)
	})

	return builder
}
