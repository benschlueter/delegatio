/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/benschlueter/delegatio/agent/container/containerapi"
	"github.com/benschlueter/delegatio/agent/container/core"
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
	WriteFileInPod(ctx context.Context, conf *config.KubeFileWriteConfig) error
}

// K8sAPIWrapper is the struct used to access kubernetes helpers.
type K8sAPIWrapper struct {
	Client *k8sapi.Client
	API    containerapi.ContainerAPI
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
	// TODO: split core and vmapi into multiple packages / services
	core, err := core.NewCore(logger)
	if err != nil {
		return nil, err
	}
	api := containerapi.New(logger, core, &net.Dialer{})

	return &K8sAPIWrapper{
		Client: client,
		logger: logger,
		API:    api,
	}, nil
}

// CreateAndWaitForRessources creates the ressources for a user in a namespace.
func (k *K8sAPIWrapper) CreateAndWaitForRessources(ctx context.Context, conf *config.KubeRessourceIdentifier) error {
	exists, err := k.Client.UserRessourcesExist(ctx, conf.Namespace, conf.UserIdentifier)
	if err != nil {
		return err
	}
	if !exists {
		if err := k.Client.CreateUserRessources(ctx, conf); err != nil {
			return err
		}
	}
	// In case the ressource exists, but the Pod is not yet ready we need this statement
	// otherwise the ssh server might crash.
	if err := k.Client.WaitForPodRunning(ctx, conf.Namespace, conf.UserIdentifier, 1*time.Minute); err != nil {
		return err
	}
	k.logger.Info("ressources created and ready", zap.String("namespace", conf.Namespace), zap.String("userIdentifier", conf.UserIdentifier))
	return nil
}

// ExecuteCommandInPod executes a command in the specified pod.
func (k *K8sAPIWrapper) ExecuteCommandInPod(ctx context.Context, conf *config.KubeExecConfig) error {
	service, err := k.Client.GetService(ctx, conf.Namespace, fmt.Sprintf("%s-service", conf.UserIdentifier))
	if err != nil {
		k.logger.Error("failed to get service", zap.Error(err))
		return err
	}
	k.logger.Info("cluster ip", zap.String("ip", service.Spec.ClusterIP))

	pod, err := k.Client.GetPod(ctx, conf.Namespace, fmt.Sprintf("%s-statefulset-0", conf.UserIdentifier))
	if err != nil {
		k.logger.Error("failed to get pod", zap.Error(err))
		return err
	}
	k.logger.Info("pod ip", zap.String("ip", pod.Status.PodIP))
	// TODO: there is a race condition, where the pod is ready, but we can't connect to the endpoint yet.
	// Probably should do a vmapi.dial until it succeeds here.
	return k.API.CreateExecInPodgRPC(ctx, net.JoinHostPort(pod.Status.PodIP, fmt.Sprint(config.AgentPort)), conf)
}

// WriteFileInPod writes a file in the specified pod on a remote agent.
func (k *K8sAPIWrapper) WriteFileInPod(ctx context.Context, conf *config.KubeFileWriteConfig) error {
	service, err := k.Client.GetService(ctx, conf.Namespace, fmt.Sprintf("%s-service", conf.UserIdentifier))
	if err != nil {
		k.logger.Error("failed to get service", zap.Error(err))
		return err
	}
	k.logger.Info("cluster ip", zap.String("ip", service.Spec.ClusterIP))

	pod, err := k.Client.GetPod(ctx, conf.Namespace, fmt.Sprintf("%s-statefulset-0", conf.UserIdentifier))
	if err != nil {
		k.logger.Error("failed to get pod", zap.Error(err))
		return err
	}
	k.logger.Info("pod ip", zap.String("ip", pod.Status.PodIP))
	// TODO: there is a race condition, where the pod is ready, but we can't connect to the endpoint yet.
	// Probably should do a vmapi.dial until it succeeds here.
	return k.API.WriteFileInPodgRPC(ctx, net.JoinHostPort(pod.Status.PodIP, fmt.Sprint(config.AgentPort)), conf)
}

// CreatePodPortForward creates a port forward on the specified pod.
func (k *K8sAPIWrapper) CreatePodPortForward(ctx context.Context, conf *config.KubeForwardConfig) error {
	return k.Client.CreatePodPortForward(ctx, conf.Namespace, conf.PodName, conf.Port, conf.Communication)
}

// GetStore returns a store backed by kube etcd. Its only supposed to used within a kubernetes pod.
func (k *K8sAPIWrapper) GetStore() (store.Store, error) {
	return k.Client.GetStore()
}
