/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateIngress creates an ingress.
func (k *Client) CreateIngress(ctx context.Context, namespace string) error {
	ing := templates.Ingress(namespace)
	class := templates.IngressClass(namespace)
	if err := k.CreateConfigMap(ctx, "default", "my-tcpservices-configmap"); err != nil {
		return err
	}
	if err := k.AddDataToConfigMap(ctx, "default", "my-tcpservices-configmap", "22", "ssh/ssh-relay-service:2200"); err != nil {
		return err
	}
	if _, err := k.Client.NetworkingV1().IngressClasses().Create(ctx, class, metaAPI.CreateOptions{}); err != nil {
		return err
	}
	_, err := k.Client.NetworkingV1().Ingresses(namespace).Create(ctx, ing, metaAPI.CreateOptions{})
	return err
}
