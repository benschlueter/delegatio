package kubernetes

import (
	"context"
	"io"
	"time"

	"github.com/benschlueter/delegatio/cli/kubernetes/helm"
	"github.com/benschlueter/delegatio/cli/kubernetes/helpers"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/remotecommand"
)

// Client is the struct used to access kubernetes helpers.
type Client struct {
	Client *helpers.Client
	logger *zap.Logger
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
func NewK8sClient(kubeconfigPath string, logger *zap.Logger) (*Client, error) {
	// use the current context in kubeconfig
	client, err := helpers.NewClient(kubeconfigPath, logger)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client: client,
		logger: logger,
	}, nil
}

// InstallCilium installs cilium in the cluster.
func (k *Client) InstallCilium(ctx context.Context) error {
	return helm.Install(ctx, k.logger.Named("helm"), "cilium")
}

// CreateAndWaitForRessources creates the ressources for a user in a namespace.
func (k *Client) CreateAndWaitForRessources(ctx context.Context, namespace, userID string) error {
	exists, err := k.Client.StatefulSetExists(ctx, namespace, userID)
	if err != nil {
		return err
	}
	if !exists {
		if err := k.Client.CreateStatefulSetForUser(ctx, namespace, userID); err != nil {
			return err
		}
	}
	if err := k.Client.WaitForPodRunning(ctx, namespace, userID, 1*time.Minute); err != nil {
		return err
	}
	return nil
}

// CreatePodShell creates a shell on the specified pod.
func (k *Client) CreatePodShell(ctx context.Context, namespace, podName string, stdin io.Reader, stdout io.Writer, stderr io.Writer, resizeQueue remotecommand.TerminalSizeQueue) error {
	return k.Client.CreatePodShell(ctx, namespace, podName, stdin, stdout, stderr, resizeQueue)
}

// CreatePersistentVolume creates a shell on the specified pod.
func (k *Client) CreatePersistentVolume(ctx context.Context, namespace, volumeName string) error {
	/* 	if err := exec.Command("kubectl", "apply", "-f", "secret.yaml").Run(); err != nil {
		return err
	} */
	if err := k.Client.CreateSecret(ctx); err != nil {
		return err
	}
	return k.Client.CreatePersistentVolume(ctx, namespace, volumeName)
}

// CreatePersistentVolumeClaim creates a shell on the specified pod.
func (k *Client) CreatePersistentVolumeClaim(ctx context.Context, namespace, volumeName string) error {
	return k.Client.CreatePersistentVolumeClaim(ctx, namespace, volumeName)
}
