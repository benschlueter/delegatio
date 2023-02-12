/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSecret creates a secret.
func (k *Client) CreateSecret(ctx context.Context, name string) error {
	secret := templates.Secret(name)
	_, err := k.Client.CoreV1().Secrets("kube-system").Create(ctx, secret, metaAPI.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
