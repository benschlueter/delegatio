/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"time"

	"github.com/benschlueter/delegatio/internal/kubernetes/helm"
	"github.com/benschlueter/delegatio/internal/kubernetes/helpers"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// Client is the struct used to access kubernetes helpers.
type Client struct {
	Client *helpers.Client
	logger *zap.Logger
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewK8sClient(logger *zap.Logger, kubeconfigPath string) (*Client, error) {
	// use the current context in kubeconfig
	client, err := helpers.NewClient(logger, kubeconfigPath)
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
		if err := k.Client.CreateServiceAccount(ctx, "testchallenge1", "development"); err != nil {
			return err
		}
		/* 		if err := k.Client.CreateClusterRoleBinding(ctx, "testchallenge1", "development"); err != nil {
			return err
		} */
		/* 		if err := k.Client.CreateSecret(ctx, "development"); err != nil {
			return err
		} */
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
func (k *Client) CreatePodShell(ctx context.Context, namespace, podName string, channel ssh.Channel, resizeQueue remotecommand.TerminalSizeQueue, tty bool) error {
	return k.Client.CreateExecInPod(ctx, namespace, podName, "bash", channel, channel, channel, resizeQueue, tty)
}

// ExecuteCommandInPod executes a command in the specified pod.
func (k *Client) ExecuteCommandInPod(ctx context.Context, namespace, podName, command string, channel ssh.Channel, resizeQueue remotecommand.TerminalSizeQueue, tty bool) error {
	return k.Client.CreateExecInPod(ctx, namespace, podName, command, channel, channel, channel, resizeQueue, tty)
}

// CreatePodPortForward creates a port forward on the specified pod.
func (k *Client) CreatePodPortForward(ctx context.Context, namespace, podName, port string, channel ssh.Channel) error {
	return k.Client.CreatePodPortForward(ctx, namespace, podName, port, channel)
}

// CreatePersistentVolume creates a shell on the specified pod.
func (k *Client) CreatePersistentVolume(ctx context.Context, volumeName string) error {
	return k.Client.CreatePersistentVolume(ctx, volumeName)
}

// CreatePersistentVolumeClaim creates a shell on the specified pod.
func (k *Client) CreatePersistentVolumeClaim(ctx context.Context, namespace, volumeName, storageClass string) error {
	return k.Client.CreatePersistentVolumeClaim(ctx, namespace, volumeName, storageClass)
}

// CreateConfigMapAndPutData creates a shell on the specified pod.
func (k *Client) CreateConfigMapAndPutData(ctx context.Context, namespace, configMapName string, data map[string]string) error {
	if err := k.Client.CreateConfigMap(ctx, namespace, configMapName); err != nil {
		return err
	}
	for key, value := range data {
		if err := k.Client.AddDataToConfigMap(ctx, namespace, configMapName, key, value); err != nil {
			return err
		}
	}
	return nil
}
