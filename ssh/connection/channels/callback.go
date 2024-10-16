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
	"github.com/benschlueter/delegatio/ssh/kubernetes"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// The function in this file are called to serve specific channel requests.

// callbackData is the data passed to the callbacks.
type callbackData struct {
	channel         ssh.Channel
	cancel          context.CancelFunc
	wg              *sync.WaitGroup
	log             *zap.Logger
	ptyReqData      *payload.PtyRequest
	directTCPIPData *payload.ForwardTCPChannelOpen
	terminalResizer *TerminalSizeHandler
	kubernetes.K8sAPIUser
}

// handleShell handles the "shell" request. This is used for "kubectl exec".
func (rd *callbackData) handleShell(ctx context.Context) {
	rd.log.Info("handleShell", zap.Any("pty", rd.ptyReqData))

	defer func() {
		rd.cancel()
		rd.wg.Done()
	}()
	// Fire up "kubectl exec" for this session
	tty := false
	if rd.ptyReqData != nil {
		if err := rd.terminalResizer.Fill(
			&remotecommand.TerminalSize{
				Width:  uint16(rd.ptyReqData.WidthColumns),
				Height: uint16(rd.ptyReqData.HeightRows),
			}); err != nil {
			rd.log.Error("failled to fill window", zap.Error(err))
		}
		tty = true
	}

	execConf := config.KubeExecConfig{
		Namespace:      rd.GetNamespace(),
		UserIdentifier: rd.GetAuthenticatedUserID(),
		Command:        "bash",
		Communication:  rd.channel,
		WinQueue:       rd.terminalResizer,
		Tty:            tty,
	}
	rd.log.Info("executeCommandInPod", zap.Any("config", execConf))
	if err := rd.ExecuteCommandInPod(ctx, &execConf); err != nil {
		rd.log.Error("executeCommandInPod exited", zap.Error(err))
		_, _ = rd.channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v | ", err)))
		return
	}
	rd.log.Debug("executeCommandInPod exited")
	_, _ = rd.channel.Write([]byte("graceful termination\n"))
}

// handleSubsystem handles the "subsystem" request. Currently only SFTP is supported.
// This is used by "scp" to copy files from the localhost to the pod or vice versa.
func (rd *callbackData) handleSubsystem(ctx context.Context, cmd string) {
	rd.log.Info("handleSubsystem callback", zap.String("subsystem", cmd))
	defer func() {
		rd.cancel()
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
		Namespace:      rd.GetNamespace(),
		UserIdentifier: fmt.Sprintf("%s-statefulset-0", rd.GetAuthenticatedUserID()),
		Command:        parsedSubsystem,
		Communication:  rd.channel,
		WinQueue:       rd.terminalResizer,
		Tty:            false,
	}
	err := rd.ExecuteCommandInPod(ctx, &execConf)
	if err != nil {
		rd.log.Error("ExecuteCommandInPod exited", zap.Error(err))
		_, _ = rd.channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v", err)))
	}
}

// handlePortForward handles the "direct-tcpip" request. This is used for "kubectl port-forward".
func (rd *callbackData) handlePortForward(ctx context.Context) {
	rd.log.Info("handlePortForward callback", zap.Any("data", rd.directTCPIPData))

	defer func() {
		rd.cancel()
		rd.wg.Done()
	}()
	forwardConf := config.KubeForwardConfig{
		Namespace:     rd.GetNamespace(),
		PodName:       fmt.Sprintf("%s-statefulset-0", rd.GetAuthenticatedUserID()),
		Communication: rd.channel,
		Port:          fmt.Sprint(rd.directTCPIPData.PortToConnect),
	}
	// this call will block until the context is cancelled, the channel is closed from the client side, or kubeapi is closing the channel (most likely an error).
	err := rd.CreatePodPortForward(ctx, &forwardConf)
	if err != nil {
		rd.log.Error("createPodPortForward exited", zap.Error(err))
		return
	}
}
