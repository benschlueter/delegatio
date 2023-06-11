/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"fmt"

	"github.com/benschlueter/delegatio/cli/bootstrapper"
	"github.com/benschlueter/delegatio/cli/infrastructure"
	"github.com/benschlueter/delegatio/cli/installer"
	"github.com/benschlueter/delegatio/cli/terminate"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/config/utils"

	"go.uber.org/zap"
)

func run(ctx context.Context, log *zap.Logger, imageLocation string) error {
	lInstance, err := infrastructure.NewQemu(log.Named("infra"), imageLocation)
	if err != nil {
		log.With(zap.Error(err)).DPanic("error creating infrastructure")
		return err
	}

	defer func(logger *zap.Logger, l infrastructure.Infrastructure) {
		if err := l.TerminateInfrastructure(); err != nil {
			logger.Error("terminate infrastructure", zap.Error(err))
		} else {
			log.Info("instaces terminated successfully")
		}
		if err := l.TerminateConnection(); err != nil {
			logger.Error("closing connection", zap.Error(err))
		} else {
			log.Info("connection successfully closed")
		}
	}(log, lInstance)
	// --- infrastructure ---
	/* 	nodes, err := createInfrastructure(ctx, log, lInstance)
	   	if err != nil {
	   		log.With(zap.Error(err)).DPanic("create infrastructure")
	   	} */
	log.Info("finished infrastructure initialization")
	nodes := &config.NodeInformation{
		Masters: map[string]string{
			"master1": "34.117.25.173",
		},
	}
	fmt.Println(nodes)
	/// --- kubernetes ---
	creds, err := bootstrapKubernetes(ctx, log, nodes)
	if err != nil {
		log.With(zap.Error(err)).DPanic("bootstrap kubernetes")
	}
	log.Info("finished kubernetes bootstrap")
	// --- install ---
	if err := installKubernetesApplications(ctx, log, creds, config.GetExampleConfig()); err != nil {
		log.With(zap.Error(err)).DPanic("failed to initialize kubernetes")
	}
	log.Info("finished kubernetes initialization")

	<-ctx.Done()
	return handleTermination(log, creds)
}

func bootstrapKubernetes(ctx context.Context, log *zap.Logger, nodes *config.NodeInformation) (*config.EtcdCredentials, error) {
	kubeConf, err := utils.GetKubeInitConfig()
	if err != nil {
		log.With(zap.Error(err)).DPanic("failed to get kubeConfig")
	}
	agent, err := bootstrapper.NewKubernetes(log, nodes, kubeConf)
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

func installKubernetesApplications(ctx context.Context, log *zap.Logger, creds *config.EtcdCredentials, userConfig *config.UserConfiguration) error {
	kubeInstaller, err := installer.NewInstaller(log)
	if err != nil {
		return err
	}
	return kubeInstaller.InstallKubernetesApplications(ctx, creds, userConfig)
}

func createInfrastructure(ctx context.Context, log *zap.Logger, infra infrastructure.Infrastructure) (*config.NodeInformation, error) {
	if err := infra.ConnectWithInfrastructureService(ctx, "qemu:///system"); err != nil {
		log.Error("failed to connect with infrastructure", zap.Error(err))
		return nil, err
	}
	nodes, err := infra.InitializeInfrastructure(ctx)
	if err != nil {
		log.Error("failed to start VMs", zap.Error(err))
		return nil, err
	}
	return nodes, nil
}

func handleTermination(log *zap.Logger, creds *config.EtcdCredentials) error {
	cleanUpCtx, secondCancel := context.WithTimeout(context.Background(), config.CleanUpTimeout)
	defer secondCancel()
	// --- terminate ---
	terminator, err := terminate.NewTerminate(log.Named("terminate"), creds)
	if err != nil {
		log.With(zap.Error(err)).DPanic("new terminate")
		return err
	}
	if err := terminator.SaveState(cleanUpCtx, "./kubernetes-state.json"); err != nil {
		log.Error("failed to save state", zap.Error(err))
		return err
	}
	return nil
}
