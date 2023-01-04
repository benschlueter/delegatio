package kubernetes

import (
	"context"

	coreAPI "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePersistentVolume creates a persistent volume.
func (k *Client) CreatePersistentVolume(ctx context.Context, namespace, name string) error {
	fsType := "xfs"
	pVolume := coreAPI.PersistentVolume{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coreAPI.PersistentVolumeSpec{
			/* 			Capacity: coreAPI.ResourceList{
				coreAPI.ResourceEphemeralStorage: resource.MustParse("10Gi"),
			}, */
			StorageClassName: "standard",
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.ReadWriteMany,
			},
			PersistentVolumeReclaimPolicy: coreAPI.PersistentVolumeReclaimPolicy("recycle"),
			PersistentVolumeSource: coreAPI.PersistentVolumeSource{
				Local: &coreAPI.LocalVolumeSource{
					Path:   "/mnt/",
					FSType: &fsType,
				},
			},
			NodeAffinity: &coreAPI.VolumeNodeAffinity{
				Required: &coreAPI.NodeSelector{
					NodeSelectorTerms: []coreAPI.NodeSelectorTerm{
						{
							MatchExpressions: []coreAPI.NodeSelectorRequirement{
								{
									Key: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := k.client.CoreV1().PersistentVolumes().Create(ctx, &pVolume, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// CreatePersistentVolumeClaim creates a persistent volume claim.
func (k *Client) CreatePersistentVolumeClaim(ctx context.Context, namespace, name string) error {
	pVolumeClaim := coreAPI.PersistentVolumeClaim{
		TypeMeta: v1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coreAPI.PersistentVolumeClaimSpec{
			AccessModes: []coreAPI.PersistentVolumeAccessMode{
				coreAPI.ReadWriteMany,
			},
			VolumeName: name,
		},
	}

	_, err := k.client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &pVolumeClaim, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
