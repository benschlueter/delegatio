/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	tokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
	kubeadmv1beta3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	tokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
)

func (a *Bootstrapper) getKubernetesConfig(ctx context.Context) (output []byte, err error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	adminConfData := resp.GetContent()
	a.adminConf = adminConfData
	return adminConfData, nil
}

func (a *Bootstrapper) getKubernetesRootCert(ctx context.Context) (output []byte, err error) {
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(a.controlPlaneIP, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ReadFile(ctx, &vmproto.ReadFileRequest{
		Filepath: "/etc/kubernetes/pki/",
		Filename: "/ca.crt",
	})
	if err != nil {
		return
	}
	return resp.GetContent(), nil
}

// getJoinToken creates a new bootstrap (join) token, which a node can use to join the cluster.
func (a *Bootstrapper) getJoinToken(ttl time.Duration, caFileContentPem []byte) (*kubeadmv1beta3.BootstrapTokenDiscovery, error) {
	a.log.Info("generating new random bootstrap token")
	rawToken, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return nil, fmt.Errorf("couldn't generate random token: %w", err)
	}
	tokenStr, err := tokenv1.NewBootstrapTokenString(rawToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	token := tokenv1.BootstrapToken{
		Token:       tokenStr,
		Description: "Bootstrap token generated by delegatio cli",
		TTL:         &metav1.Duration{Duration: ttl},
		// Usages:      kubeconstants.DefaultTokenUsages,
		// Groups:      kubeconstants.DefaultTokenGroups,
	}
	// create the token in Kubernetes
	a.log.Info("creating bootstrap token in Kubernetes")
	if err := tokenphase.CreateNewTokens(a.client, []tokenv1.BootstrapToken{token}); err != nil {
		return nil, fmt.Errorf("creating bootstrap token: %w", err)
	}
	// parse Kubernetes CA certs
	a.log.Info("Preparing join token for new node")

	caFileContent, _ := pem.Decode(caFileContentPem)
	if caFileContent == nil {
		return nil, errors.New("no PEM data found in CA cert")
	}
	caCertX509, err := x509.ParseCertificate(caFileContent.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing CA certs: %w", err)
	}
	bootstrapToken := &kubeadmv1beta3.BootstrapTokenDiscovery{
		Token:             tokenStr.String(),
		APIServerEndpoint: net.JoinHostPort(a.controlPlaneIP, "6443"),
		CACertHashes:      []string{pubkeypin.Hash(caCertX509)},
	}
	a.log.Info("Join token creation successful", zap.Any("token", bootstrapToken))
	return bootstrapToken, nil
}

// getEtcdCredentials returns the etcd credentials for the instance.
func (a *Bootstrapper) getEtcdCredentials(ctx context.Context) (*config.EtcdCredentials, error) {
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
