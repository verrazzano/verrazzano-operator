module github.com/verrazzano/verrazzano-operator

require (
	github.com/Jeffail/gabs/v2 v2.3.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/kylelemons/godebug v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/verrazzano/verrazzano-coh-cluster-operator v0.0.6
	github.com/verrazzano/verrazzano-crd-generator v0.3.27
	github.com/verrazzano/verrazzano-helidon-app-operator v0.0.5
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.12
	github.com/verrazzano/verrazzano-wko-operator v0.0.4
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	gopkg.in/yaml.v2 v2.3.0
	istio.io/api v0.0.0-20200107183329-ed4b507c54e1
	istio.io/client-go v0.0.0-20200107185429-9053b0f86b03
	k8s.io/api v0.15.7
	k8s.io/apiextensions-apiserver v0.15.7
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/yaml v1.2.0
)

// Pinned to kubernetes-0.15.7
replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.5
	k8s.io/api => k8s.io/api v0.15.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.15.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.15.7
	k8s.io/apiserver => k8s.io/apiserver v0.15.7
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.15.7
	k8s.io/client-go => k8s.io/client-go v0.15.7
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.15.7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.15.7
	k8s.io/code-generator => k8s.io/code-generator v0.15.7
	k8s.io/component-base => k8s.io/component-base v0.15.7
	k8s.io/cri-api => k8s.io/cri-api v0.15.7
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.15.7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.15.7
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.15.7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.15.7
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.15.7
	k8s.io/kubelet => k8s.io/kubelet v0.15.7
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.15.7
	k8s.io/metrics => k8s.io/metrics v0.15.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.15.7
)

go 1.13
