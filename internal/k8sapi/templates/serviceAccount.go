/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccount creates a serviceAccount.
func ServiceAccount(namespace, serviceAccountName string) *coreAPI.ServiceAccount {
	return &coreAPI.ServiceAccount{
		TypeMeta: v1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}
}
