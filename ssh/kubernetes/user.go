/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"os"

	"github.com/benschlueter/delegatio/internal/config"
)

// K8sAPIUser is the interface used to access kubernetes helpers and user data.
type K8sAPIUser interface {
	GetUserInformation() *config.KubeRessourceIdentifier
	GetNamespace() string
	GetAuthenticatedUserID() string
	GetNodeName() string
	K8sAPI
}

// K8sAPIUserWrapper is the struct used to access kubernetes helpers and user data.
type K8sAPIUserWrapper struct {
	UserInformation *config.KubeRessourceIdentifier
	K8sAPI
}

// NewK8sAPIUserWrapper returns a new kuberenetes client-go wrapper.
func NewK8sAPIUserWrapper(api K8sAPI, userInfo *config.KubeRessourceIdentifier) *K8sAPIUserWrapper {
	return &K8sAPIUserWrapper{
		UserInformation: userInfo,
		K8sAPI:          api,
	}
}

// GetUserInformation returns the user information.
func (k *K8sAPIUserWrapper) GetUserInformation() *config.KubeRessourceIdentifier {
	return k.UserInformation
}

// GetNamespace returns the namespace.
func (k *K8sAPIUserWrapper) GetNamespace() string {
	return k.UserInformation.Namespace
}

// GetAuthenticatedUserID returns the authenticated user id.
func (k *K8sAPIUserWrapper) GetAuthenticatedUserID() string {
	return k.UserInformation.UserIdentifier
}

// GetNodeName returns the node this application is currently running on.
func (k *K8sAPIUserWrapper) GetNodeName() string {
	return os.Getenv(config.NodeNameEnvVariable)
}
