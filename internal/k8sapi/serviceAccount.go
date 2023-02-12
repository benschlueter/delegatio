/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceAccount creates a serviceAccount.
func (k *Client) CreateServiceAccount(ctx context.Context, namespace, serviceAccountName string) error {
	_, err := k.Client.CoreV1().ServiceAccounts(namespace).Create(ctx, templates.ServiceAccount(namespace, serviceAccountName), v1.CreateOptions{})
	return err
}
