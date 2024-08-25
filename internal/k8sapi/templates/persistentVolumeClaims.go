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

// PersistentVolumeClaim return a PVC template.
func PersistentVolumeClaim(identifier *config.KubeRessourceIdentifier) *coreAPI.PersistentVolumeClaim {
	return &coreAPI.PersistentVolumeClaim{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      identifier.UserIdentifier,
			Namespace: identifier.Namespace,
		},
		Spec: coreAPI.PersistentVolumeClaimSpec{
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.ReadWriteMany,
			},
			VolumeName:       identifier.UserIdentifier,
			StorageClassName: &identifier.StorageClass,
			Resources: coreAPI.VolumeResourceRequirements{
				Requests: coreAPI.ResourceList{
					coreAPI.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
}
