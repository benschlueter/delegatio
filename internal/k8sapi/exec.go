/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"io"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// CreateExecInPod creates a shell on the specified pod.
func (k *Client) CreateExecInPod(ctx context.Context, namespace, podName, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, resizeQueue remotecommand.TerminalSizeQueue, tty bool) error {
	cmd := []string{
		command,
	}
	req := k.Client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Command: cmd,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     tty,
	}

	req.VersionedParams(
		option,
		// The kubectl dependency here should be removed.
		scheme.ParameterCodec,
	)
	k.logger.Info("query kubeapi", zap.String("url", req.URL().String()))
	exec, err := remotecommand.NewSPDYExecutor(k.RestConfig, "POST", req.URL())
	if err != nil {
		return err
	}
	k.logger.Info("spawning shell in pod", zap.String("name", podName))
	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: resizeQueue,
	})
}
