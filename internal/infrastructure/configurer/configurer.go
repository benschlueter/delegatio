/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package configurer

import (
	"context"
	"net"
	"time"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

// Configurer communicates with the agent inside the control-plane VM after Kubernetes was initialized.
type Configurer struct {
	Log            *zap.Logger
	client         kubernetes.Interface
	adminConf      []byte
	controlPlaneIP string
	workerIPs      map[string]string
}

// NewConfigurer creates a new agent object.
func NewConfigurer(log *zap.Logger, controlPlaneIP string, workerIPs map[string]string) (*Configurer, error) {
	agentLog := log.Named("kube")
	return &Configurer{
		Log:            agentLog,
		controlPlaneIP: controlPlaneIP,
		workerIPs:      workerIPs,
	}, nil
}

// ConfigureKubernetes configures the kubernetes cluster.
func (a *Configurer) ConfigureKubernetes(ctx context.Context) (*v1beta3.BootstrapTokenDiscovery, error) {
	if err := a.writeKubeconfigToDisk(ctx); err != nil {
		return nil, err
	}
	if err := a.establishClientGoConnection(); err != nil {
		return nil, err
	}
	a.Log.Info("admin.conf written to disk")
	joinToken, err := a.getJoinToken(ctx, 2*time.Minute)
	if err != nil {
		return nil, err
	}
	a.Log.Info("generated join token")
	return joinToken, nil
}

// GetEtcdCredentials returns the etcd credentials for the instance.
func (a *Configurer) GetEtcdCredentials(ctx context.Context) (*config.EtcdCredentials, error) {
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
func (a *Configurer) establishClientGoConnection() error {
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
