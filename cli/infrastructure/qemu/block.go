/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"time"

	"github.com/benschlueter/delegatio/agent/manageapi/manageproto"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/config/definitions"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"libvirt.org/go/libvirt"
)

func (l *LibvirtInstance) blockUntilInstanceReady(ctx context.Context, number string, controlPlane bool) error {
	var prefix string
	if controlPlane {
		prefix = definitions.DomainPrefixMaster
	} else {
		prefix = definitions.DomainPrefixWorker
	}
	nodeName := prefix + number
	l.Log.Debug("block until node is ready", zap.String("node", nodeName))
	if _, err := l.blockUntilNetworkIsReady(ctx, nodeName); err != nil {
		return err
	}
	l.Log.Debug("block until delegatio agent is ready", zap.String("node", nodeName))
	if err := l.blockUntilDelegatioAgentIsReady(ctx, nodeName); err != nil {
		return err
	}
	l.Log.Debug("node is ready", zap.String("node", nodeName))
	return nil
}

func (l *LibvirtInstance) blockUntilNetworkIsReady(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()
	domain, err := l.Conn.LookupDomainByName(id)
	if err != nil {
		return "", err
	}
	defer func() { _ = domain.Free() }()
	t := time.NewTicker(100 * time.Millisecond)
	defer func() {
		t.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			l.Log.Info("context cancel during waiting for vm init")
			return "", ctx.Err()
		case <-t.C:
			iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
			if err != nil {
				return "", err
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
			if len(ip) > 0 {
				return ip, nil
			}
		}
	}
}

func (l *LibvirtInstance) blockUntilDelegatioAgentIsReady(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 30000*time.Second)
	defer cancel()
	domain, err := l.Conn.LookupDomainByName(id)
	if err != nil {
		return err
	}
	defer func() { _ = domain.Free() }()
	iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		return err
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
		return fmt.Errorf("could not get ip addr of VM %s", definitions.DomainPrefixMaster+"0")
	}
	tlsconfig, err := config.GenerateTLSConfigClient()
	if err != nil {
		return err
	}
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(credentials.NewTLS(tlsconfig)))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := manageproto.NewAPIClient(conn)

	t := time.NewTicker(100 * time.Millisecond)
	defer func() {
		t.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			l.Log.Info("context cancel during waiting for vm init")
			return ctx.Err()
		case <-t.C:
			_, err := client.ExecCommand(ctx, &manageproto.ExecCommandRequest{
				Command: "hostnamectl",
				Args:    []string{"set-hostname", id},
			})
			if err == nil {
				return nil
			}
			// l.Log.Error("failed to set hostname", zap.Error(err))
		}
	}
}
