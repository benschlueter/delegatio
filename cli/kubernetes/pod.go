package kubernetes

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/client/conditions"
)

// ListPods list all running pods in the "kube-system" namespace.
func (k *Client) ListPods(ctx context.Context, namespace string) error {
	podList, err := k.client.CoreV1().Pods(namespace).List(ctx, metaAPI.ListOptions{})
	if err != nil {
		return err
	}
	for _, v := range podList.Items {
		k.logger.Info(v.Name)
	}
	return nil
}

// WaitForPodRunning waits for a pod to be running.
func (k *Client) WaitForPodRunning(ctx context.Context, namespace, podName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isPodRunning(ctx, k.client, podName, namespace))
}

func isPodRunning(ctx context.Context, c kubernetes.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(ctx, fmt.Sprintf("%s-statefulset-0", podName), metaAPI.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, conditions.ErrPodCompleted
		}
		return false, nil
	}
}
