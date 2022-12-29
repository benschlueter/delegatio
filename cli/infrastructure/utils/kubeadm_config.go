package utils

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconf "k8s.io/kubelet/config/v1beta1"
	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

// Uses types defined here: https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/
// Slimmed down to the fields we require

const (
	bindPort = 6443
)

type KubeadmInitYAML struct {
	InitConfiguration    kubeadm.InitConfiguration
	ClusterConfiguration kubeadm.ClusterConfiguration
	KubeletConfiguration kubeletconf.KubeletConfiguration
}

func InitConfiguration() KubeadmInitYAML {
	return KubeadmInitYAML{
		InitConfiguration: kubeadm.InitConfiguration{
			TypeMeta: v1.TypeMeta{
				APIVersion: kubeadm.SchemeGroupVersion.String(),
				Kind:       "InitConfiguration",
			},
			NodeRegistration: kubeadm.NodeRegistrationOptions{
				CRISocket: "unix:///var/run/crio/crio.sock",
				KubeletExtraArgs: map[string]string{
					"cloud-provider": "external",
				},
			},
			// AdvertiseAddress will be overwritten later
			LocalAPIEndpoint: kubeadm.APIEndpoint{
				BindPort: bindPort,
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
				ExtraArgs: map[string]string{
					"flex-volume-plugin-dir": "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/",
					"cloud-provider":         "external",
					"configure-cloud-routes": "false",
				},
			},
		},
		// warning: this config is applied to every node in the cluster!
		KubeletConfiguration: kubeletconf.KubeletConfiguration{
			TypeMeta: v1.TypeMeta{
				APIVersion: kubeletconf.SchemeGroupVersion.String(),
				Kind:       "KubeletConfiguration",
			},
			CgroupDriver: "systemd",
		},
	}
}
