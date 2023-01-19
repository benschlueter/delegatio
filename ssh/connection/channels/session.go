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
	"k8s.io/client-go/tools/remotecommand"
)

// NewSessionHandler returns a new SSHChannelServer.
func NewSessionHandler(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, namespace, userID string, execFunc func(context.Context, *config.KubeExecConfig) error) (*ChannelHandler, error) {
	return NewSessionBuilder(log, channel, requests, namespace, userID, execFunc).Build()
}

// NewSessionBuilder returns a ChannelHandlerBuilder, which has the corresponding field set to handle sessions.
func NewSessionBuilder(log *zap.Logger, channel ssh.Channel, requests <-chan *ssh.Request, namespace, userID string, execFunc func(context.Context, *config.KubeExecConfig) error) *ChannelHandlerBuilder {
	builder := NewChannelBuilder().WithChannelType("session")
	builder.SetRequests(requests)
	builder.SetChannel(channel)
	builder.SetLog(log)
	builder.SetNamespace(namespace)
	builder.SetUserID(userID)
	builder.SetOnKubeExec(execFunc)

	builder.SetOnStartup(func(ctx context.Context, rd *callbackData) {
		rd.log.Info("starting session handler")
	})

	builder.SetOnReqDefault(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		if err := req.Reply(false, nil); err != nil {
			rd.log.Error("failled to respond to request", zap.Any("request", req), zap.Error(err))
		}
		rd.log.Info("unimplemented request", zap.Any("request", req))
	})

	builder.SetOnReqShell(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		rd.log.Info("shell request", zap.Any("data", req.Payload))
		rd.wg.Add(1)
		go rd.handleShell(ctx)
		if err := req.Reply(true, nil); err != nil {
			rd.log.Error("failled to reply to \"shell\" request", zap.Error(err))
		}
	})

	builder.SetOnReqSubSys(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		subSys := payload.SubsystemRequest{}
		if err := ssh.Unmarshal(req.Payload, &subSys); err != nil {
			rd.log.Error("failled to unmarshal window-change request", zap.Error(err))
			return
		}
		rd.log.Info("subsystem request", zap.Any("data", subSys))
		rd.wg.Add(1)
		go rd.handleSubsystem(ctx, subSys.Subsystem)
		if err := req.Reply(true, nil); err != nil {
			rd.log.Error("failled to respond to \"subsystem\" request", zap.Error(err))
		}
	})

	builder.SetOnReqWinCh(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		windowChange := payload.WindowChangeRequest{}
		if err := ssh.Unmarshal(req.Payload, &windowChange); err != nil {
			rd.log.Error("failled to unmarshal window-change request", zap.Error(err))
			return
		}
		rd.log.Info("window-change request", zap.Any("data", windowChange))
		if err := rd.terminalResizer.Fill(&remotecommand.TerminalSize{
			Width:  uint16(windowChange.WidthColumns),
			Height: uint16(windowChange.HeightRows),
		}); err != nil {
			rd.log.Error("failled to fill window", zap.Error(err))
		}
	})

	builder.SetOnReqPty(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		ptyReq := payload.PtyRequest{}
		if err := ssh.Unmarshal(req.Payload, &ptyReq); err != nil {
			rd.log.Error("failled to unmarshal pty request", zap.Error(err))
			return
		}
		rd.log.Info("pty request", zap.Any("data", ptyReq))
		rd.ptyReq = &ptyReq
		if err := req.Reply(true, nil); err != nil {
			rd.log.Error("failled to respond to \"pty-req\" request", zap.Error(err))
		}
	})

	builder.SetOnRequest(func(ctx context.Context, req *ssh.Request, rd *callbackData) {
		rd.log.Debug("request", zap.Any("data", req))
	})
	return builder
}
