/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"strconv"

	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"
	"libvirt.org/go/libvirt"
)

// LibvirtInstance is a wrapper around libvirt.
type LibvirtInstance struct {
	Conn               *libvirt.Connect
	Log                *zap.Logger
	ImagePath          string
	RegisteredNetworks []string
	RegisteredPools    []string
	RegisteredDisks    []string
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
	l.Log.Info("waiting for instances to be created")
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
	l.Log.Info("waiting for instances to become ready")
	return err
}

// BootstrapKubernetes initializes kubernetes on the infrastructure.
func (l *LibvirtInstance) BootstrapKubernetes(ctx context.Context, k8sConfig []byte) (*config.EtcdCredentials, error) {
	if _, err := l.blockUntilNetworkIsReady(ctx, "delegatio-0"); err != nil {
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
