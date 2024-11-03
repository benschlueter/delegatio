/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/agent/vm/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
)

// VMAPI is the interface for the VM API.
type VMAPI interface {
	InstallKubernetes(context.Context, string, []byte) error
	GetKubernetesConfig(context.Context, string) ([]byte, error)
	GetEtcdCredentials(context.Context, string) ([]byte, []byte, error)
	GetJoinData(ctx context.Context) (*vmproto.GetJoinDataKubeResponse, error)
}

// APIExternal is the external API.
type APIExternal struct {
	logger    *zap.Logger
	dialer    Dialer
	tlsConfig *tls.Config
}

// NewExternal creates a new external API.
func NewExternal(logger *zap.Logger, dialer Dialer) (*APIExternal, error) {
	tlsconfig, err := config.GenerateTLSConfigClient()
	if err != nil {
		return nil, err
	}

	return &APIExternal{
		logger:    logger,
		dialer:    dialer,
		tlsConfig: tlsconfig,
	}, nil
}

// can only be used once the node has its initial name, i.e. after the first call to the API.
func (a *APIExternal) dialWithLoadBalancer() (*grpc.ClientConn, error) {
	// Use the round_robin load balancing policy
	return grpc.NewClient("static"+":///",
		grpc.WithTransportCredentials(credentials.NewTLS(a.tlsConfig)),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
}

// can only be used once the node has its initial name, i.e. after the first call to the API.
func (a *APIExternal) dialFirstMaster() (*grpc.ClientConn, error) {
	// Use the round_robin load balancing policy
	return grpc.NewClient("delegatio-master-0:9000",
		grpc.WithTransportCredentials(credentials.NewTLS(a.tlsConfig)),
	)
}

// staticResolver is a custom resolver that returns a static list of target addresses.
type staticResolver struct {
	targets []string
}

func (r *staticResolver) Scheme() string {
	return "static"
}

func (r *staticResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *staticResolver) Close() {}

func (r *staticResolver) Build(_ resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	addrs := make([]resolver.Address, len(r.targets))
	for i, t := range r.targets {
		addrs[i] = resolver.Address{
			Addr:       net.JoinHostPort(t, "9000"),
			ServerName: t,
		}
	}
	err := cc.UpdateState(resolver.State{Addresses: addrs})
	return r, err
}

func init() {
	// Define the list of target addresses
	targets := []string{
		"delegatio-master-0",
		"delegatio-master-1",
		"delegatio-master-2",
		"delegatio-master-3",
		"delegatio-master-4",
		"delegatio-master-5",
	}
	// Create a resolver that returns the list of target addresses
	r := &staticResolver{
		targets: targets,
	}

	// Register the resolver
	resolver.Register(r)
}

// Dialer is the dial interface. Necessary to stub network connections for local testing
// with bufconns.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// InstallKubernetes initializes a kubernetes cluster using the gRPC API.
func (a *APIExternal) InstallKubernetes(ctx context.Context, _ string, kubernetesInitConfiguration []byte) (err error) {
	conn, err := a.dialFirstMaster()
	if err != nil {
		a.logger.Error("dial", zap.Error(err))
		return err
	}
	defer conn.Close()
	manageClient := manageproto.NewAPIClient(conn)
	if err := a.executeWriteInitConfiguration(ctx, manageClient, kubernetesInitConfiguration); err != nil {
		a.logger.Error("write initconfig", zap.Error(err))
		return err
	}
	vmClient := vmproto.NewAPIClient(conn)
	_, err = a.executeKubeadm(ctx, vmClient)
	return err
}

func (a *APIExternal) executeKubeadm(ctx context.Context, client vmproto.APIClient) (output []byte, err error) {
	a.logger.Info("execute executeKubeadm")
	resp, err := client.InitFirstMaster(ctx, &vmproto.InitFirstMasterRequest{
		Command: "/usr/bin/kubeadm",
		Args: []string{
			"init",
			"--config", "/tmp/kubeadmconf.yaml",
			"--v=1",
			"--skip-certificate-key-print",
		},
	})
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			data, err := resp.Recv()
			if err != nil {
				return nil, err
			}
			if output := data.GetOutput(); len(output) > 0 {
				a.logger.Info("kubeadm init response", zap.String("response", string(output)))
				return output, nil
			}
			if log := data.GetLog().GetMessage(); len(log) > 0 {
				fmt.Println(log)
			}
		}
	}
}

// GetKubernetesConfig returns the kubernetes config for the instance.
func (a *APIExternal) GetKubernetesConfig(ctx context.Context, _ string) (output []byte, err error) {
	conn, err := a.dialFirstMaster()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := manageproto.NewAPIClient(conn)
	resp, err := client.ReadFile(ctx, &manageproto.ReadFileRequest{
		Filepath: "/etc/kubernetes",
		Filename: "/admin.conf",
	})
	if err != nil {
		return
	}
	adminConfData := resp.GetContent()
	return adminConfData, nil
}

// GetEtcdCredentials returns the etcd credentials for the instance.
func (a *APIExternal) GetEtcdCredentials(ctx context.Context, _ string) ([]byte, []byte, error) {
	conn, err := a.dialFirstMaster()
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()
	client := manageproto.NewAPIClient(conn)
	// Get the peer cert
	resp, err := client.ReadFile(ctx, &manageproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/etcd/",
		Filename: "ca.key",
	})
	if err != nil {
		return nil, nil, err
	}
	caKey := resp.Content
	// get the CA cert
	resp, err = client.ReadFile(ctx, &manageproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/etcd/",
		Filename: "ca.crt",
	})
	if err != nil {
		return nil, nil, err
	}
	caCert := resp.Content
	return caCert, caKey, nil
}

// GetJoinData returns the join data for the instance.
func (a *APIExternal) GetJoinData(ctx context.Context) (*vmproto.GetJoinDataKubeResponse, error) {
	conn, err := a.dialWithLoadBalancer()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	a.logger.Info("grpc call to master")
	return client.GetJoinDataKube(ctx, &vmproto.GetJoinDataKubeRequest{})
}

func (a *APIExternal) executeWriteInitConfiguration(ctx context.Context, client manageproto.APIClient, initConfigKubernetes []byte) (err error) {
	a.logger.Info("write initconfig", zap.String("config", string(initConfigKubernetes)))
	_, err = client.WriteFile(ctx, &manageproto.WriteFileRequest{
		Filepath: "/tmp",
		Filename: "kubeadmconf.yaml",
		Content:  initConfigKubernetes,
	})
	return err
}
