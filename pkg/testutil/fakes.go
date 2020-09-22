// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"context"

	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned"
	v1alpha32 "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/listers/networking.istio.io/v1alpha3"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	v13 "k8s.io/client-go/listers/core/v1"
)

// ----- simplePodLister
// Simple PodLister implementation.
type simplePodLister struct {
	kubeClient kubernetes.Interface
}

// list all Pods
func (s *simplePodLister) List(selector labels.Selector) (ret []*v1.Pod, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
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
func (s *simplePodLister) Pods(namespace string) v13.PodNamespaceLister {
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

	list, err := s.kubeClient.CoreV1().Pods(s.namespace).List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range list.Items {
		pods = append(pods, &pod)
	}
	return pods, nil
}

// retrieves the Pod for a given namespace and name
func (s simplePodNamespaceLister) Get(name string) (*v1.Pod, error) {
	return s.kubeClient.CoreV1().Pods(s.namespace).Get(context.TODO(), name, v12.GetOptions{})
}

// ----- simpleSecretLister
// Simple secret sister implementation.
type simpleSecretLister struct {
	kubeClient kubernetes.Interface
}

// List all secrets
func (s *simpleSecretLister) List(selector labels.Selector) (ret []*v1.Secret, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
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
func (s *simpleSecretLister) Secrets(namespace string) v13.SecretNamespaceLister {
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

	list, err := s.kubeClient.CoreV1().Secrets(s.namespace).List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	var secrets []*v1.Secret = nil
	var items []v1.Secret = list.Items
	for i := range items {
		//for i := 0; i < len(items); i++ {
		secrets = append(secrets, &items[i])
	}
	return secrets, nil
}

// Retrieves the secret for a given namespace and name
func (s simpleSecretNamespaceLister) Get(name string) (*v1.Secret, error) {
	return s.kubeClient.CoreV1().Secrets(s.namespace).Get(context.TODO(), name, v12.GetOptions{})
}

// ----- simpleGatewayLister
// simple GatewayLister implementation
type simpleGatewayLister struct {
	kubeClient     kubernetes.Interface
	istioClientSet versioned.Interface
}

// list all Gateways
func (s *simpleGatewayLister) List(selector labels.Selector) (ret []*v1alpha3.Gateway, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
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
func (s *simpleGatewayLister) Gateways(namespace string) v1alpha32.GatewayNamespaceLister {
	return simpleGatewayNamespaceLister{
		namespace:      namespace,
		istioClientSet: s.istioClientSet,
	}
}

type simpleGatewayNamespaceLister struct {
	namespace      string
	istioClientSet versioned.Interface
}

// lists all Gateways for a given namespace
func (s simpleGatewayNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.Gateway, err error) {
	var gateways []*v1alpha3.Gateway

	list, err := s.istioClientSet.NetworkingV1alpha3().Gateways(s.namespace).List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, gateway := range list.Items {
		gateways = append(gateways, &gateway)
	}
	return gateways, nil
}

// retrieves the Gateway for a given namespace and name
func (s simpleGatewayNamespaceLister) Get(name string) (*v1alpha3.Gateway, error) {
	return s.istioClientSet.NetworkingV1alpha3().Gateways(s.namespace).Get(context.TODO(), name, v12.GetOptions{})
}

// ----- simpleVirtualServiceLister
// simple VirtualServiceLister implementation
type simpleVirtualServiceLister struct {
	kubeClient     kubernetes.Interface
	istioClientSet versioned.Interface
}

// lists all VirtualServices
func (s *simpleVirtualServiceLister) List(selector labels.Selector) (ret []*v1alpha3.VirtualService, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
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
func (s *simpleVirtualServiceLister) VirtualServices(namespace string) v1alpha32.VirtualServiceNamespaceLister {
	return simpleVirtualServiceNamespaceLister{
		namespace:      namespace,
		istioClientSet: s.istioClientSet,
	}
}

type simpleVirtualServiceNamespaceLister struct {
	namespace      string
	istioClientSet versioned.Interface
}

// lists all VirtualServices for a given namespace
func (s simpleVirtualServiceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.VirtualService, err error) {
	var services []*v1alpha3.VirtualService

	list, err := s.istioClientSet.NetworkingV1alpha3().VirtualServices(s.namespace).List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, service := range list.Items {
		services = append(services, &service)
	}
	return services, nil
}

// retrieves the VirtualService for a given namespace and name
func (s simpleVirtualServiceNamespaceLister) Get(name string) (*v1alpha3.VirtualService, error) {
	return s.istioClientSet.NetworkingV1alpha3().VirtualServices(s.namespace).Get(context.TODO(), name, v12.GetOptions{})
}

// ----- simpleServiceEntryLister
// simple ServiceEntryLister implementation
type simpleServiceEntryLister struct {
	kubeClient     kubernetes.Interface
	istioClientSet versioned.Interface
}

// lists all ServiceEntries
func (s *simpleServiceEntryLister) List(selector labels.Selector) (ret []*v1alpha3.ServiceEntry, err error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
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
func (s *simpleServiceEntryLister) ServiceEntries(namespace string) v1alpha32.ServiceEntryNamespaceLister {
	return simpleServiceEntryNamespaceLister{
		namespace:      namespace,
		istioClientSet: s.istioClientSet,
	}
}

type simpleServiceEntryNamespaceLister struct {
	namespace      string
	istioClientSet versioned.Interface
}

// lists all ServiceEntries for a given namespace
func (s simpleServiceEntryNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.ServiceEntry, err error) {
	var entries []*v1alpha3.ServiceEntry

	list, err := s.istioClientSet.NetworkingV1alpha3().ServiceEntries(s.namespace).List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, entry := range list.Items {
		entries = append(entries, &entry)
	}
	return entries, nil
}

// retrieves the ServiceEntry for a given namespace and name
func (s simpleServiceEntryNamespaceLister) Get(name string) (*v1alpha3.ServiceEntry, error) {
	return s.istioClientSet.NetworkingV1alpha3().ServiceEntries(s.namespace).Get(context.TODO(), name, v12.GetOptions{})
}
