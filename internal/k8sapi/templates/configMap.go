/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap creates a configMap template.
func ConfigMap(namespace, name string) *coreAPI.ConfigMap {
	return &coreAPI.ConfigMap{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
