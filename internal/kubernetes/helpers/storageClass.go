/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"context"

	coreAPI "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateStorageClass creates a persistent volume claim.
func (k *Client) CreateStorageClass(ctx context.Context, name string, reclaimPolicy coreAPI.PersistentVolumeReclaimPolicy) error {
	stClass := storagev1.StorageClass{
		TypeMeta: v1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner:   "kubernetes.io/nfs",
		ReclaimPolicy: &reclaimPolicy,
	}

	_, err := k.Client.StorageV1().StorageClasses().Create(ctx, &stClass, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
