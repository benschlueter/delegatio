/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	"fmt"

	"github.com/benschlueter/delegatio/internal/config"
	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// HeadlessService creates a service template.
func HeadlessService(identifier *config.KubeRessourceIdentifier) *coreAPI.Service {
	return &coreAPI.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", identifier.UserIdentifier),
			Labels: map[string]string{
				"app.kubernetes.io/name": identifier.UserIdentifier,
			},
		},
		Spec: coreAPI.ServiceSpec{
			Type: coreAPI.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name": identifier.UserIdentifier,
			},
			ClusterIP: "None",
			Ports: []coreAPI.ServicePort{
				{
					Name:       "agent",
					Protocol:   coreAPI.ProtocolTCP,
					TargetPort: intstr.FromInt(config.AgentPort),
					Port:       config.AgentPort,
				},
			},
		},
	}
}

// ServiceLoadBalancer creates a LB-Service template.
func ServiceLoadBalancer(namespace, serviceName string, portNum int) *coreAPI.Service {
	return &coreAPI.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service", serviceName),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": serviceName,
			},
		},
		Spec: coreAPI.ServiceSpec{
			Type: coreAPI.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app.kubernetes.io/name": serviceName,
			},
			Ports: []coreAPI.ServicePort{
				{
					Name:       serviceName + "-port",
					Protocol:   coreAPI.ProtocolTCP,
					Port:       int32(portNum),
					TargetPort: intstr.IntOrString{IntVal: int32(portNum)},
				},
			},
		},
	}
}

// ServiceClusterIP creates a ClusterIP-Service template.
func ServiceClusterIP(namespace, serviceName string, portNum int) *coreAPI.Service {
	return &coreAPI.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service", serviceName),
			Namespace: namespace,
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
					Name:       serviceName + "-port",
					Protocol:   coreAPI.ProtocolTCP,
					Port:       int32(portNum),
					TargetPort: intstr.IntOrString{IntVal: int32(portNum)},
				},
			},
		},
	}
}
