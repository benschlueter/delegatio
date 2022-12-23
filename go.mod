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
	github.com/edgelesssys/constellation v0.0.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	go.uber.org/multierr v1.8.0
	go.uber.org/zap v1.23.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
	k8s.io/api v0.25.3
	k8s.io/client-go v0.25.3
	k8s.io/kubernetes v1.25.4
	libvirt.org/go/libvirt v1.8009.0
	libvirt.org/go/libvirtxml v1.8009.0
)

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apimachinery v0.25.3 // indirect
	k8s.io/cluster-bootstrap v0.0.0 // indirect
	k8s.io/component-base v0.25.3 // indirect
	k8s.io/klog/v2 v2.70.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
