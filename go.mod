module github.com/verrazzano/verrazzano-operator

require (
	github.com/go-bindata/go-bindata/v3 v3.1.3
	github.com/gorilla/mux v1.7.3
	github.com/kylelemons/godebug v1.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/verrazzano/pkg v0.0.2
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.0-20210506182921-9c8e9e80ff10
	go.uber.org/zap v1.16.0
	k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	k8s.io/api => k8s.io/api v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.2
	k8s.io/client-go => k8s.io/client-go v0.19.2
)

go 1.13
