/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	"fmt"

	"github.com/benschlueter/delegatio/internal/config"
	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSet return a statefulSet template.
func StatefulSet(identifier *config.KubeRessourceIdentifier) *appsAPI.StatefulSet {
	return &appsAPI.StatefulSet{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: appsAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      identifier.UserIdentifier + "-statefulset",
			Namespace: identifier.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": identifier.UserIdentifier,
			},
		},
		Spec: appsAPI.StatefulSetSpec{
			Selector: &metaAPI.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": identifier.UserIdentifier,
				},
			},
			ServiceName: fmt.Sprintf("%s-service", identifier.UserIdentifier),
			Template: coreAPI.PodTemplateSpec{
				ObjectMeta: metaAPI.ObjectMeta{
					Name:      identifier.UserIdentifier + "-pod",
					Namespace: identifier.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name": identifier.UserIdentifier,
					},
				},
				Spec: *Pod(identifier),
			},
		},
	}
}
