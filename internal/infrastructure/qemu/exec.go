/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/benschlueter/delegatio/client/config"
	"github.com/benschlueter/delegatio/client/vmapi/vmproto"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"libvirt.org/go/libvirt"

	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

// JoinClustergRPC joins a cluster using the gRPC API.
func (l *LibvirtInstance) JoinClustergRPC(ctx context.Context, id string, joinToken *kubeadm.BootstrapTokenDiscovery) (err error) {
	domain, err := l.Conn.LookupDomainByName(id)
	if err != nil {
		return err
	}
	iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		return
	}

	var ip string
	for _, netInterface := range iface {
		if netInterface.Name == "lo" {
			continue
		}
		for _, addr := range netInterface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
				ip = addr.Addr
			}
		}
	}
	if len(ip) == 0 {
		return fmt.Errorf("could not find ip addr")
	}
	l.Log.Info("executing kubeadm join", zap.String("id", id))

	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ExecCommandStream(ctx, &vmproto.ExecCommandStreamRequest{
		Command: "/usr/bin/kubeadm",
		Args: []string{
			"join", joinToken.APIServerEndpoint,
			"--token", joinToken.Token,
			"--discovery-token-ca-cert-hash", joinToken.CACertHashes[0],
		},
	})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, err := resp.Recv()
			if err != nil {
				return err
			}
			if len(data.GetOutput()) > 0 {
				l.Log.Info("kubeadm join succeed", zap.String("id", id))
				return nil
			}
			if len(data.GetLog().GetMessage()) > 0 {
				fmt.Println(data.GetLog().GetMessage())
			}
		}
	}
}

func (l *LibvirtInstance) executeKubeadm(ctx context.Context, client vmproto.APIClient) (output []byte, err error) {
	l.Log.Info("execute executeKubeadm")
	resp, err := client.ExecCommandStream(ctx, &vmproto.ExecCommandStreamRequest{
		Command: "/usr/bin/kubeadm",
		Args: []string{
			"init",
			"--config", "/tmp/kubeadmconf.yaml",
			"--v=9",
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
				l.Log.Info("kubeadm init response", zap.String("response", string(output)))
				return output, nil
			}
			if log := data.GetLog().GetMessage(); len(log) > 0 {
				fmt.Println(log)
			}
		}
	}
}

func (l *LibvirtInstance) executeWriteInitConfiguration(ctx context.Context, client vmproto.APIClient, initConfigKubernetes []byte) (err error) {
	l.Log.Info("write initconfig", zap.String("config", string(initConfigKubernetes)))
	_, err = client.WriteFile(ctx, &vmproto.WriteFileRequest{
		Filepath: "/tmp",
		Filename: "kubeadmconf.yaml",
		Content:  initConfigKubernetes,
	})
	return err
}

// InitializeKubernetesgRPC initializes a kubernetes cluster using the gRPC API.
func (l *LibvirtInstance) InitializeKubernetesgRPC(ctx context.Context, initConfigK8s []byte) (output []byte, err error) {
	ip, err := l.getControlPlaneIP()
	if err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	if err := l.executeWriteInitConfiguration(ctx, client, initConfigK8s); err != nil {
		return nil, err
	}
	return l.executeKubeadm(ctx, client)
}

func (l *LibvirtInstance) getKubeconfgRPC(ctx context.Context) (output []byte, err error) {
	ip, err := l.getControlPlaneIP()
	if err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ReadFile(ctx, &vmproto.ReadFileRequest{
		Filepath: "/etc/kubernetes",
		Filename: "/admin.conf",
	})
	if err != nil {
		return
	}
	return resp.GetContent(), nil
}

// WriteKubeconfigToDisk writes the kubeconfig to disk.
func (l *LibvirtInstance) WriteKubeconfigToDisk(ctx context.Context) (err error) {
	file, err := l.getKubeconfgRPC(ctx)
	if err != nil {
		return err
	}
	adminConfigFile, err := os.Create("admin.conf")
	defer func() {
		err = multierr.Append(err, adminConfigFile.Close())
	}()
	if err != nil {
		return fmt.Errorf("failed to create admin config file %v: %w", adminConfigFile.Name(), err)
	}

	if _, err := adminConfigFile.Write(file); err != nil {
		return fmt.Errorf("writing kubeadm init yaml config %v failed: %w", adminConfigFile.Name(), err)
	}
	return
}
