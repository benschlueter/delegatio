/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	networkAPI "k8s.io/api/networking/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	className = "haproxy"
	pathType  = networkAPI.PathTypePrefix
)

// Ingress creates an ingress template.
func Ingress(namespace string) *networkAPI.Ingress {
	return &networkAPI.Ingress{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networkAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      "ingress" + "ssh",
			Namespace: namespace,
		},
		/* 		Spec: networkAPI.IngressSpec{
			DefaultBackend: &networkAPI.IngressBackend{
				Service: &networkAPI.IngressServiceBackend{
					Name: userID + "service",
					Port: networkAPI.ServiceBackendPort{
						Number: 22,
					},
				},
			},

			IngressClassName: &className,
		}, */
		Spec: networkAPI.IngressSpec{
			Rules: []networkAPI.IngressRule{
				{
					IngressRuleValue: networkAPI.IngressRuleValue{
						HTTP: &networkAPI.HTTPIngressRuleValue{
							Paths: []networkAPI.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkAPI.IngressBackend{
										Service: &networkAPI.IngressServiceBackend{
											Name: "ssh-relay-service",
											Port: networkAPI.ServiceBackendPort{
												Number: 2200,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			IngressClassName: &className,
		},
	}
}

// IngressClass creates an ingressClass template.
func IngressClass(namespace string) *networkAPI.IngressClass {
	return &networkAPI.IngressClass{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "IngressClass",
			APIVersion: networkAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      className,
			Namespace: namespace,
		},
		Spec: networkAPI.IngressClassSpec{
			Controller: "haproxy.org/ingress-controller",
		},
	}
}
