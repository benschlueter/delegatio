/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"fmt"
	"net"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/agent/vm/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InstallKubernetes initializes a kubernetes cluster using the gRPC API.
func (a *Bootstrapper) InstallKubernetes(ctx context.Context, kubernetesInitConfiguration []byte) (err error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	manageClient := manageproto.NewAPIClient(conn)
	if err := a.executeWriteInitConfiguration(ctx, manageClient, kubernetesInitConfiguration); err != nil {
		return err
	}
	vmClient := vmproto.NewAPIClient(conn)
	_, err = a.executeKubeadm(ctx, vmClient)
	return err
}

func (a *Bootstrapper) executeKubeadm(ctx context.Context, client vmproto.APIClient) (output []byte, err error) {
	a.log.Info("execute executeKubeadm")
	resp, err := client.InitFirstMaster(ctx, &vmproto.InitFirstMasterRequest{
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
				a.log.Info("kubeadm init response", zap.String("response", string(output)))
				return output, nil
			}
			if log := data.GetLog().GetMessage(); len(log) > 0 {
				fmt.Println(log)
			}
		}
	}
}

func (a *Bootstrapper) executeWriteInitConfiguration(ctx context.Context, client manageproto.APIClient, initConfigKubernetes []byte) (err error) {
	a.log.Info("write initconfig", zap.String("config", string(initConfigKubernetes)))
	_, err = client.WriteFile(ctx, &manageproto.WriteFileRequest{
		Filepath: "/tmp",
		Filename: "kubeadmconf.yaml",
		Content:  initConfigKubernetes,
	})
	return err
}
