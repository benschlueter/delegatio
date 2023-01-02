package kubernetes

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	coreAPI "k8s.io/api/core/v1"

	appsAPI "k8s.io/api/apps/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type kubernetesClient struct {
	client kubernetes.Interface
	logger *zap.Logger
}

func (k *kubernetesClient) ListPods(ctx context.Context, namespace string) error {
	podList, err := k.client.CoreV1().Pods("kube-system").List(ctx, metaAPI.ListOptions{})
	if err != nil {
		return err
	}
	for _, v := range podList.Items {
		k.logger.Info(v.Name)
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

func (k *kubernetesClient) CreateChallengeStatefulSet(ctx context.Context, challengeNamespace, userID, pubKeyUser string) error {
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
									Name:      "ssh-pub-key-configmap-volume",
									MountPath: "/root/.ssh/authorized_keys",
									SubPath:   userID,
								},
							},
							Ports: []coreAPI.ContainerPort{
								{
									Name:          "ssh",
									Protocol:      coreAPI.ProtocolTCP,
									ContainerPort: 22,
								},
							},
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
				},
			},
		},
	}

	if err := k.CreateConfigMap(ctx, "ssh-pub-key-configmap", challengeNamespace); err != nil {
		return err
	}
	if err := k.AddDataToConfigMap(ctx, "ssh-pub-key-configmap", challengeNamespace, userID, pubKeyUser); err != nil {
		return err
	}
	if err := k.CreateService(ctx, challengeNamespace, userID, "22"); err != nil {
		return err
	}
	if err := k.CreateIngress(ctx, challengeNamespace, userID); err != nil {
		return err
	}
	_, err := k.client.AppsV1().StatefulSets(challengeNamespace).Create(ctx, &sSet, metaAPI.CreateOptions{})
	return err
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
func NewK8sClient(kubeconfigPath string, logger *zap.Logger) (*kubernetesClient, error) {
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
	return &kubernetesClient{
		client: client,
		logger: logger,
	}, nil
}
