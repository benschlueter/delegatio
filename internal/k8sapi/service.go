/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	coreAPI "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateHeadlessService creates a service.
func (k *Client) CreateHeadlessService(ctx context.Context, identifier *config.KubeRessourceIdentifier) error {
	serv := templates.HeadlessService(identifier)
	_, err := k.Client.CoreV1().Services(identifier.Namespace).Create(ctx, serv, metav1.CreateOptions{})
	return err
}

// CreateServiceLoadBalancer creates a LoadBalancer service.
func (k *Client) CreateServiceLoadBalancer(ctx context.Context, namespace, serviceName string, portNum int) error {
	serv := templates.ServiceLoadBalancer(namespace, serviceName, portNum)
	_, err := k.Client.CoreV1().Services(namespace).Create(ctx, serv, metav1.CreateOptions{})
	return err
}

// CreateServiceClusterIP creates a ClusterIP service.
func (k *Client) CreateServiceClusterIP(ctx context.Context, namespace, serviceName string, portNum int) error {
	serv := templates.ServiceClusterIP(namespace, serviceName, portNum)
	_, err := k.Client.CoreV1().Services(namespace).Create(ctx, serv, metav1.CreateOptions{})
	return err
}

// GetService gets a service.
func (k *Client) GetService(ctx context.Context, namespace, serviceName string) (*coreAPI.Service, error) {
	return k.Client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
}
