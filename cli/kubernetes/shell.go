package kubernetes

import (
	"context"
	"io"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// ExecCmd exec command on specific pod and wait the command's output.
func (k *kubernetesClient) CreatePodShell(ctx context.Context, namespace, podName string, stdin io.Reader, stdout io.Writer, stderr io.Writer, resizeQueue remotecommand.TerminalSizeQueue) error {
	cmd := []string{
		"bash",
	}
	req := k.client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Command: cmd,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(k.restClient, "POST", req.URL())
	if err != nil {
		return err
	}
	k.logger.Info("spawning shell in pod", zap.String("name", podName))
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               true,
		TerminalSizeQueue: resizeQueue,
	})
	if err != nil {
		return err
	}
	return nil
}
