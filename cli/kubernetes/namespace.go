package kubernetes

import (
	"context"

	coreAPI "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a namespace.
func (k *Client) CreateNamespace(ctx context.Context, namespace string) error {
	nspace := coreAPI.Namespace{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Namespace",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := k.client.CoreV1().Namespaces().Create(ctx, &nspace, metaAPI.CreateOptions{})
	return err
}

// NamespaceExists creates a namespace.
func (k *Client) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := k.client.CoreV1().Namespaces().Get(ctx, namespace, metaAPI.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
