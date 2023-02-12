/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"

	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	"k8s.io/apimachinery/pkg/api/errors"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a namespace.
func (k *Client) CreateNamespace(ctx context.Context, namespace string) error {
	nspace := templates.Namespace(namespace)
	_, err := k.Client.CoreV1().Namespaces().Create(ctx, nspace, metaAPI.CreateOptions{})
	return err
}

// NamespaceExists creates a namespace.
func (k *Client) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := k.Client.CoreV1().Namespaces().Get(ctx, namespace, metaAPI.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
