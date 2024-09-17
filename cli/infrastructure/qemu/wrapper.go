/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"errors"

	"libvirt.org/go/libvirt"
)

// The connection wrapper is needed, because libvirt does not return interfaces in their functions.
// This is a problem, because we need to mock the libvirt functions in our tests.

type connectionWrapper struct {
	conn *libvirt.Connect
}

func (l *connectionWrapper) ListAllNetworks(flags libvirt.ConnectListAllNetworksFlags) ([]network, error) {
	noPtrNets, err := l.conn.ListAllNetworks(flags)
	if err != nil {
		return nil, err
	}
	var nets []network
	for i := 0; i < len(noPtrNets); i++ {
		nets = append(nets, &noPtrNets[i])
	}
	return nets, nil
}

func (l *connectionWrapper) ListAllStoragePools(flags libvirt.ConnectListAllStoragePoolsFlags) ([]storagePool, error) {
	noPtrPools, err := l.conn.ListAllStoragePools(flags)
	if err != nil {
		return nil, err
	}
	var pools []storagePool
	for i := 0; i < len(noPtrPools); i++ {
		pools = append(pools, &poolWrapper{pool: &noPtrPools[i]})
	}
	return pools, err
}

func (l *connectionWrapper) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]domain, error) {
	noPtrDomains, err := l.conn.ListAllDomains(flags)
	if err != nil {
		return nil, err
	}
	var domains []domain
	for i := 0; i < len(noPtrDomains); i++ {
		domains = append(domains, &noPtrDomains[i])
	}
	return domains, nil
}

func (l *connectionWrapper) LookupDomainByName(id string) (domain, error) {
	return l.conn.LookupDomainByName(id)
}

func (l *connectionWrapper) LookupStoragePoolByTargetPath(path string) (storagePool, error) {
	pool, err := l.conn.LookupStoragePoolByTargetPath(path)
	if err != nil {
		return nil, err
	}
	return &poolWrapper{pool: pool}, nil
}

func (l *connectionWrapper) StoragePoolDefineXML(xmlConfig string, flags libvirt.StoragePoolDefineFlags) (storagePool, error) {
	pool, err := l.conn.StoragePoolDefineXML(xmlConfig, flags)
	if err != nil {
		return nil, err
	}
	return &poolWrapper{pool: pool}, nil
}

func (l *connectionWrapper) NetworkCreateXML(xmlConfig string) (network, error) {
	return l.conn.NetworkCreateXML(xmlConfig)
}

func (l *connectionWrapper) DomainCreateXML(xmlConfig string, flags libvirt.DomainCreateFlags) (domain, error) {
	return l.conn.DomainCreateXML(xmlConfig, flags)
}

func (l *connectionWrapper) NewStream(flags libvirt.StreamFlags) (stream, error) {
	return l.conn.NewStream(flags)
}

func (l *connectionWrapper) Close() (int, error) {
	return l.conn.Close()
}

type poolWrapper struct {
	pool *libvirt.StoragePool
}

func (p *poolWrapper) StorageVolCreateXML(xmlConfig string, flags libvirt.StorageVolCreateFlags) (storageVolume, error) {
	stVolume, err := p.pool.StorageVolCreateXML(xmlConfig, flags)
	if err != nil {
		return nil, err
	}
	return &storageVolumeWrapper{volume: stVolume}, nil
}

func (p *poolWrapper) ListAllStorageVolumes(flags uint32) ([]storageVolume, error) {
	var slice []storageVolume
	vols, err := p.pool.ListAllStorageVolumes(flags)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(vols); i++ {
		slice = append(slice, &storageVolumeWrapper{volume: &vols[i]})
	}

	return slice, nil
}

func (p *poolWrapper) Build(flags libvirt.StoragePoolBuildFlags) error {
	return p.pool.Build(flags)
}

func (p *poolWrapper) Create(flags libvirt.StoragePoolCreateFlags) error {
	return p.pool.Create(flags)
}

func (p *poolWrapper) GetName() (string, error) {
	return p.pool.GetName()
}

func (p *poolWrapper) IsActive() (bool, error) {
	return p.pool.IsActive()
}

func (p *poolWrapper) Destroy() error {
	return p.pool.Destroy()
}

func (p *poolWrapper) Delete(flags libvirt.StoragePoolDeleteFlags) error {
	return p.pool.Delete(flags)
}

func (p *poolWrapper) Undefine() error {
	return p.pool.Undefine()
}

func (p *poolWrapper) Free() error {
	return p.pool.Free()
}

type storageVolumeWrapper struct {
	volume *libvirt.StorageVol
}

func (v *storageVolumeWrapper) Upload(stream stream, offset, length uint64, flags libvirt.StorageVolUploadFlags) error {
	libvirtStream, ok := stream.(*libvirt.Stream)
	if !ok {
		return errors.New("stream is not a libvirt stream")
	}
	return v.volume.Upload(libvirtStream, offset, length, flags)
}

func (v *storageVolumeWrapper) Free() error {
	return v.volume.Free()
}

func (v *storageVolumeWrapper) Delete(flags libvirt.StorageVolDeleteFlags) error {
	return v.volume.Delete(flags)
}
