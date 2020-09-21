// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"

	"github.com/stretchr/testify/assert"
	v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	wls "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	v1helidonapp "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	. "github.com/verrazzano/verrazzano-operator/test/integ/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestProcessIngressConnections tests the processing of ingress connections in a ModelBindingPair.
func TestProcessIngressConnections(t *testing.T) {
	ingressName := "bobbys-ingress"
	namespace := "bobby"
	mc := types.ManagedCluster{Ingresses: make(map[string][]*types.Ingress)}
	connections := []v8o.VerrazzanoConnections{{
		Ingress: []v8o.VerrazzanoIngressConnection{{Name: ingressName}},
	}}
	var domainCR wls.Domain
	destinationHost := "bobbys.destinationHost"
	vsDestPort := 80
	ingressBindings := []v8o.VerrazzanoIngressBinding{{
		Name:    ingressName,
		DnsName: "*",
	}}
	processIngressConnections(&mc, connections, namespace, &domainCR, destinationHost, vsDestPort, &ingressBindings)
	assert.Equal(t, 1, len(mc.Ingresses[namespace]), "Expected 1 Ingress")
	assert.Equal(t, 1, len(mc.Ingresses[namespace][0].Destination), "Expected 1 Destination")
	dest := mc.Ingresses[namespace][0].Destination[0]

	ingressName = "bobs-ingress"
	namespace = "bob"
	mc = types.ManagedCluster{Ingresses: map[string][]*types.Ingress{
		namespace: {{Name: ingressName}},
	}}
	connections = []v8o.VerrazzanoConnections{{
		Ingress: []v8o.VerrazzanoIngressConnection{{Name: ingressName}},
	}}
	destinationHost = "bobs.destinationHost"
	vsDestPort = 80
	ingressBindings = []v8o.VerrazzanoIngressBinding{{
		Name:    ingressName,
		DnsName: "*",
	}}
	processIngressConnections(&mc, connections, namespace, &domainCR, destinationHost, vsDestPort, &ingressBindings)
	assert.Equal(t, 1, len(mc.Ingresses[namespace]), "Expected 1 Ingress")
	assert.Equal(t, 1, len(mc.Ingresses[namespace][0].Destination), "Expected 1 Destination")
	dest = mc.Ingresses[namespace][0].Destination[0]
	assert.Equal(t, 1, len(dest.Match), "Expected 1 matchRules")
}

// TestMultipleIngressBindings tests the processing of multiple ingress bindings
func TestMultipleIngressBindings(t *testing.T) {
	ingressName := "bobbys-ingress"
	namespace := "bobby"
	mc := types.ManagedCluster{Ingresses: make(map[string][]*types.Ingress)}
	connections := []v8o.VerrazzanoConnections{{
		Ingress: []v8o.VerrazzanoIngressConnection{{
			Name: ingressName,
			Match: []v8o.MatchRequest{
				{Uri: map[string]string{"exact": "/a-exact"}},
				{Uri: map[string]string{"prefix": "/a-prefix"}},
				{Uri: map[string]string{"prefix": "/bobbys-front-end"}},
				{Uri: map[string]string{"prefix": "/xyz"}},
				{Uri: map[string]string{"exact": "/exact"}},
			},
		}},
	}}
	var domainCR wls.Domain
	destinationHost := "bobbys.destinationHost"
	matchPort := 80
	ingressBindings := []v8o.VerrazzanoIngressBinding{{
		Name:    ingressName,
		DnsName: "*",
	}, {
		Name:    ingressName,
		DnsName: "*",
	}}
	processIngressConnections(&mc, connections, namespace, &domainCR, destinationHost, matchPort, &ingressBindings)
	assert.Equal(t, 1, len(mc.Ingresses[namespace]), "Expected 1 Ingress")
	assert.Equal(t, 1, len(mc.Ingresses[namespace][0].Destination), "Expected 1 Destination")
	dest := mc.Ingresses[namespace][0].Destination[0]
	assert.Equal(t, 5, len(dest.Match), "Expected 6 matchRules")
}

// TestSockShopIngressBindings tests the parsing of ingress bindings from a test model.
func TestSockShopIngressBindings(t *testing.T) {
	model, err := ReadModel("testdata/sockshop-model.yaml")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := ReadBinding("testdata/sockshop-binding.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ingressBindings := binding.Spec.IngressBindings
	assert.Equal(t, 2, len(ingressBindings), "Expected 2 IngressBinding's")
	cluster := "cluster1"
	namespace := "sockshop"
	vzUri := "Verrazzano.Uri"
	var optImagePullSecrets []corev1.LocalObjectReference
	pair := CreateModelBindingPair(model, binding, vzUri, optImagePullSecrets)

	validateIngressBindings(t, pair, cluster, namespace, "wl-frontend-cluster-cluster-1.sockshop.svc.cluster.local", 8001)
}

func validateIngressBindings(t *testing.T, mbp *types.ModelBindingPair, cluster string, namespace string,
	exptectedWlsHost string, expectedWlsPort int) {
	assert.Equal(t, 2, len(mbp.ManagedClusters[cluster].Ingresses[namespace]), "Expected 1 Ingress")
	frontendIngress := getIngress(t, cluster, namespace, "sockshop-frontend-ingress", mbp)
	assert.Equal(t, 1, len(frontendIngress.Destination), "Expected 1 IngressDestination")
	assert.Equal(t, 8088, frontendIngress.Destination[0].Port, "Expected IngressDestination Port")
	wlIngress := getIngress(t, cluster, namespace, "wl-ingress", mbp)
	assert.Equal(t, 1, len(wlIngress.Destination), "Expected 1 IngressDestination")
	assert.Equal(t, exptectedWlsHost, wlIngress.Destination[0].Host, "Expected IngressDestination Host")
	assert.Equal(t, expectedWlsPort, wlIngress.Destination[0].Port, "Expected IngressDestination Port")
	userApp := getVerrazzanoHelidon(t, "user", mbp)
	assert.Equal(t, uint(80), userApp.Port, "Expected user Port")
	assert.Equal(t, uint(7001), userApp.TargetPort, "Expected user targetPort")
	expectedMatch := []KvPair{
		{k: "exact", v: "/"},
		{k: "exact", v: "/cart"},
		{k: "prefix", v: "/cart"},
		{k: "exact", v: "/catalogue"},
		{k: "exact", v: "/login"},
		{k: "prefix", v: "/catalogue"},
		{k: "prefix", v: "/css"},
		{k: "prefix", v: "/js"},
		{k: "prefix", v: "/img"},
		{k: "regex", v: "^.*\\.(ico|png|jpg|html)$"},
	}
	assertMatch(t, frontendIngress.Destination[0].Match, expectedMatch...)
	assert.Equal(t, 8, len(mbp.ManagedClusters[cluster].HelidonApps), "Expected 8 HelidonApps")
	assertPorts(t, mbp, cluster, namespace, "frontend", 8088, 8079)
	assertPorts(t, mbp, cluster, namespace, "carts", 8080, 8080)
	assertPorts(t, mbp, cluster, namespace, "catalogue", 8080, 8080)
	assertPorts(t, mbp, cluster, namespace, "user", 80, 7001)
}

func TestSockShopSimpleModelBinding(t *testing.T) {
	model, err := ReadModel("testdata/sockshop-simple-model.yaml")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := ReadBinding("testdata/sockshop-simple-binding.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cluster := "local"
	namespace := "sockshop"
	vzUri := "Verrazzano.Uri"
	var optImagePullSecrets []corev1.LocalObjectReference
	pair := CreateModelBindingPair(model, binding, vzUri, optImagePullSecrets)

	assertPorts(t, pair, cluster, namespace, "frontend", 8080, 8079)
	assertPorts(t, pair, cluster, namespace, "carts", 80, 7001)
	assertPorts(t, pair, cluster, namespace, "user", 80, 7001)
	assertPorts(t, pair, cluster, namespace, "catalogue", 80, 7001)
	assertPorts(t, pair, cluster, namespace, "swagger", 80, 8080)

	//Test default ports
	var apps []v8o.VerrazzanoHelidon
	for _, helidon := range model.Spec.HelidonApplications {
		helidon.Port = 0
		helidon.TargetPort = 0
		apps = append(apps, helidon)
	}
	model.Spec.HelidonApplications = apps
	pair = CreateModelBindingPair(model, binding, vzUri, optImagePullSecrets)
	assertPorts(t, pair, cluster, namespace, "frontend", 8080, 8080)
	assertPorts(t, pair, cluster, namespace, "carts", 8080, 8080)
	assertPorts(t, pair, cluster, namespace, "user", 8080, 8080)
	assertPorts(t, pair, cluster, namespace, "swagger", 8080, 8080)

	for _, cluster := range pair.ManagedClusters {
		for _, secrets := range cluster.Secrets {
			assert.Contains(t, secrets, constants.VmiSecretName)
		}
	}
}

func assertPorts(t *testing.T, pair *types.ModelBindingPair,
	cluster, namespace, name string, port, targetPort int32) {
	app := getHelidonApp(t, cluster, name, pair)
	assert.Equal(t, name, app.Spec.Name,
		fmt.Sprintf("Expected HelidonApp %v", name))
	assert.Equal(t, port, app.Spec.Port,
		fmt.Sprintf("Expected %v port", name))
	assert.Equal(t, targetPort, app.Spec.TargetPort,
		fmt.Sprintf("Expected %v targetPort", name))
}

func getIngress(t *testing.T, cluster, namespace, name string, pair *types.ModelBindingPair) *types.Ingress {
	for _, i := range pair.ManagedClusters[cluster].Ingresses[namespace] {
		if i.Name == name {
			return i
		}
	}
	t.Fatalf("Ingress %v not found", name)
	return &types.Ingress{}
}

func getVerrazzanoHelidon(t *testing.T, name string, pair *types.ModelBindingPair) v8o.VerrazzanoHelidon {
	for _, app := range pair.Model.Spec.HelidonApplications {
		if app.Name == name {
			return app
		}
	}
	t.Fatalf("VerrazzanoHelidon %v not found", name)
	return v8o.VerrazzanoHelidon{}
}

func getHelidonApp(t *testing.T, cluster, name string, pair *types.ModelBindingPair) *v1helidonapp.HelidonApp {
	for _, app := range pair.ManagedClusters[cluster].HelidonApps {
		if app.Name == name {
			return app
		}
	}
	t.Fatalf("VerrazzanoHelidon %v not found", name)
	return &v1helidonapp.HelidonApp{}
}

type KvPair struct {
	k string
	v string
}

func assertMatch(t *testing.T, match []types.MatchRequest, expected ...KvPair) {
	size := len(expected)
	assert.Equal(t, size, len(match), fmt.Sprintf("Expected %v HttpMatch", size))
	for i, pair := range expected {
		uri := match[i].Uri[pair.k]
		assert.Equal(t, pair.v, uri, fmt.Sprintf("Expected match %v: %v", pair.k, pair.v))
	}
}

// TestCreateModelBindingPair tests the creation of a ModelBindingPair.
// A test model and binding located in testdata are used to create a model and a binding.
// A new ModelBindingPair is created using the above test model/binding.
// Assertions are then made on the fields of the ModelBindingPair.
func TestCreateModelBindingPair(t *testing.T) {
	model, _ := ReadModel("testdata/sockshop-model.yaml")
	binding, _ := ReadBinding("testdata/sockshop-binding.yaml")

	assert.NotEmpty(t, model.Name)
	assert.NotEmpty(t, binding.Name)

	// create a new ModelBindingPair
	var optImagePullSecrets []corev1.LocalObjectReference
	mbp := CreateModelBindingPair(model, binding, "/my/verrazzano/url", optImagePullSecrets)

	// gather expected state
	expectedClusterHelidonApps := map[string]map[string]struct{}{"cluster1": {"frontend": {}, "carts": {},
		"catalogue": {}, "orders": {}, "payment": {}, "shipping": {}, "user": {}, "swagger": {}}}
	expectedClusterNamespaces := map[string]map[string]struct{}{"cluster1": {"verrazzano-system": {}, "sockshop": {}, "verrazzano-sock-shop-binding": {}}}
	wlsDomain := &wls.Domain{ObjectMeta: metav1.ObjectMeta{Name: "wl-frontend", Namespace: "sockshop"},
		Spec: wls.DomainSpec{Image: util.GetTestWlsFrontendImage(), LogHome: "/u01/oracle/user_projects/domains/wl-frontend/logs"}}

	expectedValues := MbpExpectedValues{
		Binding:          binding,
		Model:            model,
		HelidonApps:      expectedClusterHelidonApps,
		Namespaces:       expectedClusterNamespaces,
		WlsDomains:       map[string]map[string]*wls.Domain{"cluster1": {"wl-frontend": wlsDomain}},
		Uri:              "/my/verrazzano/url",
		ImagePullSecrets: optImagePullSecrets,
	}
	// validate the returned mbp
	validateModelBindingPair(t, mbp, expectedValues)
}

func TestCreateModelBindingPairNoCluster(t *testing.T) {
	model, err := ReadModel("testdata/sockshop-model-no-cluster.yaml")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := ReadBinding("testdata/sockshop-binding.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ingressBindings := binding.Spec.IngressBindings
	assert.Equal(t, 2, len(ingressBindings), "Expected 2 IngressBinding's")
	cluster := "cluster1"
	namespace := "sockshop"
	vzUri := "/my/verrazzano/url"
	optImagePullSecrets := []corev1.LocalObjectReference{
		{
			Name: "testSecret",
		},
	}

	mbp := CreateModelBindingPair(model, binding, vzUri, optImagePullSecrets)

	// gather expected state
	expectedClusterHelidonApps := map[string]map[string]struct{}{"cluster1": {"frontend": {}, "carts": {},
		"catalogue": {}, "orders": {}, "payment": {}, "shipping": {}, "user": {}, "swagger": {}}}
	expectedClusterNamespaces := map[string]map[string]struct{}{"cluster1": {"verrazzano-system": {}, "sockshop": {}, "verrazzano-sock-shop-binding": {}}}
	wlsDomain := &wls.Domain{ObjectMeta: metav1.ObjectMeta{Name: "wl-frontend", Namespace: "sockshop"},
		Spec: wls.DomainSpec{Image: util.GetTestWlsFrontendImage(), LogHome: "/u01/oracle/user_projects/domains/wl-frontend/logs"}}

	expectedValues := MbpExpectedValues{
		Binding:          binding,
		Model:            model,
		HelidonApps:      expectedClusterHelidonApps,
		Namespaces:       expectedClusterNamespaces,
		WlsDomains:       map[string]map[string]*wls.Domain{"cluster1": {"wl-frontend": wlsDomain}},
		Uri:              "/my/verrazzano/url",
		ImagePullSecrets: optImagePullSecrets,
	}

	validateModelBindingPair(t, mbp, expectedValues)

	validateDatabaseBindings(t, mbp, "AdminServer")

	validateIngressBindings(t, mbp, cluster, namespace, "wl-frontend-adminserver.sockshop.svc.cluster.local", 7001)
}

// TestUpdateModelBindingPair tests the updating of a ModelBindingPair.
// A test model and binding located in testdata are used to create a model and a binding.
// A new ModelBindingPair is created using the above test model/binding.
// A new model and binding are created from testdata to use as the updated state.
// The original ModelBindingPair is updated with the new test mode/binding.
// Assertions are then made on the fields of the ModelBindingPair.
func TestUpdateModelBindingPair(t *testing.T) {
	model, _ := ReadModel("testdata/sockshop-model.yaml")
	binding, _ := ReadBinding("testdata/sockshop-binding.yaml")
	model2, _ := ReadModel("testdata/sockshop-model-2.yaml")
	binding2, _ := ReadBinding("testdata/sockshop-binding-2.yaml")

	assert.NotEmpty(t, model.Name)
	assert.NotEmpty(t, binding.Name)
	assert.NotEmpty(t, model2.Name)
	assert.NotEmpty(t, binding2.Name)

	// create a new ModelBindingPair
	optImagePullSecrets := []corev1.LocalObjectReference{
		{
			Name: "testSecret",
		},
	}
	mbp := CreateModelBindingPair(model, binding, "/my/verrazzano/url", optImagePullSecrets)

	// gather expected state
	expectedClusterHelidonApps := map[string]map[string]struct{}{"cluster1": {"frontend": {}, "carts": {}, "catalogue": {},
		"orders": {}, "payment": {}, "shipping": {}, "user": {}, "swagger": {}}}
	expectedClusterNamespaces := map[string]map[string]struct{}{"cluster1": {"verrazzano-system": {}, "sockshop": {}, "verrazzano-sock-shop-binding": {}}}
	wlsDomain := &wls.Domain{ObjectMeta: metav1.ObjectMeta{Name: "wl-frontend", Namespace: "sockshop"},
		Spec: wls.DomainSpec{Image: util.GetTestWlsFrontendImage(), LogHome: "/u01/oracle/user_projects/domains/wl-frontend/logs"}}
	expectedValues := MbpExpectedValues{
		Binding:          binding,
		Model:            model,
		HelidonApps:      expectedClusterHelidonApps,
		Namespaces:       expectedClusterNamespaces,
		WlsDomains:       map[string]map[string]*wls.Domain{"cluster1": {"wl-frontend": wlsDomain}},
		Uri:              "/my/verrazzano/url",
		ImagePullSecrets: optImagePullSecrets,
	}
	// validate create mbp
	validateModelBindingPair(t, mbp, expectedValues)

	// invoke update function that we are testing
	optImagePullSecretsUpdated := []corev1.LocalObjectReference{
		{
			Name: "testSecretUpdated",
		},
	}
	UpdateModelBindingPair(mbp, model2, binding2, "/my/verrazzano/url/updated", optImagePullSecretsUpdated)

	// gather updated state
	expectedClusterHelidonApps = map[string]map[string]struct{}{"cluster1": {"frontend": {}, "orders": {}},
		"cluster2": {"frontend": {}, "foobar": {}}}
	expectedClusterNamespaces = map[string]map[string]struct{}{"cluster1": {"verrazzano-system": {}, "sockshop": {}, "verrazzano-": {}},
		"cluster2": {"verrazzano-system": {}, "sockshop2": {}}}

	expectedValues = MbpExpectedValues{
		Binding:          binding2,
		Model:            model2,
		HelidonApps:      expectedClusterHelidonApps,
		Namespaces:       expectedClusterNamespaces,
		WlsDomains:       map[string]map[string]*wls.Domain{},
		Uri:              "/my/verrazzano/url/updated",
		ImagePullSecrets: optImagePullSecretsUpdated,
	}
	// validate updated mbp
	validateModelBindingPair(t, mbp, expectedValues)
}

func TestDatabaseBindings(t *testing.T) {
	model, _ := ReadModel("testdata/sockshop-model.yaml")
	binding, _ := ReadBinding("testdata/sockshop-binding.yaml")

	// create a new ModelBindingPair
	var optImagePullSecrets []corev1.LocalObjectReference
	mbp := CreateModelBindingPair(model, binding, "/my/verrazzano/url", optImagePullSecrets)

	// validate the returned mbp
	validateDatabaseBindings(t, mbp, "cluster-1")
}

// MbpExpectedValues is a struct of expected ModelBindingPair state used in assertions
type MbpExpectedValues struct {
	Binding          *v8o.VerrazzanoBinding
	Model            *v8o.VerrazzanoModel
	HelidonApps      map[string]map[string]struct{}
	Namespaces       map[string]map[string]struct{}
	WlsDomains       map[string]map[string]*wls.Domain
	Uri              string
	ImagePullSecrets []corev1.LocalObjectReference
}

// validateModelBindingPair validates a ModelBindingPair
func validateModelBindingPair(t *testing.T,
	mbp *types.ModelBindingPair,
	expectedValues MbpExpectedValues) {

	assert.Equal(t, expectedValues.Uri, mbp.VerrazzanoUri)
	assert.Equal(t, expectedValues.ImagePullSecrets, mbp.ImagePullSecrets)
	assert.True(t, reflect.DeepEqual(expectedValues.ImagePullSecrets, mbp.ImagePullSecrets), "ImagePullSecrets should be equal")
	assert.True(t, reflect.DeepEqual(expectedValues.Model, mbp.Model), "Models should be equal")
	assert.True(t, reflect.DeepEqual(expectedValues.Binding, mbp.Binding), "Bindings should be equal")
	managedClusters := mbp.ManagedClusters
	assert.Equal(t, len(expectedValues.HelidonApps), len(managedClusters))

	for _, cluster := range managedClusters {
		expectedHelidonApps := expectedValues.HelidonApps[cluster.Name]
		assert.NotNil(t, expectedHelidonApps)
		assert.Equal(t, len(expectedHelidonApps), len(cluster.HelidonApps))
		for _, app := range cluster.HelidonApps {
			assert.Contains(t, expectedHelidonApps, app.Name)
		}

		assert.Equal(t, len(expectedValues.WlsDomains), len(cluster.WlsDomainCRs))
		for _, wlsDomain := range cluster.WlsDomainCRs {
			expectedDomain := expectedValues.WlsDomains[cluster.Name][wlsDomain.Name]
			assert.NotNil(t, expectedDomain, "Unexpected domain: "+wlsDomain.Name)
			assert.Equal(t, expectedDomain.Namespace, wlsDomain.Namespace)
			assert.Equal(t, expectedDomain.Spec.Image, wlsDomain.Spec.Image)
			//assert.Equal(t, expectedDomain.Spec.LogHome, wlsDomain.Spec.LogHome)
		}

		expectedNamespaces := expectedValues.Namespaces[cluster.Name]
		assert.NotNil(t, expectedNamespaces)
		for _, ns := range cluster.Namespaces {
			assert.Contains(t, expectedNamespaces, ns)
		}
		for _, secrets := range cluster.Secrets {
			assert.Contains(t, secrets, constants.VmiSecretName)
		}
	}
}

// validates that the datasource model config is created when databaseBindings are specified
func validateDatabaseBindings(t *testing.T, mbp *types.ModelBindingPair, expectedTarget string) {

	const datasourceConfigMap = `resources:
  JDBCSystemResource:
    socks:
      Target: '%s'
      JdbcResource:
        JDBCDataSourceParams:
          JNDIName: [
            jdbc/socks
          ]
        JDBCDriverParams:
          DriverName: com.mysql.cj.jdbc.Driver
          URL: '@@SECRET:mysqlsecret:url@@'
          PasswordEncrypted: '@@SECRET:mysqlsecret:password@@'
          Properties:
            user:
              Value: '@@SECRET:mysqlsecret:username@@'
        JDBCConnectionPoolParams:
          ConnectionReserveTimeoutSeconds: 10
          InitialCapacity: 0
          MaxCapacity: 5
          MinCapacity: 0
          TestConnectionsOnReserve: true
          TestTableName: SQL SELECT 1
    socks2:
      Target: '%s'
      JdbcResource:
        JDBCDataSourceParams:
          JNDIName: [
            jdbc/socks2
          ]
        JDBCDriverParams:
          DriverName: com.mysql.cj.jdbc.Driver
          URL: '@@SECRET:mysql2secret:url@@'
          PasswordEncrypted: '@@SECRET:mysql2secret:password@@'
          Properties:
            user:
              Value: '@@SECRET:mysql2secret:username@@'
        JDBCConnectionPoolParams:
          ConnectionReserveTimeoutSeconds: 10
          InitialCapacity: 0
          MaxCapacity: 5
          MinCapacity: 0
          TestConnectionsOnReserve: true
          TestTableName: SQL SELECT 1
`

	managedClusters := mbp.ManagedClusters
	for _, cluster := range managedClusters {
		// Look for the config map with the WDT datasource config map
		for _, configMap := range cluster.ConfigMaps {
			if configMap.Name == "wl-frontend-wdt-config-map" {
				for key, value := range configMap.Data {
					assert.Equal(t, "datasource.yaml", key)
					assert.Equal(t, fmt.Sprintf(datasourceConfigMap, expectedTarget, expectedTarget), value)
				}
				break
			}
		}
		// Check that configurations for WDT model-in-image are set on the domain custom resource
		for _, cluster := range cluster.WlsDomainCRs {
			assert.Equal(t, cluster.Spec.Configuration.Model.ConfigMap, "wl-frontend-wdt-config-map")
			assert.Contains(t, cluster.Spec.Configuration.Secrets, "mysqlsecret")
			assert.Contains(t, cluster.Spec.Configuration.Secrets, "mysql2secret")
			assert.Equal(t, cluster.Spec.Configuration.Model.RuntimeEncryptionSecret, "wl-frontend-runtime-encrypt-secret")
			assert.Equal(t, cluster.Spec.DomainHomeSourceType, "FromModel")
		}
	}
}
