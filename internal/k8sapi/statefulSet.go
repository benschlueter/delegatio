/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"fmt"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// CreateChallengeStatefulSet creates a statefulset.
func (k *Client) CreateChallengeStatefulSet(ctx context.Context, challengeNamespace, userID string) error {
	sSet := appsAPI.StatefulSet{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: appsAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      userID + "-statefulset",
			Namespace: challengeNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": userID,
			},
		},
		Spec: appsAPI.StatefulSetSpec{
			Selector: &metaAPI.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": userID,
				},
			},
			ServiceName: fmt.Sprintf("%s-service", userID),
			Template: coreAPI.PodTemplateSpec{
				ObjectMeta: metaAPI.ObjectMeta{
					Name:      userID + "-pod",
					Namespace: challengeNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name": userID,
					},
				},
				Spec: coreAPI.PodSpec{
					/* 					ServiceAccountName:           "development",
					   					AutomountServiceAccountToken: &automountServiceAccountToken, */

					Containers: []coreAPI.Container{
						{
							Resources: coreAPI.ResourceRequirements{
								Limits: coreAPI.ResourceList{
									coreAPI.ResourceCPU:    resource.MustParse("1"),
									coreAPI.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							Name:  "archlinux-container-ssh",
							Image: config.UserContainerImage,
							TTY:   true,
							LivenessProbe: &coreAPI.Probe{
								ProbeHandler: coreAPI.ProbeHandler{
									Exec: &coreAPI.ExecAction{
										Command: []string{"whoami"},
									},
								},
							},
							VolumeMounts: []coreAPI.VolumeMount{
								{
									Name:      "home-storage",
									MountPath: "/root/",
									SubPath:   userID,
								},
							},
							ImagePullPolicy: coreAPI.PullAlways,
							SecurityContext: &coreAPI.SecurityContext{
								Capabilities: &coreAPI.Capabilities{
									Add: []coreAPI.Capability{
										"CAP_SYS_ADMIN",
									},
								},
							},
						},
					},
					Volumes: []coreAPI.Volume{
						{
							Name: "home-storage",
							VolumeSource: coreAPI.VolumeSource{
								PersistentVolumeClaim: &coreAPI.PersistentVolumeClaimVolumeSource{
									// ClaimName: fmt.Sprintf("pvc-%s-statefulset-0", userID),
									ClaimName: challengeNamespace,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := k.CreateHeadlessService(ctx, challengeNamespace, userID); err != nil {
		return err
	}
	_, err := k.Client.AppsV1().StatefulSets(challengeNamespace).Create(ctx, &sSet, metaAPI.CreateOptions{})

	return err
}

// CreateStatefulSetForUser creates want waits for the statefulSet.
func (k *Client) CreateStatefulSetForUser(ctx context.Context, challengeNamespace, userID string) error {
	exists, err := k.NamespaceExists(ctx, challengeNamespace)
	if err != nil {
		return err
	}
	if !exists {
		if err := k.CreateNamespace(ctx, challengeNamespace); err != nil {
			return err
		}
	}
	if err := k.CreateChallengeStatefulSet(ctx, challengeNamespace, userID); err != nil {
		return err
	}
	if err := k.WaitForStatefulSet(ctx, challengeNamespace, userID, 20*time.Second); err != nil {
		return err
	}
	return k.WaitForPodRunning(ctx, challengeNamespace, userID, 4*time.Minute)
}

// WaitForStatefulSet waits for a statefulSet to be active.
func (k *Client) WaitForStatefulSet(ctx context.Context, namespace, statefulSetName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isStatefulSetActive(ctx, k.Client, statefulSetName, namespace))
}

// StatefulSetExists checks if the statefulset exists.
func (k *Client) StatefulSetExists(ctx context.Context, namespace, statefulSetName string) (bool, error) {
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
