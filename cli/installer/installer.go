/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package installer

import (
	"context"
	"net"
	"net/url"
	"os"

	"github.com/benschlueter/delegatio/cli/installer/helm"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// Installer is the struct used to access kubernetes helpers.
type Installer struct {
	Client *k8sapi.Client
	logger *zap.Logger
}

// NewInstaller returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewInstaller(logger *zap.Logger) (*Installer, error) {
	// use the current context in kubeconfig
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	return &Installer{
		Client: client,
		logger: logger.Named("installer"),
	}, nil
}

// InstallKubernetesApplications installs all the kubernetes applications.
func (k *Installer) InstallKubernetesApplications(ctx context.Context, creds *config.EtcdCredentials, config *config.UserConfiguration) error {
	if err := k.InstallCilium(ctx); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to install helm charts")
		return err
	}
	if err := k.InitializeSSH(ctx, k.logger.Named("ssh"), creds); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to deploy ssh config")
		return err
	}
	if err := k.InitalizeChallenges(ctx, k.logger.Named("challenges"), config); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to deploy challenges")
		return err
	}
	return nil
}

// InstallCilium installs cilium in the cluster.
func (k *Installer) InstallCilium(ctx context.Context) error {
	u, err := url.Parse(k.Client.RestConfig.Host)
	if err != nil {
		return err
	}
	k.logger.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	vals := map[string]interface{}{
		"kubeProxyReplacement": "strict",
		"k8sServicePort":       "6443",
		"k8sServiceHost":       host,
		/* 		"prometheus.enabled":          "true",
		   		"operator.prometheus.enabled": true, */
	}
	helmInstaller := helm.NewHelmInstaller(k.logger, "cilium", "cilium", config.CiliumPath, config.Cilium256Hash, vals)
	return helmInstaller.Install(ctx)
}

// InstallTetragon installs tetragon in the cluster.
func (k *Installer) InstallTetragon(ctx context.Context) error {
	helmInstaller := helm.NewHelmInstaller(k.logger, "tetragon", "kube-system", config.TetratePath, config.Tetragon256Hash, nil)
	return helmInstaller.Install(ctx)
}

// InitalizeChallenges creates the namespaces and persistent volumes for the challenges. It also adds the users to etcd.
func (k *Installer) InitalizeChallenges(ctx context.Context, log *zap.Logger, config *config.UserConfiguration) error {
	if err := k.Client.CreateStorageClass(ctx, "nfs", "Retain"); err != nil {
		log.With(zap.Error(err)).Error("failed to CreateStorageClass")
		return err
	}
	stWrapper := storewrapper.StoreWrapper{Store: k.Client.SharedStore}

	for namespace := range config.Challenges {
		if err := k.Client.CreateNamespace(ctx, namespace); err != nil {
			log.With(zap.Error(err)).Error("failed to create namespace")
			return err
		}
		log.Info("created namespace for challenge", zap.String("challenge", namespace))
		if err := k.createPersistentVolume(ctx, namespace); err != nil {
			log.With(zap.Error(err)).Error("failed to CreatePersistentVolume")
			return err
		}
		log.Info("created pv for challenge", zap.String("challenge", namespace))
		if err := k.createPersistentVolumeClaim(ctx, namespace, namespace, "nfs"); err != nil {
			log.With(zap.Error(err)).Error("failed to CreatePersistentVolumeClaim")
			return err
		}
		log.Info("created pvc for challenge", zap.String("challenge", namespace))

		if err := stWrapper.PutChallengeData(namespace, nil); err != nil {
			return err
		}
		log.Info("added challenge to store", zap.String("challenge", namespace))

	}
	for publicKey, realName := range config.PubKeyToUser {
		if err := stWrapper.PutPublicKeyData(publicKey, realName); err != nil {
			return err
		}
		log.Info("added user to store", zap.String("publicKey", publicKey), zap.Any("userinfo", realName))
	}
	return nil
}

const sshNamespaceName = "ssh"

// InitializeSSH initializes the SSH application.
func (k *Installer) InitializeSSH(ctx context.Context, log *zap.Logger, creds *config.EtcdCredentials) error {
	if err := k.Client.CreateNamespace(ctx, sshNamespaceName); err != nil {
		log.With(zap.Error(err)).Error("failed to create namespace")
		return err
	}
	u, err := url.Parse(k.Client.RestConfig.Host)
	if err != nil {
		return err
	}
	log.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	if err := k.Client.ConnectToStore(creds, []string{net.JoinHostPort(host, "2379")}); err != nil {
		log.With(zap.Error(err)).Error("failed to connect to store")
		return err
	}
	configMapData := map[string]string{
		"key":           string(creds.KeyData),
		"cert":          string(creds.PeerCertData),
		"caCert":        string(creds.CaCertData),
		"advertiseAddr": host,
	}
	if err := k.createConfigMapAndPutData(ctx, sshNamespaceName, "etcd-credentials", configMapData); err != nil {
		log.With(zap.Error(err)).Error("failed to CreatePersistentVolumeClaim")
		return err
	}
	privateBytes, err := os.ReadFile("./server_test")
	if err != nil {
		return err
	}
	if err := k.Client.UploadSSHServerPrivKey(privateBytes); err != nil {
		return err
	}
	log.Info("uploaded ssh server private key")
	if err := k.Client.CreateServiceAccount(ctx, sshNamespaceName, "development"); err != nil {
		return err
	}
	if err := k.Client.CreateClusterRoleBinding(ctx, sshNamespaceName, "development"); err != nil {
		return err
	}

	if err := k.Client.CreateDeployment(ctx, sshNamespaceName, "ssh-relay", int32(config.ClusterConfiguration.NumberOfWorkers)); err != nil {
		return err
	}
	if err := k.Client.CreateService(ctx, sshNamespaceName, "ssh-relay"); err != nil {
		return err
	}
	if err := k.Client.CreateIngress(ctx, sshNamespaceName); err != nil {
		return err
	}
	return nil
}

// createPersistentVolume creates a shell on the specified pod.
func (k *Installer) createPersistentVolume(ctx context.Context, volumeName string) error {
	return k.Client.CreatePersistentVolume(ctx, volumeName, string(v1.ReadWriteMany))
}

// createPersistentVolumeClaim creates a shell on the specified pod.
func (k *Installer) createPersistentVolumeClaim(ctx context.Context, namespace, volumeName, storageClass string) error {
	return k.Client.CreatePersistentVolumeClaim(ctx, namespace, volumeName, storageClass)
}

// createConfigMapAndPutData creates a shell on the specified pod.
func (k *Installer) createConfigMapAndPutData(ctx context.Context, namespace, configMapName string, data map[string]string) error {
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
