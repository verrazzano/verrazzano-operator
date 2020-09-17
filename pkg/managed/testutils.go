// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Utilities used for unit testing.
package managed

import (
  v13 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
  "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
  clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
  cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned/fake"
  istioClientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned/fake"
  domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned/fake"
  "github.com/verrazzano/verrazzano-operator/pkg/types"
  "github.com/verrazzano/verrazzano-operator/pkg/util"
  istioAuthClientset "istio.io/client-go/pkg/clientset/versioned/fake"
  "k8s.io/api/core/v1"
  apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
  v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/labels"
  "k8s.io/client-go/kubernetes/fake"
  corelistersv1 "k8s.io/client-go/listers/core/v1"
)

// Fake PodLister implementation.  Used for unit testing.
type fakePodLister struct {
  pods []*v1.Pod
}

// List lists all Pods in the indexer.
func (s *fakePodLister) List(labels.Selector) (ret []*v1.Pod, err error) {
  // TODO: The selector is not currently needed and is ignored
  return s.pods, nil
}

// Pods returns an object that can list and get Pods.
func (s *fakePodLister) Pods(string) corelistersv1.PodNamespaceLister {
  // TODO: This is not currently needed and only implemented to satisfy the PodLister interface
  panic("implement me")
}

// Get a pod for testing that is populated with the given name, namespace and IP.
func getPod(name string, ns string, podIP string) *v1.Pod {
  return &v1.Pod{
    ObjectMeta: v12.ObjectMeta{
      Name:      name,
      Namespace: ns,
    },
    Status: v1.PodStatus{
      PodIP: podIP,
    },
  }
}

// Get a test map of managed cluster connections that use client set fakes,
func getManagedClusterConnections() map[string]*util.ManagedClusterConnection {
  // create a ManagedClusterConnection that uses client set fakes
  clusterConnection := &util.ManagedClusterConnection{
    KubeClient:                  fake.NewSimpleClientset(),
    KubeExtClientSet:            apiextensionsclient.NewSimpleClientset(),
    VerrazzanoOperatorClientSet: clientset.NewSimpleClientset(),
    DomainClientSet:             domclientset.NewSimpleClientset(),
    CohClusterClientSet:         cohcluclientset.NewSimpleClientset(),
    IstioClientSet:              istioClientset.NewSimpleClientset(),
    IstioAuthClientSet:          istioAuthClientset.NewSimpleClientset(),
  }
  // set a fake pod lister on the cluster connection
  clusterConnection.PodLister = &fakePodLister{
    pods: []*v1.Pod{
      getPod("prometheus-pod", "istio-system", "123.99.0.1"),
      getPod("test-pod", "test", "123.99.0.2"),
      getPod("test2-pod", "test2", "123.99.0.3"),
      getPod("test3-pod", "test3", "123.99.0.4"),
    },
  }

  return map[string]*util.ManagedClusterConnection{
    "cluster1": clusterConnection,
  }
}

// Get a test model binding pair.
func getModelBindingPair() *types.ModelBindingPair {
  var pair = &types.ModelBindingPair{
    Model: &v1beta1.VerrazzanoModel{
      ObjectMeta: v12.ObjectMeta{
        Name: "testModel",
      },
      Spec: v1beta1.VerrazzanoModelSpec{
        Description: "",
        WeblogicDomains: []v1beta1.VerrazzanoWebLogicDomain{
          {
            Name: "test-weblogic",
            Connections: []v1beta1.VerrazzanoConnections{
              {
                Ingress: []v1beta1.VerrazzanoIngressConnection{
                  {
                    Name: "test-ingress",
                  },
                },
                Rest: []v1beta1.VerrazzanoRestConnection{
                  {
                    Target: "test-helidon",
                  },
                },
              },
            },
          },
        },
        CoherenceClusters: []v1beta1.VerrazzanoCoherenceCluster{
          {
            Name: "test-coherence",
            Ports: []v13.NamedPortSpec{
              {
                Name: "extend",
                PortSpec: v13.PortSpec{
                  Port: 9000,
                },
              },
            },
            Connections: []v1beta1.VerrazzanoConnections{
              {
                Ingress: []v1beta1.VerrazzanoIngressConnection{
                  {
                    Name: "test-ingress",
                  },
                },
                Rest: []v1beta1.VerrazzanoRestConnection{
                  {
                    Target: "test-weblogic",
                  },
                },
              },
            },
          },
        },
        HelidonApplications: []v1beta1.VerrazzanoHelidon{
          {
            Name:       "test-helidon",
            Port:       8001,
            TargetPort: 8002,
            Connections: []v1beta1.VerrazzanoConnections{
              {
                Ingress: []v1beta1.VerrazzanoIngressConnection{
                  {
                    Name: "test-ingress",
                  },
                },
                Coherence: []v1beta1.VerrazzanoCoherenceConnection{
                  {
                    Target: "test-coherence",
                  },
                },
              },
            },
          },
        },
      },
    },
    Binding: &v1beta1.VerrazzanoBinding{
      ObjectMeta: v12.ObjectMeta{
        Name: "testBinding",
      },
      Spec: v1beta1.VerrazzanoBindingSpec{
        Placement: []v1beta1.VerrazzanoPlacement{
          {
            Name: "local",
            Namespaces: []v1beta1.KubernetesNamespace{
              {
                Name: "test",
                Components: []v1beta1.BindingComponent{
                  {
                    Name: "test-coherence",
                  },
                },
              },
              {
                Name: "test2",
                Components: []v1beta1.BindingComponent{
                  {
                    Name: "test-helidon",
                  },
                },
              },
              {
                Name: "test3",
                Components: []v1beta1.BindingComponent{
                  {
                    Name: "test-weblogic",
                  },
                },
              },
            },
          },
        },
        IngressBindings: []v1beta1.VerrazzanoIngressBinding{
          {
            Name:    "test-ingress",
            DnsName: "*",
          },
        },
      },
    },
    ManagedClusters: map[string]*types.ManagedCluster{
      "cluster1": {
        Name:       "cluster1",
        Namespaces: []string{"default", "test", "test2", "test3"},
        Ingresses: map[string][]*types.Ingress{
          "ingress1": {
            {
              Name: "test-ingress",
              Destination: []*types.IngressDestination{
                {
                  Host:       "testhost",
                  Port:       8888,
                  DomainName: "test.com",
                },
              },
            },
          },
        },
      },
    },
  }
  return pair
}
