/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"

	"github.com/benschlueter/delegatio/cli/apps"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/infrastructure/utils"
	"github.com/benschlueter/delegatio/internal/kubernetes"

	"go.uber.org/zap"
)

func createKubernetes(ctx context.Context, log *zap.Logger, creds *utils.EtcdCredentials, config *config.UserConfiguration) error {
	kubeClient, err := kubernetes.NewK8sClient(log.Named("k8sAPI"), "./admin.conf")
	if err != nil {
		log.With(zap.Error(err)).Error("failed to connect to Kubernetes")
		return err
	}

	if err := kubeClient.InstallCilium(ctx); err != nil {
		log.With(zap.Error(err)).Error("failed to install helm charts")
		return err
	}
	if err := apps.InitializeSSH(ctx, log, kubeClient, creds); err != nil {
		log.With(zap.Error(err)).Error("failed to deploy ssh config")
		return err
	}
	if err := apps.InitalizeChallenges(ctx, log, kubeClient, config); err != nil {
		log.With(zap.Error(err)).Error("failed to deploy challenges")
		return err
	}
	return nil
}
