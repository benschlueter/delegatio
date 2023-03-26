/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"io"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

// CreateExecInPodgRPC creates a shell on the specified pod.
func (k *Client) CreateExecInPodgRPC(ctx context.Context, namespace, podName, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, resizeQueue remotecommand.TerminalSizeQueue, tty bool) error {
	endpoint := "localhost:9000"

	conn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ExecCommandStream(ctx)
	if err != nil {
		return err
	}
	err = resp.Send(&vmproto.ExecCommandStreamRequest{
		Content: &vmproto.ExecCommandStreamRequest_Command{
			Command: &vmproto.ExecCommandRequest{
				Command: command,
				Tty:     true,
			},
		},
	})
	if err != nil {
		return err
	}
	var copier []byte
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, err := resp.Recv()
			if err != nil {
				return err
			}
			if len(data.GetStderr()) > 0 {
				stderr.Write(data.GetStderr())
			}
			if len(data.GetStdout()) > 0 {
				stdout.Write(data.GetStdout())
			}
			n, err := stdin.Read(copier)
			if err != nil {
				return err
			}
			err = resp.Send(&vmproto.ExecCommandStreamRequest{
				Content: &vmproto.ExecCommandStreamRequest_Stdin{
					Stdin: copier[:n],
				},
			})
			if err != nil {
				return err
			}
		}
	}
}
