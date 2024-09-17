/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	v1 "k8s.io/api/rbac/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateClusterRoleBinding creates a clusterRoleBinding.
func (k *Client) CreateClusterRoleBinding(ctx context.Context, namespace, name string) error {
	var binding *v1.ClusterRoleBinding

	binding, err := k.Client.RbacV1().ClusterRoleBindings().Get(ctx, "add-on-cluster-admin", metaAPI.GetOptions{})
	if err != nil {
		binding = templates.ClusterRoleBinding(namespace, name)
		_, err = k.Client.RbacV1().ClusterRoleBindings().Create(ctx, binding, metaAPI.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		binding.ResourceVersion = ""
		binding.Subjects = append(binding.Subjects, v1.Subject{
			Kind:      "ServiceAccount",
			Name:      name,
			Namespace: namespace,
		})
		_, err = k.Client.RbacV1().ClusterRoleBindings().Update(ctx, binding, metaAPI.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
