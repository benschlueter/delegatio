/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	"fmt"

	coreAPI "k8s.io/api/core/v1"

	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Secret creates a secret template.
func Secret(name string) *coreAPI.Secret {
	return &coreAPI.Secret{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Secret",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name: fmt.Sprintf("%s-secret", name),
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": name,
			},
		},
		Type: "kubernetes.io/service-account-token",
	}
}
