/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package main

import (
	"context"
	"net"
	"net/url"

	"github.com/benschlueter/delegatio/internal/infrastructure/utils"
	"github.com/benschlueter/delegatio/internal/kubernetes"

	"go.uber.org/zap"
)

func createKubernetes(ctx context.Context, log *zap.Logger, creds *utils.EtcdCredentials) error {
	kubeClient, err := kubernetes.NewK8sClient(log.Named("k8sAPI"), "./admin.conf")
	if err != nil {
		log.With(zap.Error(err)).Error("failed to connect to Kubernetes")
		return err
	}

	if err := kubeClient.InstallCilium(ctx); err != nil {
		log.With(zap.Error(err)).Error("failed to install helm charts")
		return err

	}
	if err := kubeClient.Client.CreateNamespace(ctx, "testchallenge1"); err != nil {
		log.With(zap.Error(err)).Error("failed to create namespace")
		return err
	}
	if err := kubeClient.Client.CreateNamespace(ctx, "ssh"); err != nil {
		log.With(zap.Error(err)).Error("failed to create namespace")
		return err
	}

	if err := kubeClient.Client.CreateStorageClass(ctx, "nfs", "Retain"); err != nil {
		log.With(zap.Error(err)).Error("failed to CreateStorageClass")
		return err
	}

	if err := kubeClient.CreatePersistentVolume(ctx, "nfs-storage"); err != nil {
		log.With(zap.Error(err)).Error("failed to CreatePersistentVolume")
		return err
	}

	if err := kubeClient.CreatePersistentVolumeClaim(ctx, "testchallenge1", "nfs-storage", "nfs"); err != nil {
		log.With(zap.Error(err)).Error("failed to CreatePersistentVolumeClaim")
		return err
	}

	u, err := url.Parse(kubeClient.Client.RestConfig.Host)
	if err != nil {
		return err
	}
	log.Info("endpoint", zap.String("api", u.Host))
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	if err := kubeClient.Client.ConnectToStore(creds, []string{net.JoinHostPort(host, "2379")}); err != nil {
		log.With(zap.Error(err)).Error("failed to install helm charts")
		return err
	}
	configMapData := map[string]string{
		"key":           string(creds.KeyData),
		"cert":          string(creds.PeerCertData),
		"caCert":        string(creds.CaCertData),
		"advertiseAddr": host,
	}
	if err := kubeClient.CreateConfigMapAndPutData(ctx, "ssh", "etcd-credentials", configMapData); err != nil {
		log.With(zap.Error(err)).Error("failed to CreatePersistentVolumeClaim")
		return err
	}

	etcdStore := kubeClient.Client.SharedStore
	if err := etcdStore.Put("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLYDO+DPlwJTKYU+S9Q1YkgC7lUJgfsq+V6VxmzdP+omp2EmEIEUsB8WFtr3kAgtAQntaCejJ9ITgoLimkoPs7bV1rA7BZZgRTL2sF+F5zJ1uXKNZz1BVeGGDDXHW5X5V/ZIlH5Bl4kNaAWGx/S5PIszkhyNXEkE6GHsSU4dz69rlutjSbwQRFLx8vjgdAxP9+jUbJMh9u5Dg1SrXiMYpzplJWFt/jI13dDlNTrhWW7790xhHur4fiQbhrVzru29BKNQtSywC+3eH2XKTzobK6h7ECS5X75ghemRIDPw32SHbQP7or1xI+MjFCrZsGyZr1L0yBFNkNAsztpWAqE2FZ", []byte{0x1}); err != nil {
		return err
	}
	if err := etcdStore.Put("testchallenge1", []byte{0x1}); err != nil {
		return err
	}

	if err := kubeClient.Client.CreateServiceAccount(ctx, "ssh", "development"); err != nil {
		return err
	}
	if err := kubeClient.Client.CreateClusterRoleBinding(ctx, "ssh", "development"); err != nil {
		return err
	}

	if err := kubeClient.Client.CreateDeployment(ctx, "ssh", "ssh-relay", 2); err != nil {
		return err
	}
	if err := kubeClient.Client.CreateService(ctx, "ssh", "ssh-relay"); err != nil {
		return err
	}
	if err := kubeClient.Client.CreateIngress(ctx, "ssh"); err != nil {
		return err
	}

	return nil
}
