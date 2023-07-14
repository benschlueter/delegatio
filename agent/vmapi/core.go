/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package vmapi

import (
	"time"

	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

// Core interface contains functions to access the state Core data.
type Core interface {
	ConnectToKubernetes() error
	GetJoinToken(time.Duration) (*kubeadm.BootstrapTokenDiscovery, error)
	GetControlPlaneCertificatesAndKeys() (map[string][]byte, error)
}
