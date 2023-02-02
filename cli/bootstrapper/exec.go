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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

// InstallKubernetes initializes a kubernetes cluster using the gRPC API.
func (a *bootstrapper) InstallKubernetes(ctx context.Context, kubernetesInitConfiguration []byte) (err error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	if err := a.executeWriteInitConfiguration(ctx, client, kubernetesInitConfiguration); err != nil {
		return err
	}
	_, err = a.executeKubeadm(ctx, client)
	return err
}

// JoinClusterCoordinator coordinates cluster joining for all worker nodes.
func (a *bootstrapper) JoinClusterCoordinator(ctx context.Context, joinToken *kubeadm.BootstrapTokenDiscovery) (err error) {
	a.Log.Info("coordinating kubeadm join")
	g, ctxGo := errgroup.WithContext(ctx)
	for name, addr := range a.workerIPs {
		func(nodeName, nodeIP string) {
			g.Go(func() error {
				return a.joinCluster(ctxGo, nodeName, nodeIP, joinToken)
			})
		}(name, addr)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// joinCluster connects to a node and executes kubeadm join.
func (a *bootstrapper) joinCluster(ctx context.Context, id, ip string, joinToken *kubeadm.BootstrapTokenDiscovery) (err error) {
	a.Log.Info("executing kubeadm join", zap.String("id", id), zap.String("ip", ip))
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
			"--node-name", id,
		},
	})
	if err != nil {
		return err
	}
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
				a.Log.Info("kubeadm join succeed", zap.String("id", id), zap.String("ip", ip))
				return nil
			}
			if len(data.GetLog().GetMessage()) > 0 {
				fmt.Println(data.GetLog().GetMessage())
			}
		}
	}
}

func (a *bootstrapper) executeKubeadm(ctx context.Context, client vmproto.APIClient) (output []byte, err error) {
	a.Log.Info("execute executeKubeadm")
	resp, err := client.ExecCommandStream(ctx, &vmproto.ExecCommandStreamRequest{
		Command: "/usr/bin/kubeadm",
		Args: []string{
			"init",
			"--config", "/tmp/kubeadmconf.yaml",
			"--node-name", "delegatio-master-0",
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
				a.Log.Info("kubeadm init response", zap.String("response", string(output)))
				return output, nil
			}
			if log := data.GetLog().GetMessage(); len(log) > 0 {
				fmt.Println(log)
			}
		}
	}
}

func (a *bootstrapper) executeWriteInitConfiguration(ctx context.Context, client vmproto.APIClient, initConfigKubernetes []byte) (err error) {
	a.Log.Info("write initconfig", zap.String("config", string(initConfigKubernetes)))
	_, err = client.WriteFile(ctx, &vmproto.WriteFileRequest{
		Filepath: "/tmp",
		Filename: "kubeadmconf.yaml",
		Content:  initConfigKubernetes,
	})
	return err
}
