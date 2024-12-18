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
)

// Installer is the interface for the installer. It is used to install all the kubernetes applications.
type Installer interface {
	InstallKubernetesApplications(context.Context, *config.EtcdCredentials, *config.UserConfiguration) error
}

// installer is the struct used to access kubernetes helpers.
type installer struct {
	client  *k8sapi.Client
	logger  *zap.Logger
	sshData map[string]string
}

// NewInstaller returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewInstaller(logger *zap.Logger) (Installer, error) {
	// use the current context in kubeconfig
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	return &installer{
		client: client,
		logger: logger.Named("installer"),
	}, nil
}

// InstallKubernetesApplications installs all the kubernetes applications.
func (k *installer) InstallKubernetesApplications(ctx context.Context, creds *config.EtcdCredentials, config *config.UserConfiguration) error {
	if err := k.connectToEtcd(ctx, creds); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to connect to etcd")
		return err
	}
	if err := k.installCilium(ctx); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to install helm charts")
		return err
	}
	if err := k.initalizeChallenges(ctx, config); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to deploy challenges")
		return err
	}
	if err := k.initializeSSH(ctx); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to deploy ssh server")
		return err
	}
	if err := k.initializeGrader(ctx); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to deploy grader")
		return err
	}
	return nil
}

func (k *installer) connectToEtcd(_ context.Context, creds *config.EtcdCredentials) error {
	u, err := url.Parse(k.client.RestConfig.Host)
	if err != nil {
		return err
	}
	// get direct loadbalancer IP and place here.
	k.logger.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	etcdEndpoint := net.JoinHostPort(host, "2379")
	k.logger.Info("etcd endpoint", zap.String("etcd", etcdEndpoint))
	if err := k.client.ConnectToStoreExternal(creds, []string{etcdEndpoint}); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to connect to store")
		return err
	}
	k.sshData = map[string]string{
		"key":           string(creds.KeyData),
		"cert":          string(creds.PeerCertData),
		"caCert":        string(creds.CaCertData),
		"advertiseAddr": host,
	}
	return nil
}

// installCilium installs cilium in the cluster.
func (k *installer) installCilium(ctx context.Context) error {
	u, err := url.Parse(k.client.RestConfig.Host)
	if err != nil {
		return err
	}
	k.logger.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	vals := map[string]interface{}{
		"kubeProxyReplacement": "true",
		"k8sServicePort":       "6443",
		"k8sServiceHost":       host,
		/* 		"prometheus.enabled":          "true",
		   		"operator.prometheus.enabled": true, */
	}
	helmInstaller := helm.NewHelmInstaller(k.logger, "cilium", "cilium", config.CiliumPath, config.Cilium256Hash, vals)
	return helmInstaller.Install(ctx)
}

// installTetragon installs tetragon in the cluster.
/* func (k *installer) installTetragon(ctx context.Context) error {
	helmInstaller := helm.NewHelmInstaller(k.logger, "tetragon", "kube-system", config.TetratePath, config.Tetragon256Hash, nil)
	return helmInstaller.Install(ctx)
} */

// initalizeChallenges creates the namespaces and persistent volumes for the challenges. It also adds the users to etcd.
func (k *installer) initalizeChallenges(ctx context.Context, userConfig *config.UserConfiguration) error {
	if err := k.client.CreateStorageClass(ctx, "nfs", "Retain"); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to CreateStorageClass")
		return err
	}
	if err := k.client.CreateNamespace(ctx, config.UserNamespace); err != nil {
		return err
	}
	stWrapper := storewrapper.StoreWrapper{Store: k.client.SharedStore}

	for namespace := range userConfig.Containers {
		if err := stWrapper.PutChallengeData(namespace, nil); err != nil {
			return err
		}
		k.logger.Info("added challenge to store", zap.String("challenge", namespace))
	}

	for uuid, userData := range userConfig.UUIDToUser {
		if err := stWrapper.PutDataIdxByUUID(uuid, userData); err != nil {
			return err
		}
		k.logger.Info("added user to store", zap.String("uuid", uuid), zap.Any("userinfo", userData))
	}
	for pubkey, userData := range userConfig.PubKeyToUser {
		if err := stWrapper.PutDataIdxByPubKey(pubkey, userData); err != nil {
			return err
		}
		k.logger.Info("added user to store", zap.String("pubkey", pubkey), zap.Any("userinfo", userData))
	}
	return nil
}

// initializeSSH initializes the SSH application.
func (k *installer) initializeSSH(ctx context.Context) error {
	if err := k.client.CreateNamespace(ctx, config.SSHNamespaceName); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to create namespace")
		return err
	}
	if err := k.createConfigMapAndPutData(ctx, config.SSHNamespaceName, "etcd-credentials", k.sshData); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to createConfigMapAndPutData")
		return err
	}
	privateBytes, err := os.ReadFile("./server_test")
	if err != nil {
		return err
	}
	if err := k.client.UploadSSHServerPrivKey(privateBytes); err != nil {
		return err
	}
	k.logger.Info("uploaded ssh server private key")
	if err := k.client.CreateServiceAccount(ctx, config.SSHNamespaceName, config.SSHServiceAccountName); err != nil {
		return err
	}
	if err := k.client.CreateClusterRoleBinding(ctx, config.SSHNamespaceName, config.SSHServiceAccountName); err != nil {
		return err
	}

	if err := k.client.CreateSSHDeployment(ctx, config.SSHNamespaceName, "ssh-relay", int32(config.ClusterConfiguration.NumberOfWorkers)); err != nil {
		return err
	}
	if err := k.client.CreateServiceLoadBalancer(ctx, config.SSHNamespaceName, "ssh-relay", config.SSHPort); err != nil {
		return err
	}
	if err := k.client.CreateIngress(ctx, config.SSHNamespaceName); err != nil {
		return err
	}
	k.logger.Info("init ssh success")
	return nil
}

// initializeGrader initializes the grader application.
func (k *installer) initializeGrader(ctx context.Context) error {
	if err := k.client.CreateNamespace(ctx, config.GraderNamespaceName); err != nil {
		k.logger.With(zap.Error(err)).Error("check namespace status")
		return err
	}
	k.logger.Info("create namespace", zap.String("namespace", config.GraderNamespaceName))
	if err := k.createConfigMapAndPutData(ctx, config.GraderNamespaceName, "etcd-credentials", k.sshData); err != nil {
		k.logger.With(zap.Error(err)).Error("failed to createConfigMapAndPutData")
		return err
	}
	if err := k.client.CreateServiceAccount(ctx, config.GraderNamespaceName, config.GraderServiceAccountName); err != nil {
		return err
	}
	if err := k.client.CreateClusterRoleBinding(ctx, config.GraderNamespaceName, config.GraderServiceAccountName); err != nil {
		return err
	}
	if err := k.client.CreateGraderDeployment(ctx, config.GraderNamespaceName, "grader", int32(config.ClusterConfiguration.NumberOfWorkers)); err != nil {
		return err
	}
	if err := k.client.CreateServiceClusterIP(ctx, config.GraderNamespaceName, "grader", config.GradeAPIport); err != nil {
		return err
	}
	// Not needed as long as we run on-prem
	// Probably not needed at all? Since we access the gracer tthrough the ClusterServiceName?
	/* 	if err := k.client.CreateIngress(ctx, graderNamespaceName); err != nil {
		return err
	} */
	k.logger.Info("init grader success")
	return nil
}

// createConfigMapAndPutData creates a configMaps and initializes it with the given data.
func (k *installer) createConfigMapAndPutData(ctx context.Context, namespace, configMapName string, data map[string]string) error {
	if err := k.client.CreateConfigMap(ctx, namespace, configMapName); err != nil {
		return err
	}
	for key, value := range data {
		if err := k.client.AddDataToConfigMap(ctx, namespace, configMapName, key, value); err != nil {
			return err
		}
	}
	return nil
}
