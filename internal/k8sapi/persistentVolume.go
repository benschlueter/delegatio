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

// CreatePersistentVolume creates a persistent volume.
func (k *Client) CreatePersistentVolume(ctx context.Context, identifier *config.KubeRessourceIdentifier) error {
	pVolume := templates.PersistentVolume(identifier)
	_, err := k.Client.CoreV1().PersistentVolumes().Create(ctx, pVolume, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
