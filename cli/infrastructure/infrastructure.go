/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package infrastructure

import (
	"context"

	"github.com/benschlueter/delegatio/cli/infrastructure/qemu"
	"github.com/benschlueter/delegatio/cli/infrastructure/utils"
	"go.uber.org/zap"
)

// Infrastructure Interface to create Cluster.
type Infrastructure interface {
	InitializeKubernetes(ctx context.Context, initConfigK8s []byte) error
	InitializeInfrastructure(ctx context.Context) error
	ConnectWithInfrastructureService(ctx context.Context, url string) error
	TerminateInfrastructure() error
	TerminateConnection() error
}

// NewQemu creates a new Qemu Infrastructure.
func NewQemu(log *zap.Logger, imagePath string) Infrastructure {
	return &qemu.LibvirtInstance{
		Log:               log,
		ImagePath:         imagePath,
		RegisteredDomains: make(map[string]*qemu.DomainInfo),
	}
}

// GetKubeInitConfig returns the init config for kubernetes.
func GetKubeInitConfig() ([]byte, error) {
	k8sConfig := utils.InitConfiguration()
	return utils.MarshalK8SResources(&k8sConfig)
}
