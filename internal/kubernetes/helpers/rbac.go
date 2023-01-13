/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateClusterRoleBinding creates a clusterRoleBinding.
func (k *Client) CreateClusterRoleBinding(ctx context.Context, namespace, name string) error {
	binding := rbacv1.ClusterRoleBinding{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name: "add-on-cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin",
		},
	}
	_, err := k.Client.RbacV1().ClusterRoleBindings().Create(ctx, &binding, metaAPI.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
