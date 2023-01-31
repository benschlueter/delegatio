/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package channels

import (
	"context"
	"fmt"
	"sync"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/ssh/connection/payload"
	"github.com/benschlueter/delegatio/ssh/local"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// The function in this file are called to serve specific channel requests.

// callbackData is the data passed to the callbacks.
type callbackData struct {
	channel         ssh.Channel
	wg              *sync.WaitGroup
	log             *zap.Logger
	ptyReq          *payload.PtyRequest
	directTCPIPData *payload.ForwardTCPChannelOpen
	terminalResizer *TerminalSizeHandler
	*local.Shared
}

// handleShell handles the "shell" request. This is used for "kubectl exec".
func (rd *callbackData) handleShell(ctx context.Context) {
	defer func() {
		if err := rd.channel.Close(); err != nil {
			rd.log.Error("failed to close channel", zap.Error(err))
		}
		rd.wg.Done()
	}()
	// Fire up "kubectl exec" for this session
	tty := false
	if rd.ptyReq != nil {
		if err := rd.terminalResizer.Fill(
			&remotecommand.TerminalSize{
				Width:  uint16(rd.ptyReq.WidthColumns),
				Height: uint16(rd.ptyReq.HeightRows),
			}); err != nil {
			rd.log.Error("failled to fill window", zap.Error(err))
		}
		tty = true
	}

	execConf := config.KubeExecConfig{
		Namespace:     rd.Namespace,
		PodName:       fmt.Sprintf("%s-statefulset-0", rd.AuthenticatedUserID),
		Command:       "bash",
		Communication: rd.channel,
		WinQueue:      rd.terminalResizer,
		Tty:           tty,
	}
	if err := rd.ExecFunc(ctx, &execConf); err != nil {
		rd.log.Error("createPodShell exited", zap.Error(err))
		_, _ = rd.channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v", err)))
		return
	}
	rd.log.Debug("createPodShell exited")
	_, _ = rd.channel.Write([]byte("graceful termination"))
}

// handleSubsystem handles the "subsystem" request. Currently only SFTP is supported.
// This is used by "scp" to copy files from the localhost to the pod or vice versa.
func (rd *callbackData) handleSubsystem(ctx context.Context, cmd string) {
	defer func() {
		if err := rd.channel.Close(); err != nil {
			rd.log.Error("failed to close channel", zap.Error(err))
		}
		rd.wg.Done()
	}()
	subSysMap := map[string]string{
		"sftp": "/usr/lib/ssh/sftp-server",
	}
	parsedSubsystem, ok := subSysMap[cmd]
	if !ok {
		rd.log.Error("unknown subsystem", zap.String("subsystem", cmd))
		return
	}

	execConf := config.KubeExecConfig{
		Namespace:     rd.Namespace,
		PodName:       fmt.Sprintf("%s-statefulset-0", rd.AuthenticatedUserID),
		Command:       parsedSubsystem,
		Communication: rd.channel,
		WinQueue:      rd.terminalResizer,
		Tty:           false,
	}
	err := rd.ExecFunc(ctx, &execConf)
	if err != nil {
		rd.log.Error("ExecuteCommandInPod exited", zap.Error(err))
		_, _ = rd.channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v", err)))
	}
}

// handlePortForward handles the "direct-tcpip" request. This is used for "kubectl port-forward".
func (rd *callbackData) handlePortForward(ctx context.Context) {
	defer func() {
		if err := rd.channel.Close(); err != nil {
			rd.log.Error("failed to close channel", zap.Error(err))
		}
		rd.wg.Done()
	}()
	forwardConf := config.KubeForwardConfig{
		Namespace:     rd.Namespace,
		PodName:       fmt.Sprintf("%s-statefulset-0", rd.AuthenticatedUserID),
		Communication: rd.channel,
		Port:          fmt.Sprint(rd.directTCPIPData.PortToConnect),
	}
	// this call will block until the context is cancelled, the channel is closed from the client side, or kubeapi is closing the channel (most likely an error).
	err := rd.ForwardFunc(ctx, &forwardConf)
	if err != nil {
		rd.log.Error("createPodPortForward exited", zap.Error(err))
		return
	}
}
