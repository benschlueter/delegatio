/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/kubernetes/helm"
	"github.com/benschlueter/delegatio/internal/kubernetes/helpers"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
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
	u, err := url.Parse(k.Client.RestConfig.Host)
	if err != nil {
		return err
	}
	k.logger.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	return helm.Install(ctx, k.logger.Named("helm"), "cilium", host)
}

// CreateAndWaitForRessources creates the ressources for a user in a namespace.
func (k *Client) CreateAndWaitForRessources(ctx context.Context, conf *config.KubeRessourceIdentifier) error {
	exists, err := k.Client.StatefulSetExists(ctx, conf.Namespace, conf.UserIdentifier)
	if err != nil {
		return err
	}
	if !exists {
		/* 		if err := k.Client.CreateServiceAccount(ctx, namespace, "development"); err != nil {
			return err
		} */
		/* 		if err := k.Client.CreateClusterRoleBinding(ctx, "testchallenge1", "development"); err != nil {
			return err
		} */
		/* 		if err := k.Client.CreateSecret(ctx, "development"); err != nil {
			return err
		} */
		if err := k.Client.CreateStatefulSetForUser(ctx, conf.Namespace, conf.UserIdentifier); err != nil {
			return err
		}
	}
	if err := k.Client.WaitForPodRunning(ctx, conf.Namespace, conf.UserIdentifier, 1*time.Minute); err != nil {
		return err
	}
	return nil
}

// ExecuteCommandInPod executes a command in the specified pod.
func (k *Client) ExecuteCommandInPod(ctx context.Context, conf *config.KubeExecConfig) error {
	return k.Client.CreateExecInPod(ctx, conf.Namespace, conf.PodName, conf.Command, conf.Communication, conf.Communication, conf.Communication, conf.WinQueue, conf.Tty)
}

// CreatePodPortForward creates a port forward on the specified pod.
func (k *Client) CreatePodPortForward(ctx context.Context, conf *config.KubeForwardConfig) error {
	return k.Client.CreatePodPortForward(ctx, conf.Namespace, conf.PodName, conf.Port, conf.Communication)
}

// CreatePersistentVolume creates a shell on the specified pod.
func (k *Client) CreatePersistentVolume(ctx context.Context, volumeName string) error {
	return k.Client.CreatePersistentVolume(ctx, volumeName, string(v1.ReadWriteMany))
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
