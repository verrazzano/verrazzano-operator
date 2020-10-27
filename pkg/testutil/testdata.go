// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Utilities

package testutil

import (
	"context"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned/fake"
	istioClientsetFake "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned/fake"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	testutil "github.com/verrazzano/verrazzano-operator/test/integ/util"
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
		"cluster1": GetManagedClusterConnection("cluster1"),
		"cluster2": GetManagedClusterConnection("cluster2"),
		"cluster3": GetManagedClusterConnection("cluster3"),
	}
}

// GetManagedClusterConnection returns a managed cluster connection that uses fake client sets
func GetManagedClusterConnection(clusterName string) *util.ManagedClusterConnection {
	// create a ManagedClusterConnection that uses client set fakes
	clusterConnection := &util.ManagedClusterConnection{
		KubeClient:                  fake.NewSimpleClientset(),
		KubeExtClientSet:            apiextensionsclient.NewSimpleClientset(),
		VerrazzanoOperatorClientSet: clientset.NewSimpleClientset(),
		WlsOprClientSet:             NewWlsOprClientset(),
		DomainClientSet:             domclientset.NewSimpleClientset(),
		HelidonClientSet:            NewHelidionClientset(),
		CohOprClientSet:             NewCohOprClientset(),
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

	clusterConnection.ClusterRoleLister = &simpleClusterRoleLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.ClusterRoleBindingLister = &simpleClusterRoleBindingLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.DaemonSetLister = &simpleDaemonSetLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.DeploymentLister = &simpleDeploymentLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.SecretLister = &SimpleSecretLister{
		KubeClient: clusterConnection.KubeClient,
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

	clusterConnection.CohOperatorLister = &simpleCohClusterLister{
		kubeClient:      clusterConnection.KubeClient,
		cohOprClientSet: clusterConnection.CohOprClientSet,
	}

	clusterConnection.CohClusterLister = &simpleCoherenceClusterLister{
		kubeClient:          clusterConnection.KubeClient,
		cohClusterClientSet: clusterConnection.CohClusterClientSet,
	}

	clusterConnection.HelidonLister = &simpleHelidonAppLister{
		kubeClient:       clusterConnection.KubeClient,
		helidonClientSet: clusterConnection.HelidonClientSet,
	}

	clusterConnection.WlsOperatorLister = &simpleWlsOperatorLister{
		kubeClient:      clusterConnection.KubeClient,
		wlsOprClientSet: clusterConnection.WlsOprClientSet,
	}

	clusterConnection.DomainLister = &simpleDomainLister{
		kubeClient:      clusterConnection.KubeClient,
		domainClientSet: clusterConnection.DomainClientSet,
	}

	for _, pod := range getPods() {
		clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), getNamespace(pod.Namespace, clusterName), metav1.CreateOptions{})
		clusterConnection.KubeClient.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	}
	clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), getNamespace("logging", ""), metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), getNamespace("monitoring", ""), metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), getNamespace("istio-system", ""), metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Pods("istio-system").Create(context.TODO(), getPod("prometheus-pod", "istio-system", "123.99.0.1"), metav1.CreateOptions{})

	// create test secrets to associate to default namespace
	clusterConnection.KubeClient.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testSecret1",
			Namespace: "default",
		},
	}, metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testMysqlSecret",
			Namespace: "default",
		},
	}, metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Secrets("test").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testSecret1",
			Namespace: "test",
		},
		Data: map[string][]byte{"dummy": {'a', 'b', 'c'}},
	}, metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Secrets("verrazzano-system").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testSecret2",
			Namespace: "verrazzano-system",
		},
	}, metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2Secret1",
			Namespace: "default",
		},
	}, metav1.CreateOptions{})
	clusterConnection.KubeClient.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2Secret2",
			Namespace: "default",
		},
	}, metav1.CreateOptions{})

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

	clusterConnection.ServiceAccountLister = &simpleServiceAccountLister{
		kubeClient: clusterConnection.KubeClient,
	}
	return clusterConnection
}

func getNamespace(name string, clusterName string) *corev1.Namespace {
	if clusterName == "" {
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
		getPod("test-pod", "test", "123.99.0.2"),
		getPod("test2-pod", "test2", "123.99.0.3"),
		getPod("test3-pod", "test3", "123.99.0.4"),
	}
}

// GetModelBindingPair returns a test model binding pair.
func GetModelBindingPair() *types.ModelBindingPair {
	return ReadModelBindingPair(
		"../testutil/testdata/test_model.yaml",
		"../testutil/testdata/test_binding.yaml",
		"../testutil/testdata/test_managed_cluster_1.yaml", "../testutil/testdata/test_managed_cluster_2.yaml")
}

// ReadModelBindingPair returns a test model binding pair for the given model/binding/cluster descriptors.
func ReadModelBindingPair(modelPath string, bindingPath string, managedClusterPaths ...string) *types.ModelBindingPair {
	model, _ := testutil.ReadModel(modelPath)
	binding, _ := testutil.ReadBinding(bindingPath)
	managedClusters := map[string]*types.ManagedCluster{}

	for _, managedClusterPath := range managedClusterPaths {
		managedCluster, _ := testutil.ReadManagedCluster(managedClusterPath)
		managedClusters[managedCluster.Name] = managedCluster
	}
	var pair = &types.ModelBindingPair{
		Model:           model,
		Binding:         binding,
		ManagedClusters: managedClusters,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test-imagePullSecret",
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

// GetManifest gets a test Manifest.
func GetManifest() util.Manifest {
	return util.Manifest{
		WlsMicroOperatorImage:   "WlsOperator1:latest",
		WlsMicroOperatorCrd:     "../testutil/testdata/test_wls_crd.yaml",
		HelidonAppOperatorImage: "HelidonApp1:latest",
		HelidonAppOperatorCrd:   "../testutil/testdata/test_helidon_crd.yaml",
		CohClusterOperatorImage: "CohCluster1:latest",
		CohClusterOperatorCrd:   "../testutil/testdata/test_coh_crd.yaml",
	}
}
