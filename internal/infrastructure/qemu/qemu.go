/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"strconv"
	"sync"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/infrastructure/configurer"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"
	"libvirt.org/go/libvirt"
)

// LibvirtInstance is a wrapper around libvirt.
type LibvirtInstance struct {
	ConnMux            sync.Mutex
	Conn               *libvirt.Connect
	Log                *zap.Logger
	ImagePath          string
	RegisteredNetworks []string
	RegisteredPools    []string
	RegisteredDisks    []string
	vmAgent            *configurer.Configurer
}

// NewQemu creates a new Qemu Infrastructure.
func NewQemu(log *zap.Logger, imagePath string) (*LibvirtInstance, error) {
	return &LibvirtInstance{
		Log:       log,
		ImagePath: imagePath,
	}, nil
}

// ConnectWithInfrastructureService connects to the libvirt instance.
func (l *LibvirtInstance) ConnectWithInfrastructureService(ctx context.Context, url string) error {
	conn, err := libvirt.NewConnect(url)
	if err != nil {
		l.Log.With(zap.Error(err)).DPanic("Failed to connect to libvirt")
	}
	l.Conn = conn
	return nil
}

// InitializeInfrastructure initializes the infrastructure.
func (l *LibvirtInstance) InitializeInfrastructure(ctx context.Context) (err error) {
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
	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < config.ClusterConfiguration.NumberOfWorkers+config.ClusterConfiguration.NumberOfMasters; i++ {
		// the wrapper is necessary to prevent an update of the loop variable.
		// without it, it would race and have the same value all the time.
		func(id int) {
			g.Go(func() error {
				return l.CreateInstance(strconv.Itoa(id))
			})
		}(i)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return err
}

// BootstrapKubernetes initializes kubernetes on the infrastructure.
func (l *LibvirtInstance) BootstrapKubernetes(ctx context.Context, k8sConfig []byte) (err error) {
	if err := l.blockUntilNetworkIsReady(ctx); err != nil {
		return err
	}
	l.Log.Info("network is ready")
	if err := l.blockUntilDelegatioAgentIsReady(ctx); err != nil {
		return err
	}
	l.Log.Info("delegatio-agent is ready")
	if err := l.InstallKubernetes(ctx, k8sConfig); err != nil {
		return err
	}
	if err := l.createAgent(); err != nil {
		return err
	}
	l.Log.Info("kubernetes init successful")
	joinToken, err := l.vmAgent.ConfigureKubernetes(ctx)
	if err != nil {
		return err
	}
	l.Log.Info("join token generated")
	// TODO: check if all nodes are ready
	g, ctxGo := errgroup.WithContext(ctx)
	for i := config.ClusterConfiguration.NumberOfMasters; i < config.ClusterConfiguration.NumberOfWorkers+config.ClusterConfiguration.NumberOfMasters; i++ {
		func(id int) {
			g.Go(func() error {
				return l.JoinClustergRPC(ctxGo, "delegatio-"+strconv.Itoa(id), joinToken)
			})
		}(i)
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return err
}

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

// GetEtcdCredentials returns the etcd credentials for the instance.
func (l *LibvirtInstance) GetEtcdCredentials(ctx context.Context) (*config.EtcdCredentials, error) {
	return l.vmAgent.GetEtcdCredentials(ctx)
}
