/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"errors"

	"libvirt.org/go/libvirt"
)

type stubLibvirtConnect struct {
	closed                   bool
	listAllNetworksErr       error
	listAllStoragePoolsErr   error
	listallDomainsErr        error
	storagePoolDefineErr     error
	storagePoolTargetPathErr error
	networkCreateErr         error
	domainCreateErr          error
	newStreamErr             error
	lookUpDomainErr          error
	networks                 []*stubNetwork
	pools                    []*stubStoragePool
	domains                  []*stubDomain
	storagePoolDefine        *stubStoragePool
	storagePoolTargetPath    *stubStoragePool
	networkCreate            *stubNetwork
	domainCreate             *stubDomain
}

func (l *stubLibvirtConnect) ListAllNetworks(flags libvirt.ConnectListAllNetworksFlags) ([]network, error) {
	var slice []network
	for _, v := range l.networks {
		if !v.destroyed {
			slice = append(slice, v)
		}
	}
	return slice, l.listAllNetworksErr
}

func (l *stubLibvirtConnect) isNetworkPresent(name string) bool {
	for _, v := range l.networks {
		if !v.destroyed && v.name == name {
			return true
		}
	}
	return false
}

func (l *stubLibvirtConnect) ListAllStoragePools(flags libvirt.ConnectListAllStoragePoolsFlags) ([]storagePool, error) {
	var slice []storagePool
	for _, v := range l.pools {
		if !v.destroyed {
			slice = append(slice, v)
		}
	}
	return slice, l.listAllStoragePoolsErr
}

func (l *stubLibvirtConnect) isPoolPresent(name string) bool {
	for _, v := range l.pools {
		if !v.destroyed && v.name == name {
			return true
		}
	}
	return false
}

func (l *stubLibvirtConnect) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]domain, error) {
	var slice []domain
	for _, v := range l.domains {
		if !v.destroyed {
			slice = append(slice, v)
		}
	}
	return slice, l.listallDomainsErr
}

func (l *stubLibvirtConnect) isDomainPresent(name string) bool {
	for _, v := range l.domains {
		if !v.destroyed && v.name == name {
			return true
		}
	}
	return false
}

func (l *stubLibvirtConnect) LookupDomainByName(id string) (domain, error) {
	for _, v := range l.domains {
		if !v.destroyed && v.name == id {
			return v, l.lookUpDomainErr
		}
	}
	return nil, errors.New("domain not found")
}

func (l *stubLibvirtConnect) LookupStoragePoolByTargetPath(path string) (storagePool, error) {
	return l.storagePoolTargetPath, l.storagePoolTargetPathErr
}

func (l *stubLibvirtConnect) StoragePoolDefineXML(xmlConfig string, flags libvirt.StoragePoolDefineFlags) (storagePool, error) {
	return l.storagePoolDefine, l.storagePoolDefineErr
}

func (l *stubLibvirtConnect) NetworkCreateXML(xmlConfig string) (network, error) {
	return l.networkCreate, l.networkCreateErr
}

func (l *stubLibvirtConnect) DomainCreateXML(xmlConfig string, flags libvirt.DomainCreateFlags) (domain, error) {
	return l.domainCreate, l.domainCreateErr
}

func (l *stubLibvirtConnect) NewStream(flags libvirt.StreamFlags) (stream, error) {
	return &stubStream{}, l.newStreamErr
}

func (l *stubLibvirtConnect) Close() (int, error) {
	l.closed = true
	return 0, nil
}

type stubNetwork struct {
	name       string
	destroyed  bool
	freeErr    error
	destroyErr error
	getNameErr error
}

func (n *stubNetwork) GetName() (string, error) {
	return n.name, n.getNameErr
}

func (n *stubNetwork) Free() error {
	return n.freeErr
}

func (n *stubNetwork) Destroy() error {
	n.destroyed = true
	return n.destroyErr
}

type stubDomain struct {
	name                         string
	netIfaces                    []libvirt.DomainInterface
	destroyed                    bool
	freeErr                      error
	destroyErr                   error
	getNameErr                   error
	listAllInterfaceAddressesErr error
}

func (d *stubDomain) ListAllInterfaceAddresses(src libvirt.DomainInterfaceAddressesSource) ([]libvirt.DomainInterface, error) {
	return d.netIfaces, d.listAllInterfaceAddressesErr
}

func (d *stubDomain) GetName() (string, error) {
	return d.name, d.getNameErr
}

func (d *stubDomain) Free() error {
	return d.freeErr
}

func (d *stubDomain) Destroy() error {
	d.destroyed = true
	return d.destroyErr
}

type stubVolume struct {
	freeErr   error
	destroyed bool
	deleteErr error
	uploadErr error
}

func (vo *stubVolume) Upload(stream stream, offset uint64, length uint64, flags libvirt.StorageVolUploadFlags) error {
	return vo.uploadErr
}

func (vo *stubVolume) Free() error {
	return vo.freeErr
}

func (vo *stubVolume) Delete(flags libvirt.StorageVolDeleteFlags) error {
	return vo.deleteErr
}

type stubStoragePool struct {
	name                 string
	destroyed            bool
	active               bool
	volumes              []*stubVolume
	StorageVolCreate     *stubVolume
	StorageVolCreateErr  error
	freeErr              error
	destroyErr           error
	isActiveErr          error
	deleteErr            error
	getNameErr           error
	undefineErr          error
	listAllStorageVolErr error
	buildErr             error
	createErr            error
}

func (s *stubStoragePool) StorageVolCreateXML(xmlConfig string, flags libvirt.StorageVolCreateFlags) (storageVolume, error) {
	return s.StorageVolCreate, s.StorageVolCreateErr
}

func (s *stubStoragePool) ListAllStorageVolumes(flags uint32) ([]storageVolume, error) {
	var slice []storageVolume
	for _, v := range s.volumes {
		if !v.destroyed {
			slice = append(slice, v)
		}
	}
	return slice, s.listAllStorageVolErr
}

func (s *stubStoragePool) Build(flags libvirt.StoragePoolBuildFlags) error {
	return s.buildErr
}

func (s *stubStoragePool) Create(flags libvirt.StoragePoolCreateFlags) error {
	if s.createErr == nil {
		s.active = true
	}
	return s.createErr
}

func (s *stubStoragePool) GetName() (string, error) {
	return s.name, s.getNameErr
}

func (s *stubStoragePool) IsActive() (bool, error) {
	return s.active, s.isActiveErr
}

func (s *stubStoragePool) Destroy() error {
	s.destroyed = true
	return s.destroyErr
}

func (s *stubStoragePool) Delete(flags libvirt.StoragePoolDeleteFlags) error {
	return s.deleteErr
}

func (s *stubStoragePool) Undefine() error {
	return s.undefineErr
}

func (s *stubStoragePool) Free() error {
	return s.freeErr
}

type stubStream struct {
	abortErr error
	freeErr  error
	sendErr  error
}

func (s *stubStream) Abort() error {
	return s.abortErr
}

func (s *stubStream) Free() error {
	return s.freeErr
}

func (s *stubStream) Send(p []byte) (int, error) {
	return len(p), s.sendErr
}
