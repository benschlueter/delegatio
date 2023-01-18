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
func (k *Client) CreatePersistentVolume(ctx context.Context, name, accessMode string) error {
	pVolume := coreAPI.PersistentVolume{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: coreAPI.PersistentVolumeSpec{
			Capacity: coreAPI.ResourceList{
				coreAPI.ResourceStorage: resource.MustParse("10Gi"),
			},
			StorageClassName: "nfs",
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.PersistentVolumeAccessMode(accessMode),
			},
			PersistentVolumeReclaimPolicy: coreAPI.PersistentVolumeReclaimPolicy("Retain"),
			PersistentVolumeSource: coreAPI.PersistentVolumeSource{
				NFS: &coreAPI.NFSVolumeSource{
					Server:   "10.42.0.1",
					Path:     "/",
					ReadOnly: false,
				},
			},
			MountOptions: []string{"nfsvers=4.2"},
		},
	}

	_, err := k.Client.CoreV1().PersistentVolumes().Create(ctx, &pVolume, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
