/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 */

package utils

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconf "k8s.io/kubelet/config/v1beta1"
	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
)

// GetKubeInitConfig returns the init config for kubernetes.
func GetKubeInitConfig() ([]byte, error) {
	k8sConfig := initConfiguration()
	return marshalK8SResources(&k8sConfig)
}

// kubeadmInitYAML groups multiple kubernetes config files into one struct.
type kubeadmInitYAML struct {
	InitConfiguration    kubeadm.InitConfiguration
	ClusterConfiguration kubeadm.ClusterConfiguration
	KubeletConfiguration kubeletconf.KubeletConfiguration
}

// initConfiguration sets the pre-defined values for kubernetes.
func initConfiguration() kubeadmInitYAML {
	return kubeadmInitYAML{
		InitConfiguration: kubeadm.InitConfiguration{
			TypeMeta: v1.TypeMeta{
				APIVersion: kubeadm.SchemeGroupVersion.String(),
				Kind:       "InitConfiguration",
			},
			NodeRegistration: kubeadm.NodeRegistrationOptions{
				CRISocket: "unix:///var/run/crio/crio.sock",
				KubeletExtraArgs: []kubeadm.Arg{
					{
						Name:  "cloud-provider",
						Value: "external",
					},
				},
			},
			LocalAPIEndpoint: kubeadm.APIEndpoint{
				BindPort: 6443,
			},
			// kube-proxy will be replaced by cilium.
			SkipPhases: []string{
				"addon/kube-proxy",
				"show-join-command",
			},
		},
		ClusterConfiguration: kubeadm.ClusterConfiguration{
			TypeMeta: v1.TypeMeta{
				Kind:       "ClusterConfiguration",
				APIVersion: kubeadm.SchemeGroupVersion.String(),
			},
			// necessary to be able to access the kubeapi server through localhost
			APIServer: kubeadm.APIServer{
				CertSANs: []string{"127.0.0.1"},
			},
			ControllerManager: kubeadm.ControlPlaneComponent{
				ExtraArgs: []kubeadm.Arg{
					{
						Name:  "flex-volume-plugin-dir",
						Value: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/",
					}, {
						Name:  "cloud-provider",
						Value: "external",
					}, {
						Name:  "configure-cloud-routes",
						Value: "false",
					},
				},
			},
			Etcd: kubeadm.Etcd{
				Local: &kubeadm.LocalEtcd{
					ExtraArgs: []kubeadm.Arg{},
				},
			},
		},
		KubeletConfiguration: kubeletconf.KubeletConfiguration{
			TypeMeta: v1.TypeMeta{
				APIVersion: kubeletconf.SchemeGroupVersion.String(),
				Kind:       "KubeletConfiguration",
			},
			CgroupDriver: "systemd",
		},
	}
}
