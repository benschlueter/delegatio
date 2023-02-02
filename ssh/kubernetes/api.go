/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi"
	"go.uber.org/zap"
)

// K8sAPI is the interface used to access kubernetes helpers.
type K8sAPI interface {
	CreateAndWaitForRessources(context.Context, *config.KubeRessourceIdentifier) error
	ExecuteCommandInPod(context.Context, *config.KubeExecConfig) error
	CreatePodPortForward(context.Context, *config.KubeForwardConfig) error
}

// K8sapiWrapper is the struct used to access kubernetes helpers.
type K8sapiWrapper struct {
	Client *k8sapi.Client
	logger *zap.Logger
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
// if no kubeconfig path is given we use the service account token.
func NewK8sClient(logger *zap.Logger) (*K8sapiWrapper, error) {
	// use the current context in kubeconfig
	client, err := k8sapi.NewClient(logger)
	if err != nil {
		return nil, err
	}
	return &K8sapiWrapper{
		Client: client,
		logger: logger,
	}, nil
}

// CreateAndWaitForRessources creates the ressources for a user in a namespace.
func (k *K8sapiWrapper) CreateAndWaitForRessources(ctx context.Context, conf *config.KubeRessourceIdentifier) error {
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
func (k *K8sapiWrapper) ExecuteCommandInPod(ctx context.Context, conf *config.KubeExecConfig) error {
	return k.Client.CreateExecInPod(ctx, conf.Namespace, conf.PodName, conf.Command, conf.Communication, conf.Communication, conf.Communication, conf.WinQueue, conf.Tty)
}

// CreatePodPortForward creates a port forward on the specified pod.
func (k *K8sapiWrapper) CreatePodPortForward(ctx context.Context, conf *config.KubeForwardConfig) error {
	return k.Client.CreatePodPortForward(ctx, conf.Namespace, conf.PodName, conf.Port, conf.Communication)
}
