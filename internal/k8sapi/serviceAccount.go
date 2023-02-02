/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceAccount creates a serviceAccount.
func (k *Client) CreateServiceAccount(ctx context.Context, namespace, serviceAccountName string) error {
	servAcc := coreAPI.ServiceAccount{
		TypeMeta: v1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}
	_, err := k.Client.CoreV1().ServiceAccounts(namespace).Create(ctx, &servAcc, v1.CreateOptions{})
	return err
}
