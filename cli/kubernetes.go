/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/benschlueter/delegatio/cli/apps"
	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/infrastructure/utils"
	"github.com/benschlueter/delegatio/internal/kubernetes"

	"go.uber.org/zap"
)

// kubeWrapper is a wrapper around internal kubernets.Client.
type kubeWrapper struct {
	kubeClient *kubernetes.Client
	logger     *zap.Logger
}

// NewKubeWrapper returns a new kubeWrapper.
func NewKubeWrapper(logger *zap.Logger, adminConfPath string) (*kubeWrapper, error) {
	kubeClient, err := kubernetes.NewK8sClient(logger.Named("k8sAPI"), adminConfPath)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to connect to Kubernetes")
		return nil, err
	}
	return &kubeWrapper{kubeClient: kubeClient, logger: logger}, nil
}

func (kW *kubeWrapper) createKubernetes(ctx context.Context, creds *utils.EtcdCredentials, config *config.UserConfiguration) error {
	if err := kW.kubeClient.InstallCilium(ctx); err != nil {
		kW.logger.With(zap.Error(err)).Error("failed to install helm charts")
		return err
	}
	if err := apps.InitializeSSH(ctx, kW.logger.Named("ssh"), kW.kubeClient, creds); err != nil {
		kW.logger.With(zap.Error(err)).Error("failed to deploy ssh config")
		return err
	}
	if err := apps.InitalizeChallenges(ctx, kW.logger.Named("challenges"), kW.kubeClient, config); err != nil {
		kW.logger.With(zap.Error(err)).Error("failed to deploy challenges")
		return err
	}
	return nil
}

func (kW *kubeWrapper) saveKubernetesState(_ context.Context, configFile string) error {
	configData, err := kW.kubeClient.Client.GetStoreUserData()
	if err != nil {
		return err
	}
	byteData, err := json.Marshal(configData)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, byteData, 0o600); err != nil {
		return err
	}
	return nil
}
