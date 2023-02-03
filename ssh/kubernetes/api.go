/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi"
	"github.com/benschlueter/delegatio/internal/store"
	"go.uber.org/zap"
)

// K8sAPI is the interface used to access kubernetes helpers.
type K8sAPI interface {
	CreateAndWaitForRessources(context.Context, *config.KubeRessourceIdentifier) error
	ExecuteCommandInPod(context.Context, *config.KubeExecConfig) error
	CreatePodPortForward(context.Context, *config.KubeForwardConfig) error
}

// K8sAPIWrapper is the struct used to access kubernetes helpers.
type K8sAPIWrapper struct {
	Client *k8sapi.Client
	logger *zap.Logger
}

// NewK8sAPIWrapper returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewK8sAPIWrapper(logger *zap.Logger) (*K8sAPIWrapper, error) {
	// use the current context in kubeconfig
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	return &K8sAPIWrapper{
		Client: client,
		logger: logger,
	}, nil
}

// CreateAndWaitForRessources creates the ressources for a user in a namespace.
func (k *K8sAPIWrapper) CreateAndWaitForRessources(ctx context.Context, conf *config.KubeRessourceIdentifier) error {
	exists, err := k.Client.StatefulSetExists(ctx, conf.Namespace, conf.UserIdentifier)
	if err != nil {
		return err
	}
	if !exists {
		/* 		if err := k.Client.CreateServiceAccount(ctx, namespace, "development"); err != nil {
			return err
		} */
		/* 		if err := k.Client.CreateClusterRoleBinding(ctx, "testchallenge1", "development"); err != nil {
			return err
		} */
		/* 		if err := k.Client.CreateSecret(ctx, "development"); err != nil {
			return err
		} */
		if err := k.Client.CreateStatefulSetForUser(ctx, conf.Namespace, conf.UserIdentifier); err != nil {
			return err
		}
	}
	if err := k.Client.WaitForPodRunning(ctx, conf.Namespace, conf.UserIdentifier, 1*time.Minute); err != nil {
		return err
	}
	return nil
}

// ExecuteCommandInPod executes a command in the specified pod.
func (k *K8sAPIWrapper) ExecuteCommandInPod(ctx context.Context, conf *config.KubeExecConfig) error {
	return k.Client.CreateExecInPod(ctx, conf.Namespace, conf.PodName, conf.Command, conf.Communication, conf.Communication, conf.Communication, conf.WinQueue, conf.Tty)
}

// CreatePodPortForward creates a port forward on the specified pod.
func (k *K8sAPIWrapper) CreatePodPortForward(ctx context.Context, conf *config.KubeForwardConfig) error {
	return k.Client.CreatePodPortForward(ctx, conf.Namespace, conf.PodName, conf.Port, conf.Communication)
}

// GetStore returns a store backed by kube etcd.
func (k *K8sAPIWrapper) GetStore() (store.Store, error) {
	var err error
	var ns string
	_, present := os.LookupEnv("KUBECONFIG")
	if !present {
		// ns is not ready when container spawns
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ns, err = waitForNamespaceMount(ctx)
		if err != nil {
			k.logger.Error("failed to get namespace, assuming default namespace \"ssh\"", zap.Error(err))
			ns = "ssh"
		}
	} else {
		// out of cluster mode currently assumes 'ssh' namespace
		ns = "ssh"
	}
	k.logger.Info("namespace", zap.String("namespace", ns))
	configData, err := k.Client.GetConfigMapData(context.Background(), ns, "etcd-credentials")
	if err != nil {
		return nil, err
	}
	// logger.Info("config", zap.Any("configData", configData))
	etcdStore, err := store.NewEtcdStore([]string{net.JoinHostPort(configData["advertiseAddr"], "2379")}, k.logger, []byte(configData["caCert"]), []byte(configData["cert"]), []byte(configData["key"]))
	if err != nil {
		return nil, err
	}
	return etcdStore, nil
}

// waitForNamespaceMount waits for the namespace file to be mounted and filled.
func waitForNamespaceMount(ctx context.Context) (string, error) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-t.C:
			data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
			ns := strings.TrimSpace(string(data))
			if len(ns) != 0 {
				return ns, nil
			}
		}
	}
}
