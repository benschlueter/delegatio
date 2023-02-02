/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package installer

import (
	"context"
	"net"
	"net/url"

	"github.com/benschlueter/delegatio/cli/installer/helm"
	"github.com/benschlueter/delegatio/internal/k8sapi"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// K8sapiWrapper is the struct used to access kubernetes helpers.
type K8sapiWrapper struct {
	Client *k8sapi.Client
	logger *zap.Logger
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewK8sClient(logger *zap.Logger) (*K8sapiWrapper, error) {
	// use the current context in kubeconfig
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	return &K8sapiWrapper{
		Client: client,
		logger: logger,
	}, nil
}

// InstallCilium installs cilium in the cluster.
func (k *K8sapiWrapper) InstallCilium(ctx context.Context) error {
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

// CreatePersistentVolume creates a shell on the specified pod.
func (k *K8sapiWrapper) CreatePersistentVolume(ctx context.Context, volumeName string) error {
	return k.Client.CreatePersistentVolume(ctx, volumeName, string(v1.ReadWriteMany))
}

// CreatePersistentVolumeClaim creates a shell on the specified pod.
func (k *K8sapiWrapper) CreatePersistentVolumeClaim(ctx context.Context, namespace, volumeName, storageClass string) error {
	return k.Client.CreatePersistentVolumeClaim(ctx, namespace, volumeName, storageClass)
}

// CreateConfigMapAndPutData creates a shell on the specified pod.
func (k *K8sapiWrapper) CreateConfigMapAndPutData(ctx context.Context, namespace, configMapName string, data map[string]string) error {
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
