/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"

	"github.com/benschlueter/delegatio/internal/infrastructure"
	"github.com/benschlueter/delegatio/internal/infrastructure/utils"

	"go.uber.org/zap"
)

func createInfrastructure(ctx context.Context, log *zap.Logger, infra infrastructure.Infrastructure) (*utils.EtcdCredentials, error) {
	if err := infra.ConnectWithInfrastructureService(ctx, "qemu:///system"); err != nil {
		log.Error("failed to connect with infrastructure", zap.Error(err))
		return nil, err
	}

	if err := infra.InitializeInfrastructure(ctx); err != nil {
		log.Error("failed to start VMs", zap.Error(err))
		return nil, err
	}

	kubeConf, err := infrastructure.GetKubeInitConfig()
	if err != nil {
		log.With(zap.Error(err)).DPanic("failed to get kubeConfig")
	}

	if err := infra.InitializeKubernetes(ctx, kubeConf); err != nil {
		log.Error("failed to run Kubernetes", zap.Error(err))
		return nil, err
	}
	creds, err := infra.GetEtcdCredentials(ctx)
	if err != nil {
		log.Error("failed to obtain etcd credentials", zap.Error(err))
		return nil, err
	}

	return creds, nil
}