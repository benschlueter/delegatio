/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package apps

import (
	"context"
	"net"
	"net/url"
	"os"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/installer"
	"go.uber.org/zap"
)

const sshNamespaceName = "ssh"

// InitializeSSH initializes the SSH application.
func InitializeSSH(ctx context.Context, log *zap.Logger, kubeClient *installer.Client, creds *config.EtcdCredentials) error {
	if err := kubeClient.Client.CreateNamespace(ctx, sshNamespaceName); err != nil {
		log.With(zap.Error(err)).Error("failed to create namespace")
		return err
	}

	u, err := url.Parse(kubeClient.Client.RestConfig.Host)
	if err != nil {
		return err
	}
	log.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	if err := kubeClient.Client.ConnectToStore(creds, []string{net.JoinHostPort(host, "2379")}); err != nil {
		log.With(zap.Error(err)).Error("failed to connect to store")
		return err
	}
	configMapData := map[string]string{
		"key":           string(creds.KeyData),
		"cert":          string(creds.PeerCertData),
		"caCert":        string(creds.CaCertData),
		"advertiseAddr": host,
	}
	if err := kubeClient.CreateConfigMapAndPutData(ctx, sshNamespaceName, "etcd-credentials", configMapData); err != nil {
		log.With(zap.Error(err)).Error("failed to CreatePersistentVolumeClaim")
		return err
	}
	privateBytes, err := os.ReadFile("./server_test")
	if err != nil {
		return err
	}
	if err := kubeClient.Client.UploadSSHServerPrivKey(privateBytes); err != nil {
		return err
	}
	log.Info("uploaded ssh server private key")
	if err := kubeClient.Client.CreateServiceAccount(ctx, sshNamespaceName, "development"); err != nil {
		return err
	}
	if err := kubeClient.Client.CreateClusterRoleBinding(ctx, sshNamespaceName, "development"); err != nil {
		return err
	}

	if err := kubeClient.Client.CreateDeployment(ctx, sshNamespaceName, "ssh-relay", int32(config.ClusterConfiguration.NumberOfWorkers)); err != nil {
		return err
	}
	if err := kubeClient.Client.CreateService(ctx, sshNamespaceName, "ssh-relay"); err != nil {
		return err
	}
	if err := kubeClient.Client.CreateIngress(ctx, sshNamespaceName); err != nil {
		return err
	}
	return nil
}
