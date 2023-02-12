/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	coreAPI "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageClass creates a storageClass template.
func StorageClass(name string, reclaimPolicy coreAPI.PersistentVolumeReclaimPolicy) *storagev1.StorageClass {
	return &storagev1.StorageClass{
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
}
