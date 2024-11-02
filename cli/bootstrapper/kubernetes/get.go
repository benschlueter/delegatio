/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"net"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (a *Bootstrapper) getKubernetesConfig(ctx context.Context) (output []byte, err error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	a.adminConf = adminConfData
	return adminConfData, nil
}

// getEtcdCredentials returns the etcd credentials for the instance.
func (a *Bootstrapper) getEtcdCredentials(ctx context.Context) (*config.EtcdCredentials, error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := manageproto.NewAPIClient(conn)
	// Get the peer cert
	resp, err := client.ReadFile(ctx, &manageproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/etcd/",
		Filename: "ca.key",
	})
	if err != nil {
		return nil, nil
	}
	caKey := resp.Content
	// get the CA cert
	resp, err = client.ReadFile(ctx, &manageproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/etcd/",
		Filename: "ca.crt",
	})
	if err != nil {
		return nil, nil
	}
	caCert := resp.Content
	return a.generateEtcdCertificate(caCert, caKey)
}
