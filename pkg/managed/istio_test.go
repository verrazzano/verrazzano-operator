// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	istio "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	istio2 "istio.io/api/networking/v1alpha3"

	"github.com/stretchr/testify/assert"

	v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
)

func TestCreateDestinationRules(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// create the destination rules
	err := createDestinationRules(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal("can't create destination rules")
	}
	assertCreateDestinationRules(t, clusterConnection, 9000)

	// change the coherence extend port in the model
	modelBindingPair.Model.Spec.CoherenceClusters[0].Ports[0].Port = 9001
	// create the destination rules
	err = createDestinationRules(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal("can't create destination rules")
	}
	assertCreateDestinationRules(t, clusterConnection, 9001)
}

func assertCreateDestinationRules(t *testing.T, clusterConnection *util.ManagedClusterConnection, cohPort int) {
	// test-destination-rule
	list, err := clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal("expected to find DestinationRules in test namespace")
	}
	assert.Equal(t, 1, len(list.Items), "should only be 1 DestinationRule in test namespace")
	rule, err := clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test").Get(context.TODO(), "test-destination-rule", metav1.GetOptions{})
	if err != nil {
		t.Fatal("expected to find DestinationRule test-destination-rule")
	}
	assert.Equal(t, "test", rule.Namespace)
	assert.Equal(t, istio2.ClientTLSSettings_ISTIO_MUTUAL, rule.Spec.TrafficPolicy.Tls.Mode)
	assert.Equal(t, 1, len(rule.Spec.TrafficPolicy.PortLevelSettings))
	assert.Equal(t, istio2.ClientTLSSettings_DISABLE, rule.Spec.TrafficPolicy.PortLevelSettings[0].Tls.Mode)
	assert.Equal(t, cohPort, int(rule.Spec.TrafficPolicy.PortLevelSettings[0].Port.Number))

	// test2-destination-rule
	list, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test2").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal("expected to find DestinationRules in test2 namespace")
	}
	assert.Equal(t, 1, len(list.Items), "should only be 1 DestinationRule in test2 namespace")
	rule, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test2").Get(context.TODO(), "test2-destination-rule", metav1.GetOptions{})
	if err != nil {
		t.Fatal("expected to find DestinationRule test2-destination-rule")
	}
	assert.Equal(t, "test2", rule.Namespace)
	assert.Equal(t, istio2.ClientTLSSettings_ISTIO_MUTUAL, rule.Spec.TrafficPolicy.Tls.Mode)
	assert.Nil(t, rule.Spec.TrafficPolicy.PortLevelSettings)

	// test3-destination-rule
	list, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test3").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal("expected to find DestinationRules in test2 namespace")
	}
	assert.Equal(t, 1, len(list.Items), "should only be 1 DestinationRule in test3 namespace")
	rule, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test3").Get(context.TODO(), "test3-destination-rule", metav1.GetOptions{})
	if err != nil {
		t.Fatal("expected to find DestinationRule test3-destination-rule")
	}
	assert.Equal(t, "test3", rule.Namespace)
	assert.Equal(t, istio2.ClientTLSSettings_ISTIO_MUTUAL, rule.Spec.TrafficPolicy.Tls.Mode)
	assert.Nil(t, rule.Spec.TrafficPolicy.PortLevelSettings)
}

func TestCreateAuthorizationPolicies(t *testing.T) {
	clusterConnections := getManagedClusterConnections()
	modelBindingPair := getModelBindingPair()
	clusterConnection := clusterConnections["cluster1"]

	// create the authorization policies
	err := createAuthorizationPolicies(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal("can't create authorization policies")
	}
	assertCreateAuthorizationPolicies(t, clusterConnection, true)

	// change the WebLogic domain connections in the model
	modelBindingPair.Model.Spec.WeblogicDomains[0].Connections = nil

	// recreate the authorization policies
	err = createAuthorizationPolicies(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(" should not raise an error")
	}
	assertCreateAuthorizationPolicies(t, clusterConnection, false)
}

func assertCreateAuthorizationPolicies(t *testing.T, clusterConnection *util.ManagedClusterConnection, wlsHasConn bool) {
  
	// test-authorization-policy
	list, err := clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal("expected to find AuthorizationPolicies in test namespace")
	}
	assert.Equal(t, 1, len(list.Items), "should only be 1 AuthorizationPolicy in test namespace")
	policy, err := clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test").Get(context.TODO(), "test-authorization-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatal("expected to find AuthorizationPolicy test-authorization-policy")
	}
	assert.Equal(t, "test", policy.Namespace)
	assert.Equal(t, 1, len(policy.Spec.Rules))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
	assert.Equal(t, "test", policy.Spec.Rules[0].From[0].Source.Namespaces[0])
	assert.Equal(t, "istio-system", policy.Spec.Rules[0].From[0].Source.Namespaces[1])

	// test2-authorization-policy
	list, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test2").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal("expected to find AuthorizationPolicies in test2 namespace")
	}
	assert.Equal(t, 1, len(list.Items), "should only be 1 AuthorizationPolicy in test2 namespace")
	policy, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test2").Get(context.TODO(), "test2-authorization-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatal("expected to find AuthorizationPolicy test2-authorization-policy")
	}
	assert.Equal(t, "test2", policy.Namespace)
	assert.Equal(t, 1, len(policy.Spec.Rules))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From))
	if wlsHasConn {
		assert.Equal(t, 3, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
		assert.Equal(t, "test2", policy.Spec.Rules[0].From[0].Source.Namespaces[0])
		assert.Equal(t, "test3", policy.Spec.Rules[0].From[0].Source.Namespaces[1])
		assert.Equal(t, "istio-system", policy.Spec.Rules[0].From[0].Source.Namespaces[2])
	} else {
		assert.Equal(t, 2, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
		assert.Equal(t, "test2", policy.Spec.Rules[0].From[0].Source.Namespaces[0])
		assert.Equal(t, "istio-system", policy.Spec.Rules[0].From[0].Source.Namespaces[1])
	}

	// test3-authorization-policy
	list, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test3").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal("expected to find AuthorizationPolicies in test3 namespace")
	}
	assert.Equal(t, 1, len(list.Items), "should only be 1 AuthorizationPolicy in test3 namespace")
	policy, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test3").Get(context.TODO(), "test3-authorization-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatal("expected find AuthorizationPolicy test3-authorization-policy")
	}
	assert.Equal(t, "test3", policy.Namespace)
	assert.Equal(t, 1, len(policy.Spec.Rules))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From))
	assert.Equal(t, 3, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
	assert.Equal(t, "test3", policy.Spec.Rules[0].From[0].Source.Namespaces[0])
	assert.Equal(t, "test", policy.Spec.Rules[0].From[0].Source.Namespaces[1])
	assert.Equal(t, "istio-system", policy.Spec.Rules[0].From[0].Source.Namespaces[2])
}

func TestNewIngresses(t *testing.T) {
	ingressName := "bobs-ingress"
	namespace := "bob"
	var binding v8o.VerrazzanoBinding
	binding.Name = "bobs-books-binding1"
	binding.Spec = v8o.VerrazzanoBindingSpec{}
	binding.Spec.IngressBindings = []v8o.VerrazzanoIngressBinding{{
		Name:    ingressName,
		DnsName: "*",
	}}
	var mc types.ManagedCluster
	mc.Name = "bobs-managed-1"
	destHost := "bobs-bookstore-cluster-cluster-1.bob.svc.cluster.local"
	domainName := "bobs-bookstore"
	uriPrefix := "/bobs-bookstore-order-manager"
	mc.Ingresses = map[string][]*types.Ingress{namespace: {{
		Name: ingressName,
		Destination: []*types.IngressDestination{
			&types.IngressDestination{
				Host:       destHost,
				Port:       8001,
				DomainName: domainName,
				Match: []types.MatchRequest{
					{Uri: map[string]string{"prefix": uriPrefix}},
				},
			},
		},
	}}} //, &types.Ingress{Name: "bobbys-ingress"}}
	gw, vs := newIngresses(&binding, &mc)
	assert.Equal(t, 1, len(gw), "Expected 1 Gateway")
	assert.Equal(t, "bobs-ingress-gateway", gw[0].Name, "Gateway Name")
	assert.Equal(t, namespace, gw[0].Namespace, "gateway Namespace")
	assert.Equal(t, "ingressgateway", gw[0].Spec.Selector["istio"], "gateway Selector")
	assert.Equal(t, 1, len(gw[0].Spec.Servers), "Expected 1 Gateway Server")
	gsvr := gw[0].Spec.Servers[0]
	assert.Equal(t, 1, len(gsvr.Hosts), "Expected 1 Gateway Server Host")
	assert.Equal(t, "*", gsvr.Hosts[0], "Expected Gateway Server Host")
	assert.Equal(t, uint32(80), gsvr.Port.Number, "Expected Gateway Server Port.Number")
	assert.Equal(t, "HTTP", gsvr.Port.Protocol, "Expected Gateway Server Port.Protocol")

	assert.Equal(t, 1, len(vs), "Expected 1 VirtualService")
	assert.Equal(t, namespace, vs[0].Namespace, "VirtualService Namespace")
	assert.Equal(t, "bobs-ingress-virtualservice", vs[0].Name, "VirtualService Name")
	assert.Equal(t, 1, len(vs[0].Spec.Gateways), "Expected 1 VirtualService.Gateways")
	assert.Equal(t, 1, len(vs[0].Spec.Hosts), "Expected 1 VirtualService.Hosts")
	assert.Equal(t, 2, len(vs[0].Spec.Http), "Expected 2 VirtualService.HttpRoute")
	assertMatch(t, vs[0].Spec.Http[0].Match, Pair{k: "prefix", v: uriPrefix})
	assertMatch(t, vs[0].Spec.Http[1].Match, Pair{k: "prefix", v: "/console"})
	assertRoute(t, vs[0].Spec.Http[0].Route,
		Dest{Port: 8001, Host: destHost})
	assertRoute(t, vs[0].Spec.Http[1].Route,
		Dest{Port: 7001, Host: "bobs-bookstore-adminserver.bob.svc.cluster.local"})
	assert.Equal(t, "bobs-ingress-gateway", vs[0].Spec.Gateways[0], "Expected VirtualService.Gateway")
	assert.Equal(t, "*", vs[0].Spec.Hosts[0], "Expected VirtualService.Host")
}

func TestSockshopVirtualService(t *testing.T) {
	ingressName := "sockshop-ingress"
	namespace := "sockshop"
	var binding v8o.VerrazzanoBinding
	binding.Name = "sockshop-binding"
	binding.Spec = v8o.VerrazzanoBindingSpec{}
	binding.Spec.IngressBindings = []v8o.VerrazzanoIngressBinding{{
		Name: ingressName, DnsName: "*",
	}}
	var mc types.ManagedCluster
	mc.Name = "sockshop-managed-1"
	destHost := "sockshop-cluster-cluster-1.sockshop.svc.cluster.local"
	p := []Pair{
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
	mc.Ingresses = map[string][]*types.Ingress{namespace: []*types.Ingress{&types.Ingress{
		Name: ingressName,
		Destination: []*types.IngressDestination{
			&types.IngressDestination{
				Host: destHost,
				Port: 8001,
				Match: []types.MatchRequest{
					{Uri: map[string]string{p[0].k: p[0].v}},
					{Uri: map[string]string{p[1].k: p[1].v}},
					{Uri: map[string]string{p[2].k: p[2].v}},
					{Uri: map[string]string{p[3].k: p[3].v}},
					{Uri: map[string]string{p[4].k: p[4].v}},
					{Uri: map[string]string{p[5].k: p[5].v}},
					{Uri: map[string]string{p[6].k: p[6].v}},
					{Uri: map[string]string{p[7].k: p[7].v}},
					{Uri: map[string]string{p[8].k: p[8].v}},
					{Uri: map[string]string{p[9].k: p[9].v}},
				},
			},
		},
	}}}
	gw, vs := newIngresses(&binding, &mc)
	gsvr := gw[0].Spec.Servers[0]
	assert.Equal(t, 1, len(gsvr.Hosts), "Expected 1 Gateway Server Host")
	assert.Equal(t, 1, len(vs), "Expected 1 VirtualService")
	assert.Equal(t, namespace, vs[0].Namespace, "VirtualService Namespace")
	assert.Equal(t, "sockshop-ingress-virtualservice", vs[0].Name, "VirtualService Name")
	assert.Equal(t, 1, len(vs[0].Spec.Gateways), "Expected 1 VirtualService.Gateways")
	assert.Equal(t, 1, len(vs[0].Spec.Hosts), "Expected 1 VirtualService.Hosts")
	assert.Equal(t, 1, len(vs[0].Spec.Http), "Expected 1 VirtualService.HttpRoute")
	assertMatch(t, vs[0].Spec.Http[0].Match, p...)
	assertRoute(t, vs[0].Spec.Http[0].Route, Dest{Port: 8001, Host: destHost})
	assert.Equal(t, "sockshop-ingress-gateway", vs[0].Spec.Gateways[0], "Expected VirtualService.Gateway")
	assert.Equal(t, "*", vs[0].Spec.Hosts[0], "Expected VirtualService.Host")
	//yaml, _ := util.ToYmal(*vs[0])
	t.Log("VirtualService", len(vs)) //string(yaml))
}

func assertMatch(t *testing.T, match []istio.MatchRequest, expected ...Pair) {
	size := len(expected)
	assert.Equal(t, size, len(match), fmt.Sprintf("Expected %v HttpMatch", size))
	for i, pair := range expected {
		uri := match[i].Uri[pair.k]
		assert.Equal(t, pair.v, uri, fmt.Sprintf("Expected match %v: %v", pair.k, pair.v))
	}
}

type Pair struct {
	k string
	v string
}

type Dest struct {
	Host string
	Port int
}

func assertRoute(t *testing.T, dest []istio.HTTPRouteDestination, expected ...Dest) {
	size := len(expected)
	assert.Equal(t, size, len(dest), fmt.Sprintf("Expected %v HTTPRouteDestination", size))
	for i := range expected {
		assert.Equal(t, expected[i].Port, dest[i].Destination.Port.Number, "Expected Destination.Port.Number")
		assert.Equal(t, expected[i].Host, dest[i].Destination.Host, "Expected Destination.Host")
	}
}
