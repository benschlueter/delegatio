module github.com/benschlueter/delegatio

go 1.19

replace (
	k8s.io/api v0.0.0 => k8s.io/api v0.25.3
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.25.3
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.25.3
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.25.3
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.25.3
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.25.3
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.25.3
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.25.3
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.25.3
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.25.3
	k8s.io/component-helpers v0.0.0 => k8s.io/component-helpers v0.25.3
	k8s.io/controller-manager v0.0.0 => k8s.io/controller-manager v0.25.3
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.25.3
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.25.3
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.25.3
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.25.3
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.25.3
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.25.3
	k8s.io/kubectl v0.0.0 => k8s.io/kubectl v0.25.3
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.25.3
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.25.3
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.25.3
	k8s.io/mount-utils v0.0.0 => k8s.io/mount-utils v0.25.3
	k8s.io/pod-security-admission v0.0.0 => k8s.io/pod-security-admission v0.25.3
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.25.3
)

require (
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	go.uber.org/multierr v1.8.0
	go.uber.org/zap v1.23.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	k8s.io/kubernetes v1.25.4
	libvirt.org/go/libvirt v1.8009.0
	libvirt.org/go/libvirtxml v1.8009.0
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.25.3 // indirect
	k8s.io/apimachinery v0.25.3 // indirect
	k8s.io/cluster-bootstrap v0.0.0 // indirect
	k8s.io/component-base v0.0.0 // indirect
	k8s.io/klog/v2 v2.70.1 // indirect
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
