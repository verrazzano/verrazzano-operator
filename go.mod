module github.com/verrazzano/verrazzano-operator

require (
	github.com/Jeffail/gabs/v2 v2.3.0
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cznic/lex v0.0.0-20181122101858-ce0fb5e9bb1b // indirect
	github.com/cznic/lexer v0.0.0-20181122101858-e884d4bd112e // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/ianlancetaylor/demangle v0.0.0-20200715173712-053cf528c12f // indirect
	github.com/jteeuwen/go-bindata v3.0.7+incompatible // indirect
	github.com/kylelemons/godebug v1.1.0
	github.com/lyft/protoc-gen-star v0.4.15 // indirect
	github.com/neelance/astrewrite v0.0.0-20160511093645-99348263ae86 // indirect
	github.com/neelance/sourcemap v0.0.0-20200213170602-2833bce08e4c // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/verrazzano/verrazzano-coh-cluster-operator v0.0.7
	github.com/verrazzano/verrazzano-crd-generator v0.3.29
	github.com/verrazzano/verrazzano-helidon-app-operator v0.0.6
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.16
	github.com/verrazzano/verrazzano-wko-operator v0.0.6
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
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
