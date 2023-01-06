/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"context"

	coreAPI "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePersistentVolumeClaim creates a persistent volume claim.
func (k *Client) CreatePersistentVolumeClaim(ctx context.Context, namespace, name string) error {
	stclass := "azurefile-csi"
	pVolumeClaim := coreAPI.PersistentVolumeClaim{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coreAPI.PersistentVolumeClaimSpec{
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.ReadWriteMany,
			},
			VolumeName:       name,
			StorageClassName: &stclass,
			Resources: coreAPI.ResourceRequirements{
				Requests: coreAPI.ResourceList{
					coreAPI.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}

	_, err := k.client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &pVolumeClaim, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
