/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package apps

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/kubernetes"
	"github.com/benschlueter/delegatio/internal/storewrapper"
	"go.uber.org/zap"
)

// InitalizeChallenges creates the namespaces and persistent volumes for the challenges. It also adds the users to etcd.
func InitalizeChallenges(ctx context.Context, log *zap.Logger, kubeClient *kubernetes.Client, config *config.UserConfiguration) error {
	if err := kubeClient.Client.CreateStorageClass(ctx, "nfs", "Retain"); err != nil {
		log.With(zap.Error(err)).Error("failed to CreateStorageClass")
		return err
	}
	stWrapper := storewrapper.StoreWrapper{Store: kubeClient.Client.SharedStore}

	for namespace := range config.Challenges {
		if err := kubeClient.Client.CreateNamespace(ctx, namespace); err != nil {
			log.With(zap.Error(err)).Error("failed to create namespace")
			return err
		}
		if err := kubeClient.CreatePersistentVolume(ctx, namespace); err != nil {
			log.With(zap.Error(err)).Error("failed to CreatePersistentVolume")
			return err
		}

		if err := kubeClient.CreatePersistentVolumeClaim(ctx, namespace, namespace, "nfs"); err != nil {
			log.With(zap.Error(err)).Error("failed to CreatePersistentVolumeClaim")
			return err
		}
		if err := stWrapper.PutChallenge(namespace, nil); err != nil {
			return err
		}
	}
	for publicKey, realName := range config.PubKeyToUser {
		if err := stWrapper.PutPublicKey(publicKey, realName); err != nil {
			return err
		}
		log.Info("added user to store", zap.String("publicKey", publicKey), zap.String("realName", realName))
	}

	return nil
}
