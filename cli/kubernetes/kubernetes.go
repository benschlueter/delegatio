package kubernetes

import (
	"context"
	"fmt"

	"github.com/benschlueter/delegatio/cli/kubernetes/helm"
	"go.uber.org/zap"
	appsAPI "k8s.io/api/apps/v1"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is the struct used to access kubernetes helpers.
type Client struct {
	client     kubernetes.Interface
	logger     *zap.Logger
	restClient *rest.Config
}

// NewK8sClient returns a new kuberenetes client-go wrapper.
func NewK8sClient(kubeconfigPath string, logger *zap.Logger) (*Client, error) {
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
	return &Client{
		client:     client,
		logger:     logger,
		restClient: config,
	}, nil
}

// GetClient returns the kubernetes client.
func (k *Client) GetClient() kubernetes.Interface {
	return k.client
}

// CreateNamespace creates a namespace.
func (k *Client) CreateNamespace(ctx context.Context, namespace string) error {
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
							/* 							VolumeMounts: []coreAPI.VolumeMount{
								{
									Name:      "ssh-pub-key-configmap-volume",
									MountPath: "/root/.ssh/authorized_keys",
									SubPath:   userID,
								},
							}, */
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
					/* 					Volumes: []coreAPI.Volume{
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
					}, */
				},
			},
		},
	}

	_, err := k.client.AppsV1().StatefulSets(challengeNamespace).Create(ctx, &sSet, metaAPI.CreateOptions{})

	return err
}

// InstallHelmStuff installs cilium in the cluster.
func (k *Client) InstallHelmStuff(ctx context.Context) error {
	return helm.Install(ctx, k.logger.Named("helm"), "cilium")
}
