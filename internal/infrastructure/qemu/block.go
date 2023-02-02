/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/benschlueter/delegatio/agent/vmapi/vmproto"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/config/definitions"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"libvirt.org/go/libvirt"
)

func (l *libvirtInstance) blockUntilInstanceReady(ctx context.Context, number string, controlPlane bool) error {
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
	if err := l.blockUntilDelegatioAgentIsReady(ctx, nodeName); err != nil {
		return err
	}
	l.Log.Debug("node is ready", zap.String("node", nodeName))
	return nil
}

func (l *libvirtInstance) blockUntilNetworkIsReady(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	domain, err := l.Conn.LookupDomainByName(id)
	if err != nil {
		return "", err
	}
	defer func() { _ = domain.Free() }()
	for {
		select {
		case <-ctx.Done():
			l.Log.Info("context cancel during waiting for vm init")
			return "", ctx.Err()
		default:
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

func (l *libvirtInstance) blockUntilDelegatioAgentIsReady(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
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
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	for {
		select {
		case <-ctx.Done():
			l.Log.Info("context cancel during waiting for vm init")
			return ctx.Err()
		default:
			_, err := client.ExecCommand(ctx, &vmproto.ExecCommandRequest{
				Command: "whoami",
			})
			if err == nil {
				return nil
			}
		}
	}
}
