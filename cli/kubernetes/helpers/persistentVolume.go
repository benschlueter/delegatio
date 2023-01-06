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

// CreatePersistentVolume creates a persistent volume.
func (k *Client) CreatePersistentVolume(ctx context.Context, namespace, name string) error {
	secretNamespace := "default"
	pVolume := coreAPI.PersistentVolume{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coreAPI.PersistentVolumeSpec{
			Capacity: coreAPI.ResourceList{
				coreAPI.ResourceStorage: resource.MustParse("5Gi"),
			},
			StorageClassName: "azurefile-csi",
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.ReadWriteMany,
			},
			PersistentVolumeReclaimPolicy: coreAPI.PersistentVolumeReclaimPolicy("Retain"),
			PersistentVolumeSource: coreAPI.PersistentVolumeSource{
				AzureFile: &coreAPI.AzureFilePersistentVolumeSource{
					SecretName:      "azure-secret",
					SecretNamespace: &secretNamespace,
					ShareName:       "cluster",
					ReadOnly:        false,
				},
			},
		},
	}

	_, err := k.client.CoreV1().PersistentVolumes().Create(ctx, &pVolume, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
