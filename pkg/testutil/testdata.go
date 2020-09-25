// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Utilities

package testutil

import (
	"context"

	cohv1 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned/fake"
	istioClientsetFake "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned/fake"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	istioAuthClientset "istio.io/client-go/pkg/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// IstioSystemNamespace duplicate of managed.IstioSystemNamespace to avoid circular import
const IstioSystemNamespace = "istio-system"

// GetManagedClusterConnections returns a test map of managed cluster connections that uses fake client sets
func GetManagedClusterConnections() map[string]*util.ManagedClusterConnection {
	return map[string]*util.ManagedClusterConnection{
		"cluster1": getManagedClusterConnection("cluster1"),
		"cluster2": getManagedClusterConnection("cluster2"),
		"cluster3": getManagedClusterConnection("cluster3"),
	}
}

// Get a managed cluster connection that uses fake client sets
func getManagedClusterConnection(clusterName string) *util.ManagedClusterConnection {
	// create a ManagedClusterConnection that uses client set fakes
	clusterConnection := &util.ManagedClusterConnection{
		KubeClient:                  fake.NewSimpleClientset(),
		KubeExtClientSet:            apiextensionsclient.NewSimpleClientset(),
		VerrazzanoOperatorClientSet: clientset.NewSimpleClientset(),
		DomainClientSet:             domclientset.NewSimpleClientset(),
		CohClusterClientSet:         cohcluclientset.NewSimpleClientset(),
		IstioClientSet:              istioClientsetFake.NewSimpleClientset(),
		IstioAuthClientSet:          istioAuthClientset.NewSimpleClientset(),
	}
	// set a fake pod lister on the cluster connection
	clusterConnection.PodLister = &simplePodLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.ConfigMapLister = &simpleConfigMapLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.SecretLister = &simpleSecretLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.NamespaceLister = &simpleNamespaceLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.IstioGatewayLister = &simpleGatewayLister{
		kubeClient:     clusterConnection.KubeClient,
		istioClientSet: clusterConnection.IstioClientSet,
	}

	clusterConnection.IstioVirtualServiceLister = &simpleVirtualServiceLister{
		kubeClient:     clusterConnection.KubeClient,
		istioClientSet: clusterConnection.IstioClientSet,
	}

	clusterConnection.IstioServiceEntryLister = &simpleServiceEntryLister{
		kubeClient:     clusterConnection.KubeClient,
		istioClientSet: clusterConnection.IstioClientSet,
	}

	for _, pod := range getPods() {
		clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), getNamespace(pod.Namespace, clusterName), metav1.CreateOptions{})
		clusterConnection.KubeClient.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	}

	clusterConnection.KubeClient.CoreV1().Services(IstioSystemNamespace).Create(context.TODO(),
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioSystemNamespace,
				Name:      "istio-ingressgateway",
			},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							IP:       "123.45.0.1",
							Hostname: "host",
						},
					},
				},
			},
		}, metav1.CreateOptions{})

	clusterConnection.ServiceLister = &simpleServiceLister{
		kubeClient: clusterConnection.KubeClient,
	}

	return clusterConnection
}

func getNamespace(name string, clusterName string) *corev1.Namespace {
	if name == "istio-system" || name == "verrazzano-system" {
		return &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	}
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"verrazzano.binding": "testBinding",
				"verrazzano.cluster": clusterName,
			},
		},
	}
}

// Get a pod for testing that is populated with the given name, namespace and IP.
func getPod(name string, ns string, podIP string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			PodIP: podIP,
		},
	}
}

func getPods() []*corev1.Pod {
	return []*corev1.Pod{
		getPod("prometheus-pod", "istio-system", "123.99.0.1"),
		getPod("test-pod", "test", "123.99.0.2"),
		getPod("test2-pod", "test2", "123.99.0.3"),
		getPod("test3-pod", "test3", "123.99.0.4"),
	}
}

// GetModelBindingPairWithNames returns a test model and binding pair with the specified model and binding
// names in given NS.
func GetModelBindingPairWithNames(modelName string, bindingName string, ns string) *types.ModelBindingPair {
	pair := GetModelBindingPair()
	pair.Binding.Name = bindingName
	pair.Model.Name = modelName
	pair.Binding.Spec.ModelName = modelName
	pair.Model.Namespace = ns
	pair.Binding.Namespace = ns
	return pair
}

// GetModelBindingPair returns a test model binding pair.
func GetModelBindingPair() *types.ModelBindingPair {
	var pair = &types.ModelBindingPair{
		Model: &v1beta1.VerrazzanoModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testModel",
				Namespace: "default",
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
						Ports: []cohv1.NamedPortSpec{
							{
								Name: "extend",
								PortSpec: cohv1.PortSpec{
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testBinding",
				Namespace: "default",
			},
			Spec: v1beta1.VerrazzanoBindingSpec{
				ModelName: "testModel",
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
				Namespaces: []string{"default", "test", "test2", "test3", "istio-system", "verrazzano-system"},
				Ingresses: map[string][]*types.Ingress{
					"test": {
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
				RemoteRests: map[string][]*types.RemoteRestConnection{
					"test": {
						{
							Name:              "test-remote",
							RemoteNamespace:   "test2",
							RemoteClusterName: "cluster2",
							LocalNamespace:    "test",
							Port:              8182,
						},
						{
							Name:              "test2-remote",
							RemoteNamespace:   "test3",
							RemoteClusterName: "cluster2",
							LocalNamespace:    "test",
							Port:              8183,
						},
					},
				},
				ConfigMaps: []*corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "test",
						},
						Data: map[string]string{
							"foo": "aaa",
							"bar": "bbb",
						},
					},
				},
			},
			"cluster2": {
				Name:       "cluster2",
				Namespaces: []string{"default", "test2"},
			},
		},
	}
	return pair
}

// GetTestClusters returns a list of Verrazzano Managed Cluster resources.
func GetTestClusters() []v1beta1.VerrazzanoManagedCluster {
	return []v1beta1.VerrazzanoManagedCluster{
		{
			ObjectMeta: metav1.ObjectMeta{UID: "123-456-789", Name: "cluster1", Namespace: "default"},
			Spec:       v1beta1.VerrazzanoManagedClusterSpec{Type: "testCluster", ServerAddress: "test.com", Description: "Test Cluster"},
		},
	}
}
