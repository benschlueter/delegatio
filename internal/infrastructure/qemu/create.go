/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"fmt"
	"path"

	deepcopy "github.com/barkimedes/go-deepcopy"
	"github.com/benschlueter/delegatio/internal/config/definitions"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

// CreateInstance creates a new instance. The instance consists of a boot image and a domain.
func (l *LibvirtInstance) CreateInstance(id string) (err error) {
	l.Log.Debug("creating instance", zap.String("id", id))
	if err := l.createBootImage("delegatio-" + id); err != nil {
		return err
	}
	if err := l.createDomain("delegatio-" + id); err != nil {
		return err
	}
	return nil
}

func (l *LibvirtInstance) createStoragePool() error {
	// Might be needed for renaming in the future
	poolXMLCopy := definitions.PoolXMLConfig
	poolXMLString, err := poolXMLCopy.Marshal()
	if err != nil {
		return err
	}
	l.Log.Info("creating storage pool")
	poolObject, err := l.Conn.StoragePoolDefineXML(poolXMLString, libvirt.STORAGE_POOL_DEFINE_VALIDATE)
	if err != nil {
		return fmt.Errorf("error defining libvirt storage pool: %s", err)
	}
	defer func() { _ = poolObject.Free() }()
	if err := poolObject.Build(libvirt.STORAGE_POOL_BUILD_NEW); err != nil {
		return fmt.Errorf("error building libvirt storage pool: %s", err)
	}
	if err := poolObject.Create(libvirt.STORAGE_POOL_CREATE_NORMAL); err != nil {
		return fmt.Errorf("error creating libvirt storage pool: %s", err)
	}
	l.RegisteredPools = append(l.RegisteredPools, poolXMLCopy.Name)
	return nil
}

func (l *LibvirtInstance) createBaseImage(ctx context.Context) error {
	volumeBaseXMLString, err := definitions.VolumeBaseXMLConfig.Marshal()
	if err != nil {
		return err
	}
	storagePool, err := l.Conn.LookupStoragePoolByTargetPath(definitions.LibvirtStoragePoolPath)
	if err != nil {
		return err
	}
	defer func() { _ = storagePool.Free() }()
	l.Log.Info("creating base storage image")
	volumeBaseObject, err := storagePool.StorageVolCreateXML(volumeBaseXMLString, 0)
	if err != nil {
		return fmt.Errorf("error creating libvirt storage volume 'base': %s", err)
	}
	defer func() { _ = volumeBaseObject.Free() }()
	l.RegisteredDisks = append(l.RegisteredDisks, definitions.VolumeBaseXMLConfig.Name)

	l.Log.Info("uploading baseimage to libvirt storage pool")
	return l.uploadBaseImage(ctx, volumeBaseObject)
}

func (l *LibvirtInstance) createBootImage(id string) error {
	volumeBootXMLCopy := definitions.VolumeBootXMLConfig
	volumeBootXMLCopy.Name = id
	volumeBootXMLCopy.Target.Path = path.Join(definitions.LibvirtStoragePoolPath, id)

	volumeBootXMLString, err := volumeBootXMLCopy.Marshal()
	if err != nil {
		return err
	}
	storagePool, err := l.Conn.LookupStoragePoolByTargetPath(definitions.LibvirtStoragePoolPath)
	if err != nil {
		return err
	}
	defer func() { _ = storagePool.Free() }()
	l.Log.Info("creating storage volume 'boot'", zap.String("id", id))
	bootVol, err := storagePool.StorageVolCreateXML(volumeBootXMLString, 0)
	if err != nil {
		return fmt.Errorf("error creating libvirt storage volume 'boot': %s", err)
	}
	defer func() { _ = bootVol.Free() }()
	l.RegisteredDisks = append(l.RegisteredDisks, volumeBootXMLCopy.Name)
	return nil
}

func (l *LibvirtInstance) createNetwork() error {
	networkXMLString, err := definitions.NetworkXMLConfig.Marshal()
	if err != nil {
		return err
	}
	l.Log.Info("creating network")
	network, err := l.Conn.NetworkCreateXML(networkXMLString)
	if err != nil {
		return err
	}
	defer func() { _ = network.Free() }()
	return nil
}

func (l *LibvirtInstance) createDomain(id string) error {
	domainCpyIface, err := deepcopy.Anything(&definitions.DomainXMLConfig)
	if err != nil {
		return err
	}
	domainCpy := domainCpyIface.(*libvirtxml.Domain)
	domainCpy.Name = id
	domainCpy.Devices.Disks[0].Source.Volume.Volume = id
	/* 	domainCpy.Devices.Serials[0].Log = &libvirtxml.DomainChardevLog{
	   		File: path.Join("/tmp", id),
	   	}
	*/
	domainXMLString, err := domainCpy.Marshal()
	if err != nil {
		return err
	}
	l.Log.Info("creating domain", zap.String("id", id))
	domain, err := l.Conn.DomainCreateXML(domainXMLString, libvirt.DOMAIN_NONE)
	if err != nil {
		return fmt.Errorf("error creating libvirt domain: %s", err)
	}
	defer func() { _ = domain.Free() }()
	return nil
}
