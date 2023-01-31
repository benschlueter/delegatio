/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// DirectTCPIPBuilderSkeleton returns a new ChannelHandlerBuilder, which has the corresponding field set to handle directTCPIP.
func DirectTCPIPBuilderSkeleton() *ChannelHandlerBuilder {
	builder := NewChannelBuilder().WithChannelType("direct-tcpip")

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
