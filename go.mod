module github.com/verrazzano/verrazzano-operator

require (
	github.com/Jeffail/gabs/v2 v2.3.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/jteeuwen/go-bindata v3.0.7+incompatible // indirect
	github.com/kylelemons/godebug v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/verrazzano/verrazzano-coh-cluster-operator v0.0.6-1
	github.com/verrazzano/verrazzano-crd-generator v0.3.28-1
	github.com/verrazzano/verrazzano-helidon-app-operator v0.0.5-1
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.14-2
	github.com/verrazzano/verrazzano-wko-operator v0.0.4-1
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	gopkg.in/yaml.v2 v2.3.0
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
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.5
	k8s.io/client-go => k8s.io/client-go v0.18.2
)

go 1.13
