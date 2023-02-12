/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"fmt"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi/templates"
	"k8s.io/apimachinery/pkg/api/errors"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// CreateUserStatefulSet creates a statefulset.
func (k *Client) CreateUserStatefulSet(ctx context.Context, identifier *config.KubeRessourceIdentifier) error {
	if err := k.CreateHeadlessService(ctx, identifier); err != nil {
		return err
	}
	_, err := k.Client.AppsV1().StatefulSets(identifier.Namespace).Create(ctx, templates.StatefulSet(identifier), metaAPI.CreateOptions{})

	return err
}

// CreateUserRessources creates want waits for the statefulSet.
func (k *Client) CreateUserRessources(ctx context.Context, identifier *config.KubeRessourceIdentifier) error {
	identifier.StorageClass = "nfs"
	exists, err := k.NamespaceExists(ctx, identifier.Namespace)
	if err != nil {
		return err
	}
	if !exists {
		if err := k.CreateNamespace(ctx, identifier.Namespace); err != nil {
			return err
		}
	}
	if err := k.CreateUserStatefulSet(ctx, identifier); err != nil {
		return err
	}
	if err := k.WaitForStatefulSet(ctx, identifier.Namespace, identifier.UserIdentifier, 20*time.Second); err != nil {
		return err
	}
	if err := k.CreatePersistentVolumeClaim(ctx, identifier); err != nil {
		return err
	}
	if err := k.CreatePersistentVolume(ctx, identifier); err != nil {
		return err
	}
	return k.WaitForPodRunning(ctx, identifier.Namespace, identifier.UserIdentifier, 4*time.Minute)
}

// WaitForStatefulSet waits for a statefulSet to be active.
func (k *Client) WaitForStatefulSet(ctx context.Context, namespace, statefulSetName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isStatefulSetActive(ctx, k.Client, statefulSetName, namespace))
}

// UserRessourcesExist checks if the statefulset exists.
func (k *Client) UserRessourcesExist(ctx context.Context, namespace, statefulSetName string) (bool, error) {
	return isStatefulSetActive(ctx, k.Client, statefulSetName, namespace)()
}

func isStatefulSetActive(ctx context.Context, c kubernetes.Interface, statefulSetName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := c.AppsV1().StatefulSets(namespace).Get(ctx, fmt.Sprintf("%s-statefulset", statefulSetName), metaAPI.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}
}
