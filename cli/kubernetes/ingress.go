package kubernetes

import (
	"context"

	networkAPI "k8s.io/api/networking/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *kubernetesClient) CreateIngress(ctx context.Context, namespace, userID string) error {
	className := "nginx"
	// pathType := networkAPI.PathTypePrefix

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
			DefaultBackend: &networkAPI.IngressBackend{
				Service: &networkAPI.IngressServiceBackend{
					Name: userID + "service",
					Port: networkAPI.ServiceBackendPort{
						Number: 22,
					},
				},
			},

			IngressClassName: &className,
		},
		/* 		Spec: networkAPI.IngressSpec{
		Rules: []networkAPI.IngressRule{
			{
				Host: "challenge1",
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
											Number: 22,
										},
									},
								},
							},
						},
					},
				},
			},
		}, */

	}
	if err := k.CreateNamespace(ctx, "ingress-nginx"); err != nil {
		return err
	}
	if err := k.CreateConfigMap(ctx, "tcp-services", "ingress-nginx"); err != nil {
		return err
	}
	if err := k.AddDataToConfigMap(ctx, "tcp-services", "ingress-nginx", "22", namespace+"/"+userID+"service"+":"+"22"); err != nil {
		return err
	}

	_, err := k.client.NetworkingV1().Ingresses(namespace).Create(ctx, &ing, metaAPI.CreateOptions{})
	return err
}
