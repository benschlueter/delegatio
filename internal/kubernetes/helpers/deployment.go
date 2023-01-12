/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"context"

	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var automountServiceAccountToken = true

// CreateDeployment creates a deployment.
func (k *Client) CreateDeployment(ctx context.Context, namespace, deploymentName string, replicas int32) error {
	deployment := appsAPI.Deployment{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      deploymentName + "-deployment",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Spec: appsAPI.DeploymentSpec{
			Selector: &metaAPI.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": deploymentName,
				},
			},
			Replicas: &replicas,
			Template: coreAPI.PodTemplateSpec{
				ObjectMeta: metaAPI.ObjectMeta{
					Name:      deploymentName + "-pod",
					Namespace: namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name": deploymentName,
					},
				},
				Spec: coreAPI.PodSpec{
					ServiceAccountName:           "development",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					// Somehow needed, otherwise nginx wont connect to the pods.
					HostNetwork: true,
					Containers: []coreAPI.Container{
						{
							Name:  "ssh-relay",
							Image: "ghcr.io/benschlueter/delegatio/ssh:0.1",
							TTY:   true,
							LivenessProbe: &coreAPI.Probe{
								ProbeHandler: coreAPI.ProbeHandler{
									Exec: &coreAPI.ExecAction{
										Command: []string{"whoami"},
									},
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
							Ports: []coreAPI.ContainerPort{
								{
									Name:          "ssh",
									ContainerPort: 2200,
									Protocol:      coreAPI.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := k.Client.AppsV1().Deployments(namespace).Create(ctx, &deployment, metaAPI.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
