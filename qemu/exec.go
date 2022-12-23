package qemu

import (
	"context"
	"fmt"
	"net"

	"github.com/benschlueter/delegatio/core/config"
	"github.com/benschlueter/delegatio/core/vmapi/vmproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"libvirt.org/go/libvirt"

	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

func (l *LibvirtInstance) JoinClustergRPC(ctx context.Context, id string, joinToken *kubeadm.BootstrapTokenDiscovery) (err error) {
	domain, err := l.conn.LookupDomainByName(id)
	if err != nil {
		return err
	}
	iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
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
	l.log.Info("executing kubeadm join", zap.String("id", id))

	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	_, err = client.ExecCommand(ctx, &vmproto.ExecCommandRequest{
		Command: "/usr/bin/kubeadm",
		Args:    []string{"join", joinToken.APIServerEndpoint, "--token", joinToken.Token, "--discovery-token-ca-cert-hash", joinToken.CACertHashes[0]},
	})
	return err
}

func (l *LibvirtInstance) InitializeKubernetesgRPC(ctx context.Context) (output string, err error) {
	domain, err := l.conn.LookupDomainByName("delegatio-0")
	if err != nil {
		return
	}
	defer func() { _ = domain.Free() }()
	iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
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
		return "", fmt.Errorf("could not find ip addr to domain")
	}
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ExecCommand(ctx, &vmproto.ExecCommandRequest{
		Command: "/usr/bin/kubeadm",
		Args:    []string{"init"},
	})
	if err != nil {
		return
	}

	l.log.Info("kubeadm init response", zap.String("response", string(resp.Output)))
	return string(resp.Output), err
}
