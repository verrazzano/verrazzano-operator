module github.com/verrazzano/verrazzano-operator

require (
	github.com/Jeffail/gabs/v2 v2.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gordonklaus/ineffassign v0.0.0-20200809085317-e36bfde3bb78 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/kylelemons/godebug v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/verrazzano/verrazzano-coh-cluster-operator v0.0.10
	github.com/verrazzano/verrazzano-crd-generator v0.3.34
	github.com/verrazzano/verrazzano-helidon-app-operator v0.0.9
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.26-0.20201008210424-c475e6d327a2
	github.com/verrazzano/verrazzano-wko-operator v0.0.13
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/tools v0.0.0-20201013053347-2db1cd791039 // indirect
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
