/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package infrastructure

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/infrastructure/qemu"
	"go.uber.org/zap"
)

// Infrastructure Interface to create Cluster.
type Infrastructure interface {
	BootstrapKubernetes(ctx context.Context, initConfigK8s []byte) (*config.EtcdCredentials, error)
	InitializeInfrastructure(ctx context.Context) error
	ConnectWithInfrastructureService(ctx context.Context, url string) error
	TerminateInfrastructure() error
	TerminateConnection() error
}

// NewQemu creates a new Qemu Infrastructure.
func NewQemu(log *zap.Logger, imagePath string) (Infrastructure, error) {
	instance, err := qemu.NewQemu(log, imagePath)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
