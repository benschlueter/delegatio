/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package templates

import (
	"github.com/benschlueter/delegatio/internal/config"
	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
)

// Pod creates a Pod template.
func Pod(identifier *config.KubeRessourceIdentifier) *coreAPI.PodSpec {
	return &coreAPI.PodSpec{
		/* 	ServiceAccountName:           "development",
		AutomountServiceAccountToken: &automountServiceAccountToken, */
		NodeName: identifier.NodeName,
		Containers: []coreAPI.Container{
			{
				Env: []v1.EnvVar{
					{
						Name: config.NodeNameEnvVariable,
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								FieldPath: "spec.nodeName",
							},
						},
					},
					{
						Name:  config.UUIDEnvVariable,
						Value: identifier.UserIdentifier,
					},
				},
				/* 				Resources: coreAPI.ResourceRequirements{
					Limits: coreAPI.ResourceList{
						coreAPI.ResourceCPU:    resource.MustParse("1"),
						coreAPI.ResourceMemory: resource.MustParse("1Gi"),
					},
				}, */
				Name:  "archlinux-container-ssh",
				Image: config.UserContainerImage,
				TTY:   true,
				LivenessProbe: &coreAPI.Probe{
					ProbeHandler: coreAPI.ProbeHandler{
						Exec: &coreAPI.ExecAction{
							Command: []string{"whoami"},
						},
						GRPC: &v1.GRPCAction{},
					},
				},
				VolumeMounts: []coreAPI.VolumeMount{
					{
						Name:      "home-storage",
						MountPath: "/root/",
						SubPath:   identifier.UserIdentifier,
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
				Ports: []v1.ContainerPort{
					{
						Name:          "agent",
						ContainerPort: config.AgentPort,
						Protocol:      coreAPI.ProtocolTCP,
					},
				},
			},
		},
		Volumes: []coreAPI.Volume{
			{
				Name: "home-storage",
				VolumeSource: coreAPI.VolumeSource{
					PersistentVolumeClaim: &coreAPI.PersistentVolumeClaimVolumeSource{
						ClaimName: identifier.UserIdentifier,
					},
				},
			},
		},
	}
}
