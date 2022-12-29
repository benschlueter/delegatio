package kubernetes

import (
	"context"
	"fmt"

	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type kubernetesClient struct {
	client kubernetes.Interface
}

func (k *kubernetesClient) CreateSSHConfigMap(ctx context.Context, namespace string) error {
	cm := coreAPI.ConfigMap{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      "ssh-pub-key-configmap",
			Namespace: namespace,
		},
		Data: map[string]string{
			"dummyuser": "testfiledata",
		},
	}
	_, err := k.client.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metaAPI.CreateOptions{})
	return err
}

func (k *kubernetesClient) ListPods(ctx context.Context, namespace string) error {
	podList, err := k.client.CoreV1().Pods("kube-system").List(ctx, metaAPI.ListOptions{})
	if err != nil {
		return err
	}
	for _, v := range podList.Items {
		fmt.Println(v.Name, v.Status.PodIPs)
	}
	return nil
}

func (k *kubernetesClient) CreateNamespace(ctx context.Context, namespace string) error {
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

func (k *kubernetesClient) CreatePod(ctx context.Context, challengeNamespace, userID string) error {
	pod := coreAPI.Pod{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "Pod",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      userID,
			Namespace: challengeNamespace,
		},
		Spec: coreAPI.PodSpec{
			Containers: []coreAPI.Container{
				{
					Name:  "archlinux-container",
					Image: "archlinux:latest",
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
							Name:      "ssh-pub-key-configmap-volume",
							MountPath: "/root/.ssh/authorized_keys",
							SubPath:   "dummyuser",
						},
					},
				},
			},
			Volumes: []coreAPI.Volume{
				{
					Name: "ssh-pub-key-configmap-volume",
					VolumeSource: coreAPI.VolumeSource{
						ConfigMap: &coreAPI.ConfigMapVolumeSource{
							LocalObjectReference: coreAPI.LocalObjectReference{
								Name: "ssh-pub-key-configmap",
							},
						},
					},
				},
			},
			// NodeSelector: ,
		},
	}
	if err := k.CreateSSHConfigMap(ctx, challengeNamespace); err != nil {
		return err
	}
	_, err := k.client.CoreV1().Pods(challengeNamespace).Create(ctx, &pod, metaAPI.CreateOptions{})
	return err
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
func NewK8sClient(kubeconfigPath string) (*kubernetesClient, error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	// create the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &kubernetesClient{client: client}, nil
}
