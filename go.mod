module github.com/verrazzano/verrazzano-operator

require (
	github.com/gorilla/mux v1.7.3
	github.com/kylelemons/godebug v1.1.0
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/verrazzano/verrazzano-coh-cluster-operator v0.0.0-20201112155446-709d712760e1
	github.com/verrazzano/verrazzano-crd-generator v0.0.0-20201214161122-0330d094db41
	github.com/verrazzano/verrazzano-helidon-app-operator v0.0.0-20201214161020-501df7fe4ccc
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.0-20201110193547-32e100ccdb72
	github.com/verrazzano/verrazzano-wko-operator v0.0.0-20201112153813-48bbb61a1abb
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/mod v0.4.1 // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	golang.org/x/tools v0.1.0 // indirect
	istio.io/client-go v0.0.0-20200630182733-fd3f873f3f52
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	k8s.io/client-go => k8s.io/client-go v0.18.2
)

go 1.13
