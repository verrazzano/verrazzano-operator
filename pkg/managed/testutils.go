// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Utilities
package managed

import (
	"context"

	v13 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned/fake"
	istioClientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned"
	istioClientsetFake "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned/fake"
	istioLister "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/listers/networking.istio.io/v1alpha3"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	istioAuthClientset "istio.io/client-go/pkg/clientset/versioned/fake"
	v1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
)

// ----- simplePodLister
// Simple PodLister implementation.
type simplePodLister struct {
	kubeClient kubernetes.Interface
}

// list all Pods
func (s *simplePodLister) List(selector labels.Selector) (ret []*v1.Pod, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []*v1.Pod
	for _, namespace := range namespaces.Items {

		list, err := s.Pods(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		pods = append(pods, list...)
	}
	return pods, nil
}

// returns an object that can list and get Pods for the given namespace
func (s *simplePodLister) Pods(namespace string) corelistersv1.PodNamespaceLister {
	return simplePodNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

type simplePodNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// list all Pods for a given namespace
func (s simplePodNamespaceLister) List(selector labels.Selector) (ret []*v1.Pod, err error) {
	var pods []*v1.Pod

	list, err := s.kubeClient.CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		pod := list.Items[i]
		if selector.Matches(labels.Set(pod.Labels)) {
			pods = append(pods, &pod)
		}
	}
	return pods, nil
}

// retrieves the Pod for a given namespace and name
func (s simplePodNamespaceLister) Get(name string) (*v1.Pod, error) {
	return s.kubeClient.CoreV1().Pods(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleSecretLister
// Simple secret sister implementation.
type simpleSecretLister struct {
	kubeClient kubernetes.Interface
}

// List all secrets
func (s *simpleSecretLister) List(selector labels.Selector) (ret []*v1.Secret, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var secrets []*v1.Secret
	for _, namespace := range namespaces.Items {
		list, err := s.Secrets(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, list...)
	}
	return secrets, nil
}

// Returns an object that can list and get secrets for the given namespace
func (s *simpleSecretLister) Secrets(namespace string) corelistersv1.SecretNamespaceLister {
	return simpleSecretNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

// Simple secret namespace lister implementation.
type simpleSecretNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// List all secret for a given namespace
func (s simpleSecretNamespaceLister) List(selector labels.Selector) (ret []*v1.Secret, err error) {

	list, err := s.kubeClient.CoreV1().Secrets(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var secrets []*v1.Secret = nil
	var items []v1.Secret = list.Items
	for i := 0; i < len(items); i++ {
		secrets = append(secrets, &items[i])
	}
	return secrets, nil
}

// Retrieves the secret for a given namespace and name
func (s simpleSecretNamespaceLister) Get(name string) (*v1.Secret, error) {
	return s.kubeClient.CoreV1().Secrets(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleNamespaceLister
// simple NamespaceLister implementation
type simpleNamespaceLister struct {
	kubeClient kubernetes.Interface
}

// list all Namespaces
func (s *simpleNamespaceLister) List(selector labels.Selector) (ret []*v1.Namespace, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var list []*v1.Namespace
	for i := range namespaces.Items {
		namespace := namespaces.Items[i]
		if selector.Matches(labels.Set(namespace.Labels)) {
			list = append(list, &namespace)
		}
	}
	return list, nil
}

// retrieves the Namespace for a given name
func (s *simpleNamespaceLister) Get(name string) (*v1.Namespace, error) {
	return s.kubeClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleGatewayLister
// simple GatewayLister implementation
type simpleGatewayLister struct {
	kubeClient     kubernetes.Interface
	istioClientSet istioClientset.Interface
}

// list all Gateways
func (s *simpleGatewayLister) List(selector labels.Selector) (ret []*v1alpha3.Gateway, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var gateways []*v1alpha3.Gateway
	for _, namespace := range namespaces.Items {

		list, err := s.Gateways(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, list...)
	}
	return gateways, nil
}

// returns an object that can list and get Gateways for the given namespace
func (s *simpleGatewayLister) Gateways(namespace string) istioLister.GatewayNamespaceLister {
	return simpleGatewayNamespaceLister{
		namespace:      namespace,
		istioClientSet: s.istioClientSet,
	}
}

type simpleGatewayNamespaceLister struct {
	namespace      string
	istioClientSet istioClientset.Interface
}

// lists all Gateways for a given namespace
func (s simpleGatewayNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.Gateway, err error) {
	var gateways []*v1alpha3.Gateway

	list, err := s.istioClientSet.NetworkingV1alpha3().Gateways(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		gateway := list.Items[i]
		if selector.Matches(labels.Set(gateway.Labels)) {
			gateways = append(gateways, &gateway)
		}
	}
	return gateways, nil
}

// retrieves the Gateway for a given namespace and name
func (s simpleGatewayNamespaceLister) Get(name string) (*v1alpha3.Gateway, error) {
	return s.istioClientSet.NetworkingV1alpha3().Gateways(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleVirtualServiceLister
// simple VirtualServiceLister implementation
type simpleVirtualServiceLister struct {
	kubeClient     kubernetes.Interface
	istioClientSet istioClientset.Interface
}

// lists all VirtualServices
func (s *simpleVirtualServiceLister) List(selector labels.Selector) (ret []*v1alpha3.VirtualService, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var services []*v1alpha3.VirtualService
	for _, namespace := range namespaces.Items {

		list, err := s.VirtualServices(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		services = append(services, list...)
	}
	return services, nil
}

// returns an object that can list and get VirtualServices for a given namespace
func (s *simpleVirtualServiceLister) VirtualServices(namespace string) istioLister.VirtualServiceNamespaceLister {
	return simpleVirtualServiceNamespaceLister{
		namespace:      namespace,
		istioClientSet: s.istioClientSet,
	}
}

type simpleVirtualServiceNamespaceLister struct {
	namespace      string
	istioClientSet istioClientset.Interface
}

// lists all VirtualServices for a given namespace
func (s simpleVirtualServiceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.VirtualService, err error) {
	var services []*v1alpha3.VirtualService

	list, err := s.istioClientSet.NetworkingV1alpha3().VirtualServices(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		service := list.Items[i]
		if selector.Matches(labels.Set(service.Labels)) {
			services = append(services, &service)
		}
	}
	return services, nil
}

// retrieves the VirtualService for a given namespace and name
func (s simpleVirtualServiceNamespaceLister) Get(name string) (*v1alpha3.VirtualService, error) {
	return s.istioClientSet.NetworkingV1alpha3().VirtualServices(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleServiceEntryLister
// simple ServiceEntryLister implementation
type simpleServiceEntryLister struct {
	kubeClient     kubernetes.Interface
	istioClientSet istioClientset.Interface
}

// lists all ServiceEntries
func (s *simpleServiceEntryLister) List(selector labels.Selector) (ret []*v1alpha3.ServiceEntry, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var entries []*v1alpha3.ServiceEntry
	for _, namespace := range namespaces.Items {

		list, err := s.ServiceEntries(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		entries = append(entries, list...)
	}
	return entries, nil
}

// returns an object that can list and get ServiceEntries for a given namespace
func (s *simpleServiceEntryLister) ServiceEntries(namespace string) istioLister.ServiceEntryNamespaceLister {
	return simpleServiceEntryNamespaceLister{
		namespace:      namespace,
		istioClientSet: s.istioClientSet,
	}
}

type simpleServiceEntryNamespaceLister struct {
	namespace      string
	istioClientSet istioClientset.Interface
}

// lists all ServiceEntries for a given namespace
func (s simpleServiceEntryNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.ServiceEntry, err error) {
	var entries []*v1alpha3.ServiceEntry

	list, err := s.istioClientSet.NetworkingV1alpha3().ServiceEntries(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		entry := list.Items[i]
		entries = append(entries, &entry)
	}
	return entries, nil
}

// retrieves the ServiceEntry for a given namespace and name
func (s simpleServiceEntryNamespaceLister) Get(name string) (*v1alpha3.ServiceEntry, error) {
	return s.istioClientSet.NetworkingV1alpha3().ServiceEntries(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// Get a test map of managed cluster connections that uses fake client sets
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

	clusterConnection.NamespaceLister = &simpleNamespaceLister{
		kubeClient: clusterConnection.KubeClient,
	}

	clusterConnection.SecretLister = &simpleSecretLister{
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
		&v1.Service{
			ObjectMeta: v12.ObjectMeta{
				Namespace: IstioSystemNamespace,
				Name:      "istio-ingressgateway",
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							IP:       "123.45.0.1",
							Hostname: "host",
						},
					},
				},
			},
		}, metav1.CreateOptions{})

	return clusterConnection
}

func getNamespace(name string, clusterName string) *v1.Namespace {
	if name == "istio-system" || name == "verrazzano-system" {
		return &v1.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: name,
			},
		}
	} else {
		return &v1.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"verrazzano.binding": "testBinding",
					"verrazzano.cluster": clusterName,
				},
			},
		}
	}
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

func getPods() []*v1.Pod {
	return []*v1.Pod{
		getPod("prometheus-pod", "istio-system", "123.99.0.1"),
		getPod("test-pod", "test", "123.99.0.2"),
		getPod("test2-pod", "test2", "123.99.0.3"),
		getPod("test3-pod", "test3", "123.99.0.4"),
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
			},
			"cluster2": {
				Name:       "cluster2",
				Namespaces: []string{"default", "test2"},
			},
		},
	}
	return pair
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
