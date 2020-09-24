// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"context"

	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	istioClientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned"
	istioLister "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/listers/networking.istio.io/v1alpha3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
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

// simple ConfigMapLister implementation
type simpleConfigMapLister struct {
	kubeClient kubernetes.Interface
}

// lists all ConfigMaps
func (s *simpleConfigMapLister) List(selector labels.Selector) (ret []*v1.ConfigMap, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []*v1.ConfigMap
	for _, namespace := range namespaces.Items {

		list, err := s.ConfigMaps(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		pods = append(pods, list...)
	}
	return pods, nil
}

// ConfigMaps returns an object that can list and get ConfigMaps.
func (s *simpleConfigMapLister) ConfigMaps(namespace string) corelistersv1.ConfigMapNamespaceLister {
	return simpleConfigMapNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

// configMapNamespaceLister implements the ConfigMapNamespaceLister
// interface.
type simpleConfigMapNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// List lists all ConfigMaps in the indexer for a given namespace.
func (s simpleConfigMapNamespaceLister) List(selector labels.Selector) (ret []*v1.ConfigMap, err error) {
	var configMaps []*v1.ConfigMap

	list, err := s.kubeClient.CoreV1().ConfigMaps(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		configMap := list.Items[i]
		if selector.Matches(labels.Set(configMap.Labels)) {
			configMaps = append(configMaps, &configMap)
		}
	}
	return configMaps, nil
}

// Get retrieves the ConfigMap from the indexer for a given namespace and name.
func (s simpleConfigMapNamespaceLister) Get(name string) (*v1.ConfigMap, error) {
	return s.kubeClient.CoreV1().ConfigMaps(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
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
	for i := range items {
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

// ----- simpleServiceLister
// Simple ServiceLister implementation.
type simpleServiceLister struct {
	kubeClient kubernetes.Interface
}

// list all Services
func (s *simpleServiceLister) List(selector labels.Selector) (ret []*v1.Service, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var services []*v1.Service
	for _, namespace := range namespaces.Items {

		list, err := s.Services(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		services = append(services, list...)
	}
	return services, nil
}

// returns an object that can list and get Services for the given namespace
func (s *simpleServiceLister) Services(namespace string) corelistersv1.ServiceNamespaceLister {
	return simpleServiceNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

type simpleServiceNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// list all Services for a given namespace
func (s simpleServiceNamespaceLister) List(selector labels.Selector) (ret []*v1.Service, err error) {
	var services []*v1.Service

	list, err := s.kubeClient.CoreV1().Services(s.namespace).List(context.TODO(), metav1.ListOptions{})
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

// retrieves the Service for a given namespace and name
func (s simpleServiceNamespaceLister) Get(name string) (*v1.Service, error) {
	return s.kubeClient.CoreV1().Services(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleServiceAccountLister
// Simple ServiceAccountLister implementation.
type simpleServiceAccountLister struct {
	kubeClient kubernetes.Interface
}

// list all Service Accounts
func (s *simpleServiceAccountLister) List(selector labels.Selector) (ret []*v1.ServiceAccount, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var serviceAccounts []*v1.ServiceAccount
	for _, namespace := range namespaces.Items {

		list, err := s.ServiceAccounts(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		serviceAccounts = append(serviceAccounts, list...)
	}
	return serviceAccounts, nil
}

// returns an object that can list and get Service Accounts for the given namespace
func (s *simpleServiceAccountLister) ServiceAccounts(namespace string) corelistersv1.ServiceAccountNamespaceLister {
	return simpleServiceAccountNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

type simpleServiceAccountNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// list all Service Accounts for a given namespace
func (s simpleServiceAccountNamespaceLister) List(selector labels.Selector) (ret []*v1.ServiceAccount, err error) {
	var serviceAccounts []*v1.ServiceAccount

	list, err := s.kubeClient.CoreV1().ServiceAccounts(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		serviceAccount := list.Items[i]
		if selector.Matches(labels.Set(serviceAccount.Labels)) {
			serviceAccounts = append(serviceAccounts, &serviceAccount)
		}
	}
	return serviceAccounts, nil
}

// retrieves the Service Account for a given namespace and name
func (s simpleServiceAccountNamespaceLister) Get(name string) (*v1.ServiceAccount, error) {
	return s.kubeClient.CoreV1().ServiceAccounts(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}
