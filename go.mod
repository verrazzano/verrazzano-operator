module github.com/verrazzano/verrazzano-operator

require (
	github.com/Jeffail/gabs/v2 v2.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/kylelemons/godebug v1.1.0
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/verrazzano/verrazzano-coh-cluster-operator v0.0.0-20200921120302-d4040fa2f3c4
	github.com/verrazzano/verrazzano-crd-generator v0.0.0-20201022185955-0f25ee2b7108
	github.com/verrazzano/verrazzano-helidon-app-operator v0.0.0-20200921121606-448c29f0ce58
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.0-20200925113618-cc189235ad3d
	github.com/verrazzano/verrazzano-wko-operator v0.0.0-20201005195304-9238fa0e1f4a
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.2.8
	istio.io/api v0.0.0-20200629210345-933b83065c19
	istio.io/client-go v0.0.0-20200630182733-fd3f873f3f52
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/yaml v1.2.0
)

replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	k8s.io/client-go => k8s.io/client-go v0.18.2
)

go 1.13
