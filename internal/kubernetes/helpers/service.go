/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"context"
	"fmt"

	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateHeadlessService creates a service.
func (k *Client) CreateHeadlessService(ctx context.Context, namespace, userID string) error {
	serv := coreAPI.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", userID),
			Labels: map[string]string{
				"app.kubernetes.io/name": userID,
			},
		},
		Spec: coreAPI.ServiceSpec{
			Type: coreAPI.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name": userID,
			},
			ClusterIP: "None",
		},
	}
	_, err := k.Client.CoreV1().Services(namespace).Create(ctx, &serv, v1.CreateOptions{})
	return err
}

// CreateService creates a service.
func (k *Client) CreateService(ctx context.Context, namespace, serviceName string) error {
	serv := coreAPI.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", serviceName),
			Labels: map[string]string{
				"app.kubernetes.io/name": serviceName,
			},
		},
		Spec: coreAPI.ServiceSpec{
			Type: coreAPI.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name": serviceName,
			},
			Ports: []coreAPI.ServicePort{
				{
					Name:       "ssh",
					Protocol:   coreAPI.ProtocolTCP,
					Port:       2200,
					TargetPort: intstr.IntOrString{IntVal: 2200},
				},
			},
		},
	}
	_, err := k.Client.CoreV1().Services(namespace).Create(ctx, &serv, v1.CreateOptions{})
	return err
}
