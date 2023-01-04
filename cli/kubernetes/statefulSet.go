package kubernetes

import (
	"context"
	"fmt"
	"time"

	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
					GenerateName: userID + "-pod",
				},
				Spec: coreAPI.PodSpec{
					Containers: []coreAPI.Container{
						{
							Name:  "archlinux-container-ssh",
							Image: "ghcr.io/benschlueter/delegatio/archimage:0.1",
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
									Name:      "root-storage",
									MountPath: "/root/",
									SubPath:   userID,
								},
							},
							/* 							Ports: []coreAPI.ContainerPort{
								{
									Name:          "ssh",
									Protocol:      coreAPI.ProtocolTCP,
									ContainerPort: 22,
								},
							}, */
							ImagePullPolicy: coreAPI.PullAlways,
							SecurityContext: &coreAPI.SecurityContext{
								Capabilities: &coreAPI.Capabilities{
									Add: []coreAPI.Capability{
										"CAP_SYS_CHROOT",
									},
								},
							},
						},
					},
					Volumes: []coreAPI.Volume{
						{
							Name: "root-storage",
							VolumeSource: coreAPI.VolumeSource{
								PersistentVolumeClaim: &coreAPI.PersistentVolumeClaimVolumeSource{
									ClaimName: "root-storage-claim",
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
	_, err := k.client.AppsV1().StatefulSets(challengeNamespace).Create(ctx, &sSet, metaAPI.CreateOptions{})

	return err
}

// WaitForStatefulSet waits for a statefulSet to be active.
func (k *Client) WaitForStatefulSet(ctx context.Context, namespace, statefulSetName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isStatefulSetActive(ctx, k.client, statefulSetName, namespace))
}

// StatefulSetExists checks if the statefulset exists.
func (k *Client) StatefulSetExists(ctx context.Context, namespace, statefulSetName string) (bool, error) {
	return isStatefulSetActive(ctx, k.client, statefulSetName, namespace)()
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
