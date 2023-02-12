/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateStorageClass creates a persistent volume claim.
func (k *Client) CreateStorageClass(ctx context.Context, name string, reclaimPolicy coreAPI.PersistentVolumeReclaimPolicy) error {
	stClass := templates.StorageClass(name, reclaimPolicy)

	_, err := k.Client.StorageV1().StorageClasses().Create(ctx, stClass, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
