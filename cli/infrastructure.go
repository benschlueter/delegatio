/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"

	"github.com/benschlueter/delegatio/cli/bootstrapper"
	"github.com/benschlueter/delegatio/cli/infrastructure"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/config/utils"

	"go.uber.org/zap"
)

func createInfrastructure(ctx context.Context, log *zap.Logger, infra infrastructure.Infrastructure) (*config.EtcdCredentials, error) {
	if err := infra.ConnectWithInfrastructureService(ctx, "qemu:///system"); err != nil {
		log.Error("failed to connect with infrastructure", zap.Error(err))
		return nil, err
	}
	nodes, err := infra.InitializeInfrastructure(ctx)
	if err != nil {
		log.Error("failed to start VMs", zap.Error(err))
		return nil, err
	}
	kubeConf, err := utils.GetKubeInitConfig()
	if err != nil {
		log.With(zap.Error(err)).DPanic("failed to get kubeConfig")
	}
	agent, err := bootstrapper.NewBootstrapper(log, nodes, kubeConf)
	if err != nil {
		log.Error("failed to initialize bootstrapper", zap.Error(err))
		return nil, err
	}
	creds, err := agent.BootstrapKubernetes(ctx)
	if err != nil {
		log.Error("failed to initialize Kubernetes", zap.Error(err))
		return nil, err
	}

	return creds, nil
}
