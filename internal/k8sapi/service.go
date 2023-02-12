/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateHeadlessService creates a service.
func (k *Client) CreateHeadlessService(ctx context.Context, identifier *config.KubeRessourceIdentifier) error {
	serv := templates.HeadlessService(identifier)
	_, err := k.Client.CoreV1().Services(identifier.Namespace).Create(ctx, serv, v1.CreateOptions{})
	return err
}

// CreateService creates a service.
func (k *Client) CreateService(ctx context.Context, namespace, serviceName string) error {
	serv := templates.Service(namespace, serviceName)
	_, err := k.Client.CoreV1().Services(namespace).Create(ctx, serv, v1.CreateOptions{})
	return err
}
