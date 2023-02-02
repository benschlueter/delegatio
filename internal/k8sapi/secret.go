/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSecret creates a secret.
func (k *Client) CreateSecret(ctx context.Context, name string) error {
	secret := v1.Secret{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Secret",
			APIVersion: v1.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name: fmt.Sprintf("%s-secret", name),
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": name,
			},
		},
		Type: "kubernetes.io/service-account-token",
	}

	_, err := k.Client.CoreV1().Secrets("kube-system").Create(ctx, &secret, metaAPI.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
