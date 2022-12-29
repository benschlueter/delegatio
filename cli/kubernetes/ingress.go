package kubernetes

import (
	"context"

	networkAPI "k8s.io/api/networking/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *kubernetesClient) CreateIngress(ctx context.Context, namespace, userID string) error {
	pathType := networkAPI.PathTypePrefix

	ing := networkAPI.Ingress{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networkAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      "ingress" + userID,
			Namespace: namespace,
		},
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
											Name: userID + "service",
											Port: networkAPI.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := k.client.NetworkingV1().Ingresses(namespace).Create(ctx, &ing, metaAPI.CreateOptions{})
	return err
}
