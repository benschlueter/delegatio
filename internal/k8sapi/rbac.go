/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateClusterRoleBinding creates a clusterRoleBinding.
func (k *Client) CreateClusterRoleBinding(ctx context.Context, namespace, name string) error {
	binding := templates.ClusterRoleBinding(namespace, name)
	_, err := k.Client.RbacV1().ClusterRoleBindings().Create(ctx, binding, metaAPI.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
