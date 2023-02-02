/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	networkAPI "k8s.io/api/networking/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateIngress creates an ingress.
func (k *Client) CreateIngress(ctx context.Context, namespace string) error {
	className := "haproxy"
	pathType := networkAPI.PathTypePrefix

	ing := networkAPI.Ingress{
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

	class := networkAPI.IngressClass{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "IngressClass",
			APIVersion: networkAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name: className,
		},
		Spec: networkAPI.IngressClassSpec{
			Controller: "haproxy.org/ingress-controller",
		},
	}
	/* 	if err := k.CreateNamespace(ctx, "ingress-nginx"); err != nil {
	   		return err
	   	}
	   	if err := k.CreateNamespace(ctx, "tcp-services"); err != nil {
	   		return err
	   	} */
	if err := k.CreateConfigMap(ctx, "default", "my-tcpservices-configmap"); err != nil {
		return err
	}
	if err := k.AddDataToConfigMap(ctx, "default", "my-tcpservices-configmap", "22", "ssh/ssh-relay-service:2200"); err != nil {
		return err
	}
	if _, err := k.Client.NetworkingV1().IngressClasses().Create(ctx, &class, metaAPI.CreateOptions{}); err != nil {
		return err
	}
	_, err := k.Client.NetworkingV1().Ingresses(namespace).Create(ctx, &ing, metaAPI.CreateOptions{})
	return err
}
