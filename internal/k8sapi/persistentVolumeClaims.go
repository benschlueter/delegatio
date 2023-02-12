/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePersistentVolumeClaim creates a persistent volume claim.
func (k *Client) CreatePersistentVolumeClaim(ctx context.Context, identifier *config.KubeRessourceIdentifier) error {
	pvc := templates.PersistentVolumeClaim(identifier)
	_, err := k.Client.CoreV1().PersistentVolumeClaims(identifier.Namespace).Create(ctx, pvc, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
