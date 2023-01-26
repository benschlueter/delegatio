/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"strconv"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/config/definitions"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"
	"libvirt.org/go/libvirt"
)

// libvirtInstance is a wrapper around libvirt.
type libvirtInstance struct {
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
func NewQemu(log *zap.Logger, imagePath string) (*libvirtInstance, error) {
	return &libvirtInstance{
		Log:           log,
		ImagePath:     imagePath,
		fs:            &afero.Afero{Fs: afero.NewOsFs()},
		masterNodeNum: config.ClusterConfiguration.NumberOfMasters,
		workerNodeNum: config.ClusterConfiguration.NumberOfWorkers,
	}, nil
}

// ConnectWithInfrastructureService connects to the libvirt instance.
func (l *libvirtInstance) ConnectWithInfrastructureService(ctx context.Context, url string) error {
	conn, err := libvirt.NewConnect(url)
	if err != nil {
		return err
	}
	l.Conn = &connectionWrapper{conn: conn}
	return nil
}

// InitializeInfrastructure initializes the infrastructure.
func (l *libvirtInstance) InitializeInfrastructure(ctx context.Context) (err error) {
	// sanity check
	if err := l.TerminateInfrastructure(); err != nil {
		return err
	}
	if err := l.createStoragePool(); err != nil {
		return err
	}
	if err := l.createBaseImage(ctx); err != nil {
		return err
	}
	if err := l.createNetwork(); err != nil {
		return err
	}
	l.Log.Info("waiting for instances to be created")
	g, _ := errgroup.WithContext(ctx)
	l.createInstances(g, l.workerNodeNum, false)
	l.createInstances(g, l.masterNodeNum, true)
	if err := g.Wait(); err != nil {
		return err
	}
	l.Log.Info("waiting for instances to become ready")
	return err
}

func (l *libvirtInstance) createInstances(g *errgroup.Group, num int, masters bool) {
	for i := 0; i < num; i++ {
		// the wrapper is necessary to prevent an update of the loop variable.
		// without it, it would race and have the same value all the time.
		func(id int) {
			g.Go(func() error {
				return l.CreateInstance(strconv.Itoa(id), masters)
			})
		}(i)
	}
}

// BootstrapKubernetes initializes kubernetes on the infrastructure.
func (l *libvirtInstance) BootstrapKubernetes(ctx context.Context, k8sConfig []byte) (*config.EtcdCredentials, error) {
	if _, err := l.blockUntilNetworkIsReady(ctx, definitions.DomainPrefixMaster+"0"); err != nil {
		return nil, err
	}
	l.Log.Info("network is ready")
	if err := l.blockUntilDelegatioAgentIsReady(ctx); err != nil {
		return nil, err
	}
	l.Log.Info("delegatio-agent is ready")
	agent, err := l.createAgent(ctx)
	if err != nil {
		return nil, err
	}
	l.Log.Info("agent created")
	if err := agent.InstallKubernetes(ctx, k8sConfig); err != nil {
		return nil, err
	}
	l.Log.Info("kubernetes init successful")
	joinToken, err := agent.ConfigureKubernetes(ctx)
	if err != nil {
		return nil, err
	}
	l.Log.Info("join token generated")
	if err := agent.JoinClusterCoordinator(ctx, joinToken); err != nil {
		return nil, err
	}
	return agent.GetEtcdCredentials(ctx)
}

// TerminateInfrastructure deletes all resources created by the infrastructure.
func (l *libvirtInstance) TerminateInfrastructure() error {
	var err error
	err = multierr.Append(err, l.deleteNetwork())
	err = multierr.Append(err, l.deleteDomains())
	err = multierr.Append(err, l.deletePool())
	return err
}

// TerminateConnection closes the libvirt connection.
func (l *libvirtInstance) TerminateConnection() error {
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
	Send(p []byte) (int, error)
}
