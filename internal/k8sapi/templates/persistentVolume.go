/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	"github.com/benschlueter/delegatio/internal/config"
	coreAPI "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolume creates a PV template.
func PersistentVolume(identifier *config.KubeRessourceIdentifier) *coreAPI.PersistentVolume {
	return &coreAPI.PersistentVolume{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: identifier.UserIdentifier,
		},
		Spec: coreAPI.PersistentVolumeSpec{
			Capacity: coreAPI.ResourceList{
				coreAPI.ResourceStorage: resource.MustParse("10Gi"),
			},
			StorageClassName: "nfs",
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.ReadWriteMany,
			},
			PersistentVolumeReclaimPolicy: coreAPI.PersistentVolumeReclaimPolicy("Retain"),
			PersistentVolumeSource: coreAPI.PersistentVolumeSource{
				NFS: &coreAPI.NFSVolumeSource{
					Server: "10.42.0.1",
					// Authenticated userID cannot contains dots (.), thus we can use it as a path
					Path:     "/",
					ReadOnly: false,
				},
			},
			MountOptions: []string{"nfsvers=4.2"},
			ClaimRef: &coreAPI.ObjectReference{
				Kind:       "PersistentVolumeClaim",
				APIVersion: coreAPI.SchemeGroupVersion.Version,
				Name:       identifier.UserIdentifier,
				Namespace:  config.UserNamespace,
			},
		},
	}
}
