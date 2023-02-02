/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package bootstrapper

import (
	"context"
	"fmt"
	"net"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

// Bootstrapper is the interface for the bootstrapper. It is used to bootstrap a kubernetes cluster.
// It covers Kubeadm init on the control plane, as well as joining the worker nodes.
type Bootstrapper interface {
	BootstrapKubernetes(context.Context) (*config.EtcdCredentials, error)
}

// bootstrapper communicates with the agent inside the control-plane VM after Kubernetes was initialized.
type bootstrapper struct {
	Log            *zap.Logger
	client         kubernetes.Interface
	adminConf      []byte
	controlPlaneIP string
	workerIPs      map[string]string
	k8sConfig      []byte
}

// NewBootstrapper creates a new agent object.
func NewBootstrapper(log *zap.Logger, nodes *config.NodeInformation, k8sConfig []byte) (Bootstrapper, error) {
	agentLog := log.Named("bootstrapper")
	var controlPlaneIP string
	for _, ip := range nodes.Masters {
		controlPlaneIP = ip
	}
	return &bootstrapper{
		Log:            agentLog,
		controlPlaneIP: controlPlaneIP,
		workerIPs:      nodes.Workers,
		k8sConfig:      k8sConfig,
	}, nil
}

func (a *bootstrapper) BootstrapKubernetes(ctx context.Context) (*config.EtcdCredentials, error) {
	if err := a.InstallKubernetes(ctx, a.k8sConfig); err != nil {
		return nil, err
	}
	a.Log.Info("kubernetes init successful")
	joinToken, err := a.configureKubernetes(ctx)
	if err != nil {
		return nil, err
	}
	a.Log.Info("join token generated")
	if err := a.JoinClusterCoordinator(ctx, joinToken); err != nil {
		return nil, err
	}
	return a.getEtcdCredentials(ctx)
}

// configureKubernetes configures the kubernetes cluster.
func (a *bootstrapper) configureKubernetes(ctx context.Context) (*v1beta3.BootstrapTokenDiscovery, error) {
	if err := a.writeKubeconfigToDisk(ctx); err != nil {
		return nil, err
	}
	if err := a.establishClientGoConnection(); err != nil {
		return nil, err
	}
	a.Log.Info("admin.conf written to disk")
	caFileContentPem, err := a.getKubernetesRootCert(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading ca.crt file: %w", err)
	}
	joinToken, err := a.getJoinToken(config.DefaultTimeout, caFileContentPem)
	if err != nil {
		return nil, err
	}
	a.Log.Info("generated join token")
	return joinToken, nil
}

// getEtcdCredentials returns the etcd credentials for the instance.
func (a *bootstrapper) getEtcdCredentials(ctx context.Context) (*config.EtcdCredentials, error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	// Get the peer cert
	resp, err := client.ReadFile(ctx, &vmproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/etcd/",
		Filename: "ca.key",
	})
	if err != nil {
		return nil, nil
	}
	caKey := resp.Content
	// get the CA cert
	resp, err = client.ReadFile(ctx, &vmproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/etcd/",
		Filename: "ca.crt",
	})
	if err != nil {
		return nil, nil
	}
	caCert := resp.Content
	return a.generateEtcdCertificate(caCert, caKey)
}

// establishClientGoConnection configures the client-go connection.
func (a *bootstrapper) establishClientGoConnection() error {
	config, err := clientcmd.BuildConfigFromFlags("", "./admin.conf")
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
