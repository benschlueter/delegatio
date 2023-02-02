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
	"github.com/benschlueter/delegatio/internal/installer"
	v1 "k8s.io/api/core/v1"

	"go.uber.org/zap"
)

// KubeWrapper is a wrapper around internal kubernets.Client.
type KubeWrapper struct {
	kubeClient *installer.Client
	logger     *zap.Logger
}

// NewKubeWrapper returns a new kubeWrapper.
func NewKubeWrapper(logger *zap.Logger) (*KubeWrapper, error) {
	kubeClient, err := installer.NewK8sClient(logger.Named("k8sAPI"))
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to connect to Kubernetes")
		return nil, err
	}
	return &KubeWrapper{kubeClient: kubeClient, logger: logger}, nil
}

func (kW *KubeWrapper) createKubernetes(ctx context.Context, creds *config.EtcdCredentials, config *config.UserConfiguration) error {
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
	if err := kW.kubeClient.Client.CreatePersistentVolume(ctx, "prometheus-2", string(v1.ReadWriteOnce)); err != nil {
		kW.logger.With(zap.Error(err)).Error("failed to create persistent volume")
		return err
	}
	if err := kW.kubeClient.Client.CreatePersistentVolume(ctx, "prometheus-1", string(v1.ReadWriteOnce)); err != nil {
		kW.logger.With(zap.Error(err)).Error("failed to create persistent volume")
		return err
	}
	return nil
}

func (kW *KubeWrapper) saveKubernetesState(_ context.Context, configFile string) error {
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
