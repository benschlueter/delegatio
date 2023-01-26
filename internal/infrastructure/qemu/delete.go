/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"strings"

	"github.com/benschlueter/delegatio/internal/config/definitions"
	"libvirt.org/go/libvirt"
)

func (l *libvirtInstance) deleteNetwork() error {
	nets, err := l.Conn.ListAllNetworks(libvirt.CONNECT_LIST_NETWORKS_ACTIVE)
	if err != nil {
		return err
	}
	defer func() {
		for _, net := range nets {
			_ = net.Free()
		}
	}()
	for _, net := range nets {
		name, err := net.GetName()
		if err != nil {
			return err
		}
		if name != definitions.NetworkName {
			continue
		}
		if err := net.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *libvirtInstance) deleteDomains() error {
	doms, err := l.Conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		return err
	}
	defer func() {
		for _, dom := range doms {
			_ = dom.Free()
		}
	}()
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			return err
		}
		if !strings.HasPrefix(name, definitions.DomainPrefixMaster) && !strings.HasPrefix(name, definitions.DomainPrefixWorker) {
			continue
		}
		if err := dom.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *libvirtInstance) deletePool() error {
	pools, err := l.Conn.ListAllStoragePools(libvirt.CONNECT_LIST_STORAGE_POOLS_DIR)
	if err != nil {
		return err
	}
	defer func() {
		for _, pool := range pools {
			_ = pool.Free()
		}
	}()
	for _, pool := range pools {
		name, err := pool.GetName()
		if err != nil {
			return err
		}
		if name != definitions.DiskPoolName {
			continue
		}
		active, err := pool.IsActive()
		if err != nil {
			return err
		}
		if active {
			if err := deleteVolumesFromPool(pool); err != nil {
				return err
			}
			if err := pool.Destroy(); err != nil {
				return err
			}
			if err := pool.Delete(libvirt.STORAGE_POOL_DELETE_NORMAL); err != nil {
				return err
			}
		}
		// If something fails and the Pool becomes inactive, we cannot delete/destroy it anymore.
		// We have to undefine it in this case
		if err := pool.Undefine(); err != nil {
			return err
		}
	}
	return nil
}
