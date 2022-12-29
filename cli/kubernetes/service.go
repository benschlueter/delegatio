package kubernetes

import (
	"context"

	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (k *kubernetesClient) CreateService(ctx context.Context, namespace, userID, podSSHPort string) error {
	serv := coreAPI.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: userID + "service",
		},
		Spec: coreAPI.ServiceSpec{
			Type: coreAPI.ServiceTypeExternalName,
			Selector: map[string]string{
				"app.kubernetes.io/name": userID,
			},
			Ports: []coreAPI.ServicePort{
				{
					Protocol:   coreAPI.ProtocolTCP,
					Port:       22,
					TargetPort: intstr.Parse(podSSHPort),
				},
			},
		},
	}
	_, err := k.client.CoreV1().Services(namespace).Create(ctx, &serv, v1.CreateOptions{})
	return err
}
