/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"fmt"

	"github.com/benschlueter/delegatio/cli/infrastructure/qemu/definitions"
	"go.uber.org/multierr"
	"libvirt.org/go/libvirt"
)

// TerminateInfrastructure deletes all resources created by the infrastructure.
func (l *LibvirtInstance) TerminateInfrastructure() error {
	var err error
	err = multierr.Append(err, l.deleteNetwork())
	err = multierr.Append(err, l.deleteDomain())
	err = multierr.Append(err, l.deletePool())
	return err
}

// TerminateConnection closes the libvirt connection.
func (l *LibvirtInstance) TerminateConnection() error {
	_, err := l.Conn.Close()
	return err
}

func (l *LibvirtInstance) deleteNetwork() error {
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

func (l *LibvirtInstance) deleteDomain() error {
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
		/* 		name, err := dom.GetName()
		   		if err != nil {
		   			return err
		   		}
		   		if name != definitions.DomainName {
		   			continue
		   		} */
		if err := dom.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (l *LibvirtInstance) deleteVolumesFromPool(pool *libvirt.StoragePool) error {
	volumes, err := pool.ListAllStorageVolumes(0)
	if err != nil {
		return err
	}
	defer func() {
		for _, volume := range volumes {
			_ = volume.Free()
		}
	}()
	for _, volume := range volumes {
		/* 		name, err := volume.GetName()
		   		if err != nil {
		   			return err
		   		}
		   		if !strings.Contains(name, definitions.BaseDiskName) && name != definitions.BaseDiskName {
		   			continue
		   		} */
		if err := volume.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL); err != nil {
			return fmt.Errorf("test %w", err)
		}
	}
	return nil
}

func (l *LibvirtInstance) deletePool() error {
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
			if err := l.deleteVolumesFromPool(&pool); err != nil {
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
