/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"errors"
	"os"

	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Bootstrapper communicates with the agent inside the control-plane VM after Kubernetes was initialized.
type Bootstrapper struct {
	log            *zap.Logger
	client         kubernetes.Interface
	loadbalancerIP string
	adminConf      []byte
	k8sConfig      []byte
}

// NewBootstrapper creates a new agent object.
func NewBootstrapper(log *zap.Logger, loadbalancerIP string, k8sConfig []byte) (*Bootstrapper, error) {
	agentLog := log.Named("bootstrapper")
	return &Bootstrapper{
		log:            agentLog,
		loadbalancerIP: loadbalancerIP,
		k8sConfig:      k8sConfig,
	}, nil
}

// BootstrapKubernetes initializes the kubernetes cluster.
func (a *Bootstrapper) BootstrapKubernetes(ctx context.Context) (*config.EtcdCredentials, error) {
	if err := a.InstallKubernetes(ctx, a.k8sConfig); err != nil {
		return nil, err
	}
	a.log.Info("kubernetes init successful")
	err := a.configureKubernetes(ctx)
	if err != nil {
		return nil, err
	}
	a.log.Info("getting etcd credentials")
	return a.getEtcdCredentials(ctx)
}

// configureKubernetes configures the kubernetes cluster.
func (a *Bootstrapper) configureKubernetes(ctx context.Context) error {
	if err := a.writeKubeconfigToDisk(ctx); err != nil {
		return err
	}
	a.log.Info("admin.conf written to disk")
	if err := a.establishClientGoConnection(); err != nil {
		return err
	}
	/* 	a.log.Info("client-go connection established")
	   	caFileContentPem, err := a.getKubernetesRootCert(ctx)
	   	if err != nil {
	   		return nil, fmt.Errorf("loading ca.crt file: %w", err)
	   	}
	   	a.log.Info("ca.crt loaded")
	   	joinToken, err := a.getJoinToken(config.DefaultTimeout, caFileContentPem)
	   	if err != nil {
	   		return nil, err
	   	}
	   	a.log.Info("join token generated") */
	return nil
}

// establishClientGoConnection configures the client-go connection.
func (a *Bootstrapper) establishClientGoConnection() error {
	val, present := os.LookupEnv("KUBECONFIG")
	if !present {
		return errors.New("KUBECONFIG environment variable not set")
	}
	config, err := clientcmd.BuildConfigFromFlags("", val)
	if err != nil {
		return err
	}
	// create the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	a.client = client
	return nil
}
