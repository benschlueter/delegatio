/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"errors"
	"strconv"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/spf13/afero"
	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"
	"libvirt.org/go/libvirt"
)

// LibvirtInstance is a wrapper around libvirt.
// Probably a better way to test the package is trough the test:///default connection.
type LibvirtInstance struct {
	Conn               libvirtInterface
	fs                 *afero.Afero
	Log                *zap.Logger
	masterNodeNum      int
	workerNodeNum      int
	ImagePath          string
	RegisteredNetworks []string
	RegisteredPools    []string
	RegisteredDisks    []string
}

// NewQemu creates a new Qemu Infrastructure.
func NewQemu(log *zap.Logger, imagePath string) (*LibvirtInstance, error) {
	return &LibvirtInstance{
		Log:           log,
		ImagePath:     imagePath,
		fs:            &afero.Afero{Fs: afero.NewOsFs()},
		masterNodeNum: config.ClusterConfiguration.NumberOfMasters,
		workerNodeNum: config.ClusterConfiguration.NumberOfWorkers,
	}, nil
}

// ConnectWithInfrastructureService connects to the libvirt instance.
func (l *LibvirtInstance) ConnectWithInfrastructureService(_ context.Context, url string) error {
	conn, err := libvirt.NewConnect(url)
	if err != nil {
		return err
	}
	l.Conn = &connectionWrapper{conn: conn}
	return nil
}

// InitializeInfrastructure initializes the infrastructure.
func (l *LibvirtInstance) InitializeInfrastructure(ctx context.Context) (nodes *config.NodeInformation, err error) {
	// sanity check
	if err := l.TerminateInfrastructure(); err != nil {
		return nil, err
	}
	if err := l.createStoragePool(); err != nil {
		return nil, err
	}
	if err := l.createBaseImage(ctx); err != nil {
		return nil, err
	}
	if err := l.createNetwork(); err != nil {
		return nil, err
	}
	l.Log.Info("waiting for instances to be created")
	g, _ := errgroup.WithContext(ctx)
	l.createInstances(g, l.workerNodeNum, false)
	l.createInstances(g, l.masterNodeNum, true)
	if err := g.Wait(); err != nil {
		return nil, err
	}
	l.Log.Info("waiting for instances to become ready")
	return l.waitUntilAllNodesAreReady(ctx)
}

func (l *LibvirtInstance) createInstances(g *errgroup.Group, num int, isMaster bool) {
	for i := 0; i < num; i++ {
		// the wrapper is necessary to prevent an update of the loop variable.
		// without it, it would race and have the same value all the time.
		func(id int) {
			g.Go(func() error {
				return l.createInstance(strconv.Itoa(id), isMaster)
			})
		}(i)
	}
}

// waitUntilAllNodesAreReady initializes kubernetes on the infrastructure.
func (l *LibvirtInstance) waitUntilAllNodesAreReady(ctx context.Context) (*config.NodeInformation, error) {
	g, _ := errgroup.WithContext(ctx)
	l.blockUntilInstancesReady(ctx, g, l.workerNodeNum, false)
	l.blockUntilInstancesReady(ctx, g, l.masterNodeNum, true)
	if err := g.Wait(); err != nil {
		return nil, err
	}
	l.Log.Info("all instaces are ready")
	agent, err := l.getNodeInformation(ctx)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// blockUntilInstancesReady blocks until all instances are ready.
func (l *LibvirtInstance) blockUntilInstancesReady(ctx context.Context, g *errgroup.Group, num int, isMaster bool) {
	for i := 0; i < num; i++ {
		// the wrapper is necessary to prevent an update of the loop variable.
		// without it, it would race and have the same value all the time.
		func(id int) {
			g.Go(func() error {
				return l.blockUntilInstanceReady(ctx, strconv.Itoa(id), isMaster)
			})
		}(i)
	}
}

// TerminateInfrastructure deletes all resources created by the infrastructure.
func (l *LibvirtInstance) TerminateInfrastructure() error {
	return errors.Join(l.deleteNetwork(), l.deleteDomains(), l.deletePool())
}

// TerminateConnection closes the libvirt connection.
func (l *LibvirtInstance) TerminateConnection() error {
	_, err := l.Conn.Close()
	return err
}

type libvirtInterface interface {
	ListAllNetworks(flags libvirt.ConnectListAllNetworksFlags) ([]network, error)
	ListAllStoragePools(flags libvirt.ConnectListAllStoragePoolsFlags) ([]storagePool, error)
	ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]domain, error)

	LookupDomainByName(id string) (domain, error)
	LookupStoragePoolByTargetPath(path string) (storagePool, error)

	StoragePoolDefineXML(xmlConfig string, flags libvirt.StoragePoolDefineFlags) (storagePool, error)
	NetworkCreateXML(xmlConfig string) (network, error)
	DomainCreateXML(xmlConfig string, flags libvirt.DomainCreateFlags) (domain, error)

	NewStream(flags libvirt.StreamFlags) (stream, error)
	Close() (int, error)
}

type storagePool interface {
	StorageVolCreateXML(xmlConfig string, flags libvirt.StorageVolCreateFlags) (storageVolume, error)
	ListAllStorageVolumes(flags uint32) ([]storageVolume, error)
	Build(flags libvirt.StoragePoolBuildFlags) error
	Create(flags libvirt.StoragePoolCreateFlags) error
	GetName() (string, error)
	IsActive() (bool, error)
	Destroy() error
	Delete(flags libvirt.StoragePoolDeleteFlags) error
	Undefine() error
	Free() error
}

type storageVolume interface {
	Upload(stream stream, offset, length uint64, flags libvirt.StorageVolUploadFlags) error
	Free() error
	Delete(flags libvirt.StorageVolDeleteFlags) error
}

type domain interface {
	ListAllInterfaceAddresses(src libvirt.DomainInterfaceAddressesSource) ([]libvirt.DomainInterface, error)
	GetName() (string, error)
	Free() error
	Destroy() error
}

type network interface {
	GetName() (string, error)
	Free() error
	Destroy() error
}

type stream interface {
	Abort() error
	Free() error
	Finish() error
	Send(p []byte) (int, error)
}
