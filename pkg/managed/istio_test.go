// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/labels"
  "testing"

	istio "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	istio2 "istio.io/api/networking/v1alpha3"

	"github.com/stretchr/testify/assert"

	v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
)

func TestGetUniqueServiceEntryAddress(t *testing.T) {
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	var startIPIndex = 1

	address, err := getUniqueServiceEntryAddress(clusterConnection, &startIPIndex)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get unique ServiceEntry address: %v", err))
	}
	assert.Equal(t, "240.0.0.1", address)

	entry := istio.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: istio2.ServiceEntry{
			Addresses: []string{
				"240.0.0.1",
				"240.0.0.2",
				"240.0.0.3",
			},
		},
	}
	_, err = clusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Create(context.TODO(), &entry, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create ServiceEntry: %v", err))
	}

	address, err = getUniqueServiceEntryAddress(clusterConnection, &startIPIndex)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get unique ServiceEntry address: %v", err))
	}
	assert.Equal(t, "240.0.0.4", address)
}

func TestGetIstioGateways(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster2"]

	gatewayAddress := getIstioGateways(modelBindingPair, clusterConnections, "cluster2")

	assert.Equal(t, "123.45.0.1", gatewayAddress)

	service, err := clusterConnection.KubeClient.CoreV1().Services(IstioSystemNamespace).Get(context.TODO(), "istio-ingressgateway", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't delete istio-ingressgateway service: %v", err))
	}
	service.Status.LoadBalancer.Ingress = nil
	_, err = clusterConnection.KubeClient.CoreV1().Services(IstioSystemNamespace).Update(context.TODO(), service, metav1.UpdateOptions{})

	gatewayAddress = getIstioGateways(modelBindingPair, clusterConnections, "cluster2")
	assert.Equal(t, "", gatewayAddress)

	err = clusterConnection.KubeClient.CoreV1().Services(IstioSystemNamespace).Delete(context.TODO(), "istio-ingressgateway", metav1.DeleteOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't delete istio-ingressgateway service: %v", err))
	}
	gatewayAddress = getIstioGateways(modelBindingPair, clusterConnections, "cluster2")
	assert.Equal(t, "", gatewayAddress)
}

func TestCleanupOrphanedServiceEntries(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	clusterConnection2 := clusterConnections["cluster3"]

	se := istio.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "test",
		},
	}
	_, err := clusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Create(context.TODO(), &se, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service entry: %v", err))
	}
	_, err = clusterConnection2.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Create(context.TODO(), &se, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service entry: %v", err))
	}

	// clean up the service entries
	err = CleanupOrphanedServiceEntries(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't cleanup orphaned service entries: %v", err))
	}

  _, err = clusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Get(context.TODO(), "foo", metav1.GetOptions{})
  if err == nil {
    t.Fatal("expected service entry to be cleaned up")
  }
  _, err = clusterConnection2.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Get(context.TODO(), "foo", metav1.GetOptions{})
  if err == nil {
    t.Fatal("expected service entry to be cleaned up")
  }
}

func TestCleanupOrphanedIngresses(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	clusterConnection2 := clusterConnections["cluster3"]

	gw := istio.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "test",
		},
	}
	_, err := clusterConnection.IstioClientSet.NetworkingV1alpha3().Gateways("test").Create(context.TODO(), &gw, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create gateway: %v", err))
	}
	_, err = clusterConnection2.IstioClientSet.NetworkingV1alpha3().Gateways("test").Create(context.TODO(), &gw, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create gateway: %v", err))
	}

	vs := istio.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "test",
		},
	}
	_, err = clusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices("test").Create(context.TODO(), &vs, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create gateway: %v", err))
	}
	_, err = clusterConnection2.IstioClientSet.NetworkingV1alpha3().VirtualServices("test").Create(context.TODO(), &vs, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create gateway: %v", err))
	}

	// clean up the ingresses
	err = CleanupOrphanedIngresses(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't cleanup orphaned ingresses: %v", err))
	}

  _, err = clusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices("test").Get(context.TODO(), "foo", metav1.GetOptions{})
  if err == nil {
    t.Fatal("expected ingress to be cleaned up")
  }
  _, err = clusterConnection2.IstioClientSet.NetworkingV1alpha3().VirtualServices("test").Get(context.TODO(), "foo", metav1.GetOptions{})
  if err == nil {
    t.Fatal("expected ingress to be cleaned up")
  }
}

func TestCreateServiceEntries(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// create the service entries
	err := CreateServiceEntries(modelBindingPair, clusterConnections, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service entries: %v", err))
	}
	assertCreateServiceEntries(t, clusterConnection)

	// change the remote namespace
	modelBindingPair.ManagedClusters["cluster1"].RemoteRests["test"][0].RemoteNamespace = "test3"

	// create the service entries
	err = CreateServiceEntries(modelBindingPair, clusterConnections, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service entries: %v", err))
	}
	assertCreateServiceEntries(t, clusterConnection)

	// get the service entry created above and set the addresses to nil so that getUniqueServiceEntryAddress is called
	entry, err := clusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Get(context.TODO(), "test-remote", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service entry: %v", err))
	}
	entry.Spec.Addresses = nil
	_, err = clusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Update(context.TODO(), entry, metav1.UpdateOptions{})

	// create the service entries
	err = CreateServiceEntries(modelBindingPair, clusterConnections, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service entries: %v", err))
	}
	assertCreateServiceEntries(t, clusterConnection)

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName
	// create the service entries
	err = CreateServiceEntries(modelBindingPair, clusterConnections, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service entries: %v", err))
	}
}

func assertCreateServiceEntries(t *testing.T, clusterConnection *util.ManagedClusterConnection) {
  list, err := clusterConnection.IstioServiceEntryLister.ServiceEntries("test").List(labels.Everything())
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find ServiceEntries in test namespace: %v", err))
  }
  assert.Equal(t, 2, len(list), "should be 2 ServiceEntry in test namespace")

  entry, err := clusterConnection.IstioServiceEntryLister.ServiceEntries("test").Get("test-remote")
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find ServiceEntry test-remote: %v", err))
  }
  assert.Equal(t, "test-remote", entry.Name)
  assert.Equal(t, "test", entry.Namespace)
  assert.Equal(t, 1, len(entry.Spec.Ports))
  assert.Equal(t, 8182, int(entry.Spec.Ports[0].Number))
  assert.Equal(t, 1, len(entry.Spec.Addresses))
  assert.Equal(t, "240.0.0.1", entry.Spec.Addresses[0])

  entry, err = clusterConnection.IstioServiceEntryLister.ServiceEntries("test").Get("test2-remote")
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find ServiceEntry test2-remote: %v", err))
  }
  assert.Equal(t, "test2-remote", entry.Name)
  assert.Equal(t, "test", entry.Namespace)
  assert.Equal(t, 1, len(entry.Spec.Ports))
  assert.Equal(t, 8183, int(entry.Spec.Ports[0].Number))
  assert.Equal(t, 1, len(entry.Spec.Addresses))
  assert.Equal(t, "240.0.0.2", entry.Spec.Addresses[0])
}

func TestCreateIngresses(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// create the ingresses
	err := CreateIngresses(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create ingresses: %v", err))
	}
	assertCreateIngresses(t, clusterConnection, "*")

	// change the ingress dns name in the binding
	modelBindingPair.Binding.Spec.IngressBindings[0].DnsName = "foo"
	// create the ingresses
	err = CreateIngresses(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create ingresses: %v", err))
	}
	assertCreateIngresses(t, clusterConnection, "foo")

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName
	err = CreateIngresses(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create ingresses: %v", err))
	}
}

func assertCreateIngresses(t *testing.T, clusterConnection *util.ManagedClusterConnection, dnsName string) {
  list, err := clusterConnection.IstioGatewayLister.Gateways("test").List(labels.Everything())
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find Gateways in test namespace: %v", err))
  }
  assert.Equal(t, 1, len(list), "should be 1 Gateways in test namespace")

  gateway, err := clusterConnection.IstioGatewayLister.Gateways("test").Get("test-ingress-gateway")
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find Gateway test-ingress-gateway: %v", err))
  }
  assert.Equal(t, "test-ingress-gateway", gateway.Name)

  list2, err := clusterConnection.IstioVirtualServiceLister.VirtualServices("test").List(labels.Everything())
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find VirtualServices in test namespace: %v", err))
  }
  assert.Equal(t, 1, len(list2), "should be 1 VirtualServices in test namespace")

  service, err := clusterConnection.IstioVirtualServiceLister.VirtualServices("test").Get("test-ingress-virtualservice")
  if err != nil {
    t.Fatal(fmt.Sprintf("expected to find VirtualService test-ingress-virtualservice: %v", err))
  }
  assert.Equal(t, "test-ingress-virtualservice", service.Name)
}

func TestCreateDestinationRules(t *testing.T) {
	modelBindingPair := getModelBindingPair()
	clusterConnections := getManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// create the destination rules
	err := CreateDestinationRules(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create destination rules: %v", err))
	}
	assertCreateDestinationRules(t, clusterConnection, 9000)

	// change the coherence extend port in the model
	modelBindingPair.Model.Spec.CoherenceClusters[0].Ports[0].Port = 9001
	// create the destination rules
	err = CreateDestinationRules(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create destination rules: %v", err))
	}
	assertCreateDestinationRules(t, clusterConnection, 9001)
}

func assertCreateDestinationRules(t *testing.T, clusterConnection *util.ManagedClusterConnection, cohPort int) {
	// test-destination-rule
	list, err := clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find DestinationRules in test namespace: %v", err))
	}
	assert.Equal(t, 1, len(list.Items), "should be 1 DestinationRule in test namespace")
	rule, err := clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test").Get(context.TODO(), "test-destination-rule", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find DestinationRule test-destination-rule: %v", err))
	}
	assert.Equal(t, "test", rule.Namespace)
	assert.Equal(t, istio2.ClientTLSSettings_ISTIO_MUTUAL, rule.Spec.TrafficPolicy.Tls.Mode)
	assert.Equal(t, 1, len(rule.Spec.TrafficPolicy.PortLevelSettings))
	assert.Equal(t, istio2.ClientTLSSettings_DISABLE, rule.Spec.TrafficPolicy.PortLevelSettings[0].Tls.Mode)
	assert.Equal(t, cohPort, int(rule.Spec.TrafficPolicy.PortLevelSettings[0].Port.Number))

	// test2-destination-rule
	list, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test2").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find DestinationRules in test2 namespace: %v", err))
	}
	assert.Equal(t, 1, len(list.Items), "should be 1 DestinationRule in test2 namespace")
	rule, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test2").Get(context.TODO(), "test2-destination-rule", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find DestinationRule test2-destination-rule: %v", err))
	}
	assert.Equal(t, "test2", rule.Namespace)
	assert.Equal(t, istio2.ClientTLSSettings_ISTIO_MUTUAL, rule.Spec.TrafficPolicy.Tls.Mode)
	assert.Nil(t, rule.Spec.TrafficPolicy.PortLevelSettings)

	// test3-destination-rule
	list, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test3").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find DestinationRules in test2 namespace: %v", err))
	}
	assert.Equal(t, 1, len(list.Items), "should be 1 DestinationRule in test3 namespace")
	rule, err = clusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules("test3").Get(context.TODO(), "test3-destination-rule", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find DestinationRule test3-destination-rule: %v", err))
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
	err := CreateAuthorizationPolicies(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create authorization policies: %v", err))
	}
	assertCreateAuthorizationPolicies(t, clusterConnection, true)

	// change the WebLogic domain connections in the model
	modelBindingPair.Model.Spec.WeblogicDomains[0].Connections = nil

	// recreate the authorization policies
	err = CreateAuthorizationPolicies(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf(" should not raise an error: %v", err))
	}
	assertCreateAuthorizationPolicies(t, clusterConnection, false)
}

func assertCreateAuthorizationPolicies(t *testing.T, clusterConnection *util.ManagedClusterConnection, wlsHasConn bool) {

	// test-authorization-policy
	list, err := clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find AuthorizationPolicies in test namespace: %v", err))
	}
	assert.Equal(t, 1, len(list.Items), "should be 1 AuthorizationPolicy in test namespace")
	policy, err := clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test").Get(context.TODO(), "test-authorization-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find AuthorizationPolicy test-authorization-policy: %v", err))
	}
	assert.Equal(t, "test", policy.Namespace)
	assert.Equal(t, 1, len(policy.Spec.Rules))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
	assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "test"))
	assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "istio-system"))

	// test2-authorization-policy
	list, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test2").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find AuthorizationPolicies in test2 namespace: %v", err))
	}
	assert.Equal(t, 1, len(list.Items), "should be 1 AuthorizationPolicy in test2 namespace")
	policy, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test2").Get(context.TODO(), "test2-authorization-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find AuthorizationPolicy test2-authorization-policy: %v", err))
	}
	assert.Equal(t, "test2", policy.Namespace)
	assert.Equal(t, 1, len(policy.Spec.Rules))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From))
	if wlsHasConn {
		assert.Equal(t, 3, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
		assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "test2"))
		assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "test3"))
		assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "istio-system"))
	} else {
		assert.Equal(t, 2, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
		assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "test2"))
		assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "istio-system"))
	}

	// test3-authorization-policy
	list, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test3").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected to find AuthorizationPolicies in test3 namespace: %v", err))
	}
	assert.Equal(t, 1, len(list.Items), "should be 1 AuthorizationPolicy in test3 namespace")
	policy, err = clusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies("test3").Get(context.TODO(), "test3-authorization-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("expected find AuthorizationPolicy test3-authorization-policy: %v", err))
	}
	assert.Equal(t, "test3", policy.Namespace)
	assert.Equal(t, 1, len(policy.Spec.Rules))
	assert.Equal(t, 2, len(policy.Spec.Rules[0].From))
	assert.Equal(t, 3, len(policy.Spec.Rules[0].From[0].Source.Namespaces))
	assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "test"))
	assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "test3"))
	assert.True(t, contains(policy.Spec.Rules[0].From[0].Source.Namespaces, "istio-system"))
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
