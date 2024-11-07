/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package bootstrapper

import (
	"context"

	"github.com/benschlueter/delegatio/cli/bootstrapper/kubernetes"
	"github.com/benschlueter/delegatio/internal/config"
	"go.uber.org/zap"
)

// Bootstrapper is the interface for the bootstrapper. It is used to bootstrap a kubernetes cluster.
// It covers Kubeadm init on the control plane, as well as joining the worker nodes.
type Bootstrapper interface {
	BootstrapKubernetes(context.Context) (*config.EtcdCredentials, error)
}

// NewKubernetes creates a new Kubernetes bootstrapper.
func NewKubernetes(log *zap.Logger, k8sConfig []byte) (Bootstrapper, error) {
	instance, err := kubernetes.NewBootstrapper(log, k8sConfig)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
