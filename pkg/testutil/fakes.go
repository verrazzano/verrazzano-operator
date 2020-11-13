// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	cohv1beta1 "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/apis/verrazzano/v1beta1"
	cohoprclientset "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/clientset/versioned"
	cohoprclientsetbeta1 "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"
	cohoprlister "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/listers/verrazzano/v1beta1"
	cohv1 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	v8 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned"
	cohclulister "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/listers/coherence/v1"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned"
	domlister "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/listers/weblogic/v8"
	helidionv1beta1 "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	helidonclientset "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/clientset/versioned"
	helidionclientsetbeta1 "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"
	helidionlister "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/listers/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	wlsoprclientset "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned"
	wlsoprclientsetbeta1 "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned/typed/verrazzano/v1beta1"
	wlsoprlister "github.com/verrazzano/verrazzano-wko-operator/pkg/client/listers/verrazzano/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioClientset "istio.io/client-go/pkg/clientset/versioned"
	istioLister "istio.io/client-go/pkg/listers/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	appslistersv1 "k8s.io/client-go/listers/apps/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	k8stesting "k8s.io/client-go/testing"
)

// ----- simplePodLister
// Simple PodLister implementation.
type simplePodLister struct {
	kubeClient kubernetes.Interface
}

// list all Pods
func (s *simplePodLister) List(selector labels.Selector) ([]*v1.Pod, error) {
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
func (s simplePodNamespaceLister) List(selector labels.Selector) ([]*v1.Pod, error) {
	var pods []*v1.Pod

	list, err := s.kubeClient.CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			pods = append(pods, &list.Items[i])
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
func (s *simpleConfigMapLister) List(selector labels.Selector) ([]*v1.ConfigMap, error) {
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

// List lists all ConfigMaps for a given namespace.
func (s simpleConfigMapNamespaceLister) List(selector labels.Selector) ([]*v1.ConfigMap, error) {
	var configMaps []*v1.ConfigMap

	list, err := s.kubeClient.CoreV1().ConfigMaps(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			configMaps = append(configMaps, &list.Items[i])
		}
	}
	return configMaps, nil
}

// Get retrieves the ConfigMap for a given namespace and name.
func (s simpleConfigMapNamespaceLister) Get(name string) (*v1.ConfigMap, error) {
	return s.kubeClient.CoreV1().ConfigMaps(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- SimpleSecretLister

// SimpleSecretLister is a simple secret Lister implementation useful for tests.
type SimpleSecretLister struct {
	KubeClient kubernetes.Interface
}

// List all secrets
func (s *SimpleSecretLister) List(selector labels.Selector) ([]*v1.Secret, error) {
	namespaces, err := s.KubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})

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

// Secrets returns an object that can list and get secrets for the given namespace
func (s *SimpleSecretLister) Secrets(namespace string) corelistersv1.SecretNamespaceLister {
	return simpleSecretNamespaceLister{
		namespace:  namespace,
		kubeClient: s.KubeClient,
	}
}

// Simple secret namespace lister implementation.
type simpleSecretNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// List all secret for a given namespace
func (s simpleSecretNamespaceLister) List(labels.Selector) ([]*v1.Secret, error) {

	list, err := s.kubeClient.CoreV1().Secrets(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var secrets []*v1.Secret = nil
	var items = list.Items
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
func (s *simpleNamespaceLister) List(selector labels.Selector) ([]*v1.Namespace, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []*v1.Namespace
	for i := range namespaces.Items {
		if selector.Matches(labels.Set(namespaces.Items[i].Labels)) {
			list = append(list, &namespaces.Items[i])
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
func (s *simpleGatewayLister) List(selector labels.Selector) ([]*v1alpha3.Gateway, error) {
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
func (s simpleGatewayNamespaceLister) List(selector labels.Selector) ([]*v1alpha3.Gateway, error) {
	var gateways []*v1alpha3.Gateway

	list, err := s.istioClientSet.NetworkingV1alpha3().Gateways(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			gateways = append(gateways, &list.Items[i])
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
func (s *simpleVirtualServiceLister) List(selector labels.Selector) ([]*v1alpha3.VirtualService, error) {
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
func (s simpleVirtualServiceNamespaceLister) List(selector labels.Selector) ([]*v1alpha3.VirtualService, error) {
	var services []*v1alpha3.VirtualService

	list, err := s.istioClientSet.NetworkingV1alpha3().VirtualServices(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			services = append(services, &list.Items[i])
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
func (s *simpleServiceEntryLister) List(selector labels.Selector) ([]*v1alpha3.ServiceEntry, error) {
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
func (s simpleServiceEntryNamespaceLister) List(labels.Selector) ([]*v1alpha3.ServiceEntry, error) {
	var entries []*v1alpha3.ServiceEntry

	list, err := s.istioClientSet.NetworkingV1alpha3().ServiceEntries(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		entries = append(entries, &list.Items[i])
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
func (s *simpleServiceLister) List(selector labels.Selector) ([]*v1.Service, error) {
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
func (s simpleServiceNamespaceLister) List(selector labels.Selector) ([]*v1.Service, error) {
	var services []*v1.Service

	list, err := s.kubeClient.CoreV1().Services(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			services = append(services, &list.Items[i])
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
func (s *simpleServiceAccountLister) List(selector labels.Selector) ([]*v1.ServiceAccount, error) {
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
func (s simpleServiceAccountNamespaceLister) List(selector labels.Selector) ([]*v1.ServiceAccount, error) {
	var serviceAccounts []*v1.ServiceAccount

	list, err := s.kubeClient.CoreV1().ServiceAccounts(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			serviceAccounts = append(serviceAccounts, &list.Items[i])
		}
	}
	return serviceAccounts, nil
}

// retrieves the Service Account for a given namespace and name
func (s simpleServiceAccountNamespaceLister) Get(name string) (*v1.ServiceAccount, error) {
	return s.kubeClient.CoreV1().ServiceAccounts(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// simple ClusterRoleLister implementation
type simpleClusterRoleLister struct {
	kubeClient kubernetes.Interface
}

// List lists all the ClusterRoles.
func (s *simpleClusterRoleLister) List(selector labels.Selector) ([]*rbacv1.ClusterRole, error) {
	var clusterRoles []*rbacv1.ClusterRole

	list, err := s.kubeClient.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			clusterRoles = append(clusterRoles, &list.Items[i])
		}
	}
	return clusterRoles, nil
}

// Get retrieves the ClusterRole for a given name.
func (s *simpleClusterRoleLister) Get(name string) (*rbacv1.ClusterRole, error) {
	return s.kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
}

// simpleClusterRoleBindingLister implements the ClusterRoleBindingLister interface.
type simpleClusterRoleBindingLister struct {
	kubeClient kubernetes.Interface
}

// List lists all ClusterRoleBindings.
func (s *simpleClusterRoleBindingLister) List(selector labels.Selector) ([]*rbacv1.ClusterRoleBinding, error) {
	var bindings []*rbacv1.ClusterRoleBinding

	list, err := s.kubeClient.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			bindings = append(bindings, &list.Items[i])
		}
	}
	return bindings, nil
}

// Get retrieves the ClusterRoleBinding for a given name.
func (s *simpleClusterRoleBindingLister) Get(name string) (*rbacv1.ClusterRoleBinding, error) {
	return s.kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
}

// InvokeHTTPHandler sets up a HTTP handler
func InvokeHTTPHandler(request *http.Request, path string, handler func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	responseRecorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc(path, handler)
	router.ServeHTTP(responseRecorder, request)

	return responseRecorder
}

//NewConfigMapLister creates a ConfigMapLister
func NewConfigMapLister(kubeClient kubernetes.Interface) corelistersv1.ConfigMapLister {
	return &simpleConfigMapLister{kubeClient: kubeClient}
}

//NewSecretLister creates a SecretLister
func NewSecretLister(kubeClient kubernetes.Interface) corelistersv1.SecretLister {
	return &SimpleSecretLister{KubeClient: kubeClient}
}

// simpleDaemonSetLister implements the DaemonSetLister interface.
type simpleDaemonSetLister struct {
	kubeClient kubernetes.Interface
}

// GetPodDaemonSets is not currently needed for testing.  Leaving unimplemented.
func (s *simpleDaemonSetLister) GetPodDaemonSets(*v1.Pod) ([]*appsv1.DaemonSet, error) {
	panic("not currently supported")
}

// GetHistoryDaemonSets is not currently needed for testing.  Leaving unimplemented.
func (s *simpleDaemonSetLister) GetHistoryDaemonSets(*appsv1.ControllerRevision) ([]*appsv1.DaemonSet, error) {
	panic("not currently supported")
}

// List lists all DaemonSets.
func (s *simpleDaemonSetLister) List(selector labels.Selector) ([]*appsv1.DaemonSet, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var sets []*appsv1.DaemonSet
	for _, namespace := range namespaces.Items {

		list, err := s.DaemonSets(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		sets = append(sets, list...)
	}
	return sets, nil
}

// DaemonSets returns an object that can list and get DaemonSets.
func (s *simpleDaemonSetLister) DaemonSets(namespace string) appslistersv1.DaemonSetNamespaceLister {
	return simpleDaemonSetNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

// daemonSetNamespaceLister implements the DaemonSetNamespaceLister
// interface.
type simpleDaemonSetNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

// List lists all DaemonSets for a given namespace.
func (s simpleDaemonSetNamespaceLister) List(selector labels.Selector) ([]*appsv1.DaemonSet, error) {
	var sets []*appsv1.DaemonSet

	list, err := s.kubeClient.AppsV1().DaemonSets(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		set := list.Items[i]
		if selector.Matches(labels.Set(set.Labels)) {
			sets = append(sets, &set)
		}
	}
	return sets, nil
}

// Get retrieves the DaemonSet for a given namespace and name.
func (s simpleDaemonSetNamespaceLister) Get(name string) (*appsv1.DaemonSet, error) {
	return s.kubeClient.AppsV1().DaemonSets(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

//MockError mocks a fake kubernetes.Interface to return an expected error
func MockError(kubeCli kubernetes.Interface, verb, resource string, obj runtime.Object) kubernetes.Interface {
	kubeCli.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor(verb, resource,
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, obj, fmt.Errorf("error %s %s", verb, resource)
		})
	return kubeCli
}

// simpleDeploymentLister implements the DeploymentLister interface.
type simpleDeploymentLister struct {
	kubeClient kubernetes.Interface
}

func (s simpleDeploymentLister) List(selector labels.Selector) ([]*appsv1.Deployment, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var deployments []*appsv1.Deployment
	for _, namespace := range namespaces.Items {

		list, err := s.Deployments(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, list...)
	}
	return deployments, nil
}

func (s simpleDeploymentLister) Deployments(namespace string) appslistersv1.DeploymentNamespaceLister {
	return simpleDeploymentNamespaceLister{
		namespace:  namespace,
		kubeClient: s.kubeClient,
	}
}

// deploymentNamespaceLister implements the DeploymentNamespaceLister
// interface.
type simpleDeploymentNamespaceLister struct {
	namespace  string
	kubeClient kubernetes.Interface
}

func (s simpleDeploymentNamespaceLister) List(selector labels.Selector) ([]*appsv1.Deployment, error) {
	var deployments []*appsv1.Deployment

	list, err := s.kubeClient.AppsV1().Deployments(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			deployments = append(deployments, &list.Items[i])
		}
	}
	return deployments, nil
}

func (s simpleDeploymentNamespaceLister) Get(name string) (*appsv1.Deployment, error) {
	return s.kubeClient.AppsV1().Deployments(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// FakeSecrets is a fake Secrets implementation
type FakeSecrets struct {
	Secrets map[string]*v1.Secret
}

// Get returns a secret for a given secret name and namespace.
func (s *FakeSecrets) Get(name string) (*v1.Secret, error) {
	return s.Secrets[name], nil
}

// Create creates a secret for a given secret name and namespace.
func (s *FakeSecrets) Create(newSecret *v1.Secret) (*v1.Secret, error) {
	s.Secrets[newSecret.Name] = newSecret
	return newSecret, nil
}

// Update updates a secret for a given secret name and namespace.
func (s *FakeSecrets) Update(newSecret *v1.Secret) (*v1.Secret, error) {
	s.Secrets[newSecret.Name] = newSecret
	return newSecret, nil
}

// List returns a list of secrets for given namespace and selector.
func (s *FakeSecrets) List(ns string, selector labels.Selector) ([]*v1.Secret, error) {
	var secrets []*v1.Secret
	for i := range s.Secrets {
		if s.Secrets[i].Namespace == ns && selector.Matches(labels.Set(s.Secrets[i].Labels)) {
			secrets = append(secrets, s.Secrets[i])
		}
	}
	return secrets, nil
}

// Delete deletes a secret for a given secret name and namespace.
func (s *FakeSecrets) Delete(_, name string) error {
	delete(s.Secrets, name)
	return nil
}

// GetVmiPassword returns a Verrazzano Monitoring Instance password for given namespace.
func (s *FakeSecrets) GetVmiPassword() (string, error) {
	return monitoring.GetVmiPassword(s)
}

// ----- simpleCohClusterLister
// simpleCohClusterLister implements the CohClusterLister interface.
type simpleCohClusterLister struct {
	kubeClient      kubernetes.Interface
	cohOprClientSet cohoprclientset.Interface
}

// List lists all CohClusters.
func (s *simpleCohClusterLister) List(selector labels.Selector) ([]*cohv1beta1.CohCluster, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var clusters []*cohv1beta1.CohCluster
	for _, namespace := range namespaces.Items {

		list, err := s.CohClusters(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, list...)
	}
	return clusters, nil
}

// CohClusters returns an object that can list and get CohClusters.
func (s *simpleCohClusterLister) CohClusters(namespace string) cohoprlister.CohClusterNamespaceLister {
	return simpleCohClusterNamespaceLister{
		namespace:       namespace,
		cohOprClientSet: s.cohOprClientSet,
	}
}

// simpleCohClusterNamespaceLister implements the CohClusterNamespaceLister interface.
type simpleCohClusterNamespaceLister struct {
	namespace       string
	cohOprClientSet cohoprclientset.Interface
}

// List lists all CohClusters for a given namespace.
func (s simpleCohClusterNamespaceLister) List(selector labels.Selector) ([]*cohv1beta1.CohCluster, error) {
	var clusters []*cohv1beta1.CohCluster

	list, err := s.cohOprClientSet.VerrazzanoV1beta1().CohClusters(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			clusters = append(clusters, &list.Items[i])
		}
	}
	return clusters, nil
}

// Get retrieves the CohCluster for a given namespace and name.
func (s simpleCohClusterNamespaceLister) Get(name string) (*cohv1beta1.CohCluster, error) {
	return s.cohOprClientSet.VerrazzanoV1beta1().CohClusters(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleCoherenceClusterLister
// simpleCoherenceClusterLister implements the CoherenceClusterLister interface.
type simpleCoherenceClusterLister struct {
	kubeClient          kubernetes.Interface
	cohClusterClientSet cohcluclientset.Interface
}

// List lists all CoherenceClusters.
func (s *simpleCoherenceClusterLister) List(selector labels.Selector) ([]*cohv1.CoherenceCluster, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var clusters []*cohv1.CoherenceCluster
	for _, namespace := range namespaces.Items {

		list, err := s.CoherenceClusters(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, list...)
	}
	return clusters, nil
}

// CoherenceClusters returns an object that can list and get CoherenceClusters.
func (s *simpleCoherenceClusterLister) CoherenceClusters(namespace string) cohclulister.CoherenceClusterNamespaceLister {
	return simpleCoherenceClusterNamespaceLister{
		namespace:       namespace,
		cohOprClientSet: s.cohClusterClientSet,
	}
}

// coherenceClusterNamespaceLister implements the CoherenceClusterNamespaceLister interface.
type simpleCoherenceClusterNamespaceLister struct {
	namespace       string
	cohOprClientSet cohcluclientset.Interface
}

// List lists all CoherenceClusters for a given namespace.
func (s simpleCoherenceClusterNamespaceLister) List(selector labels.Selector) ([]*cohv1.CoherenceCluster, error) {
	var clusters []*cohv1.CoherenceCluster

	clusterList, err := s.cohOprClientSet.CoherenceV1().CoherenceClusters(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range clusterList.Items {
		if selector.Matches(labels.Set(clusterList.Items[i].Labels)) {
			clusters = append(clusters, &clusterList.Items[i])
		}
	}
	return clusters, nil
}

// Get retrieves the CoherenceCluster for a given namespace and name.
func (s simpleCoherenceClusterNamespaceLister) Get(name string) (*cohv1.CoherenceCluster, error) {
	return s.cohOprClientSet.CoherenceV1().CoherenceClusters(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleHelidonAppLister
// simpleHelidonAppLister implements the HelidonAppLister interface.
type simpleHelidonAppLister struct {
	kubeClient       kubernetes.Interface
	helidonClientSet helidonclientset.Interface
}

// List lists all HelidonApps.
func (s *simpleHelidonAppLister) List(selector labels.Selector) ([]*helidionv1beta1.HelidonApp, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var apps []*helidionv1beta1.HelidonApp
	for _, namespace := range namespaces.Items {

		list, err := s.HelidonApps(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		apps = append(apps, list...)
	}
	return apps, nil
}

// HelidonApps returns an object that can list and get HelidonApps.
func (s *simpleHelidonAppLister) HelidonApps(namespace string) helidionlister.HelidonAppNamespaceLister {
	return simpleHelidonAppNamespaceLister{
		namespace:        namespace,
		helidonClientSet: s.helidonClientSet,
	}
}

// simpleHelidonAppNamespaceLister implements the HelidonAppNamespaceLister interface.
type simpleHelidonAppNamespaceLister struct {
	namespace        string
	helidonClientSet helidonclientset.Interface
}

// List lists all HelidonApps for a given namespace.
func (s simpleHelidonAppNamespaceLister) List(selector labels.Selector) ([]*helidionv1beta1.HelidonApp, error) {
	var apps []*helidionv1beta1.HelidonApp

	list, err := s.helidonClientSet.VerrazzanoV1beta1().HelidonApps(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			apps = append(apps, &list.Items[i])
		}
	}
	return apps, nil
}

// Get retrieves the HelidonApp for a given namespace and name.
func (s simpleHelidonAppNamespaceLister) Get(name string) (*helidionv1beta1.HelidonApp, error) {
	return s.helidonClientSet.VerrazzanoV1beta1().HelidonApps(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleDomainLister
// simpleDomainLister implements the DomainLister interface.
type simpleDomainLister struct {
	kubeClient      kubernetes.Interface
	domainClientSet domclientset.Interface
}

// List lists all Domains.`
func (s *simpleDomainLister) List(selector labels.Selector) ([]*v8.Domain, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var domains []*v8.Domain
	for _, namespace := range namespaces.Items {
		list, err := s.Domains(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		domains = append(domains, list...)
	}
	return domains, nil
}

// Domains returns an object that can list and get Domains.
func (s *simpleDomainLister) Domains(namespace string) domlister.DomainNamespaceLister {
	return simpleDomainNamespaceLister{
		namespace:       namespace,
		domainClientSet: s.domainClientSet,
	}
}

// simpleDomainNamespaceLister implements the DomainNamespaceLister interface.
type simpleDomainNamespaceLister struct {
	namespace       string
	domainClientSet domclientset.Interface
}

// List lists all Domains for a given namespace.
func (s simpleDomainNamespaceLister) List(selector labels.Selector) ([]*v8.Domain, error) {
	var domains []*v8.Domain

	list, err := s.domainClientSet.WeblogicV8().Domains(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			domains = append(domains, &list.Items[i])
		}
	}
	return domains, nil
}

// Get retrieves the Domain for a given namespace and name.
func (s simpleDomainNamespaceLister) Get(name string) (*v8.Domain, error) {
	return s.domainClientSet.WeblogicV8().Domains(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- simpleWlsOperatorLister
// simpleWlsOperatorLister implements the WlsOperatorLister interface.
type simpleWlsOperatorLister struct {
	kubeClient      kubernetes.Interface
	wlsOprClientSet wlsoprclientset.Interface
}

// List lists all WlsOperators.
func (s *simpleWlsOperatorLister) List(selector labels.Selector) ([]*v1beta1.WlsOperator, error) {
	namespaces, err := s.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var apps []*v1beta1.WlsOperator
	for _, namespace := range namespaces.Items {

		list, err := s.WlsOperators(namespace.Name).List(selector)
		if err != nil {
			return nil, err
		}
		apps = append(apps, list...)
	}
	return apps, nil
}

// WlsOperators returns an object that can list and get WlsOperators.
func (s *simpleWlsOperatorLister) WlsOperators(namespace string) wlsoprlister.WlsOperatorNamespaceLister {
	return simpleWlsOperatorNamespaceLister{
		namespace:       namespace,
		wlsOprClientSet: s.wlsOprClientSet,
	}
}

// simpleWlsOperatorNamespaceLister implements the WlsOperatorNamespaceLister interface.
type simpleWlsOperatorNamespaceLister struct {
	namespace       string
	wlsOprClientSet wlsoprclientset.Interface
}

// List lists all WlsOperators for a given namespace.
func (s simpleWlsOperatorNamespaceLister) List(selector labels.Selector) ([]*v1beta1.WlsOperator, error) {
	var ops []*v1beta1.WlsOperator

	list, err := s.wlsOprClientSet.VerrazzanoV1beta1().WlsOperators(s.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if selector.Matches(labels.Set(list.Items[i].Labels)) {
			ops = append(ops, &list.Items[i])
		}
	}
	return ops, nil
}

// Get retrieves the WlsOperator for a given namespace and name.
func (s simpleWlsOperatorNamespaceLister) Get(name string) (*v1beta1.WlsOperator, error) {
	return s.wlsOprClientSet.VerrazzanoV1beta1().WlsOperators(s.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// ----- fakeWlsOperators
// fakeWlsOperators implements WlsOperatorInterface
type fakeWlsOperators struct {
	ops map[string]*v1beta1.WlsOperator
}

func (c fakeWlsOperators) Create(_ context.Context, wlsOperator *v1beta1.WlsOperator, _ metav1.CreateOptions) (*v1beta1.WlsOperator, error) {
	c.ops[wlsOperator.Name] = wlsOperator
	return wlsOperator, nil
}

func (c fakeWlsOperators) Update(_ context.Context, wlsOperator *v1beta1.WlsOperator, _ metav1.UpdateOptions) (*v1beta1.WlsOperator, error) {
	c.ops[wlsOperator.Name] = wlsOperator
	return wlsOperator, nil
}

func (c fakeWlsOperators) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	delete(c.ops, name)
	return nil
}

func (c fakeWlsOperators) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error {
	c.ops = make(map[string]*v1beta1.WlsOperator)
	return nil
}

func (c fakeWlsOperators) Get(_ context.Context, name string, _ metav1.GetOptions) (*v1beta1.WlsOperator, error) {
	return c.ops[name], nil
}

func (c fakeWlsOperators) List(context.Context, metav1.ListOptions) (*v1beta1.WlsOperatorList, error) {
	items := make([]v1beta1.WlsOperator, 0, len(c.ops))
	for _, item := range c.ops {
		items = append(items, *item)
	}
	return &v1beta1.WlsOperatorList{Items: items}, nil
}

func (c fakeWlsOperators) Watch(context.Context, metav1.ListOptions) (watch.Interface, error) {
	return watch.NewFake(), nil
}

func (c fakeWlsOperators) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (result *v1beta1.WlsOperator, err error) {
	return nil, errors.New("the Patch function is not implemented")
}

// ----- fakeHelidonApps
// fakeHelidonApps implements HelidonAppInterface
type fakeHelidonApps struct {
	apps map[string]*helidionv1beta1.HelidonApp
}

func (c fakeHelidonApps) Create(_ context.Context, helidonApp *helidionv1beta1.HelidonApp, _ metav1.CreateOptions) (*helidionv1beta1.HelidonApp, error) {
	c.apps[helidonApp.Name] = helidonApp
	return helidonApp, nil
}

func (c fakeHelidonApps) Update(_ context.Context, helidonApp *helidionv1beta1.HelidonApp, _ metav1.UpdateOptions) (*helidionv1beta1.HelidonApp, error) {
	c.apps[helidonApp.Name] = helidonApp
	return helidonApp, nil
}

func (c fakeHelidonApps) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	delete(c.apps, name)
	return nil
}

func (c fakeHelidonApps) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	c.apps = make(map[string]*helidionv1beta1.HelidonApp)
	return nil
}

func (c fakeHelidonApps) Get(_ context.Context, name string, _ metav1.GetOptions) (*helidionv1beta1.HelidonApp, error) {
	return c.apps[name], nil
}

func (c fakeHelidonApps) List(_ context.Context, _ metav1.ListOptions) (*helidionv1beta1.HelidonAppList, error) {
	items := make([]helidionv1beta1.HelidonApp, 0, len(c.apps))
	for _, item := range c.apps {
		items = append(items, *item)
	}
	return &helidionv1beta1.HelidonAppList{Items: items}, nil
}

func (c fakeHelidonApps) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return watch.NewFake(), nil
}

func (c fakeHelidonApps) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (result *helidionv1beta1.HelidonApp, err error) {
	return nil, errors.New("the Patch function is not implemented")
}

// ----- fakeCohClusters
// fakeCohClusters implements CohClusterInterface
type fakeCohClusters struct {
	clusters map[string]*cohv1beta1.CohCluster
}

func (c fakeCohClusters) Create(_ context.Context, cohCluster *cohv1beta1.CohCluster, _ metav1.CreateOptions) (*cohv1beta1.CohCluster, error) {
	c.clusters[cohCluster.Name] = cohCluster
	return cohCluster, nil
}

func (c fakeCohClusters) Update(_ context.Context, cohCluster *cohv1beta1.CohCluster, _ metav1.UpdateOptions) (*cohv1beta1.CohCluster, error) {
	c.clusters[cohCluster.Name] = cohCluster
	return cohCluster, nil
}

func (c fakeCohClusters) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	delete(c.clusters, name)
	return nil
}

func (c fakeCohClusters) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error {
	c.clusters = make(map[string]*cohv1beta1.CohCluster)
	return nil
}

func (c fakeCohClusters) Get(_ context.Context, name string, _ metav1.GetOptions) (*cohv1beta1.CohCluster, error) {
	return c.clusters[name], nil
}

func (c fakeCohClusters) List(context.Context, metav1.ListOptions) (*cohv1beta1.CohClusterList, error) {
	items := make([]cohv1beta1.CohCluster, 0, len(c.clusters))
	for _, item := range c.clusters {
		items = append(items, *item)
	}
	return &cohv1beta1.CohClusterList{Items: items}, nil
}

func (c fakeCohClusters) Watch(context.Context, metav1.ListOptions) (watch.Interface, error) {
	return watch.NewFake(), nil
}

func (c fakeCohClusters) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (result *cohv1beta1.CohCluster, err error) {
	return nil, errors.New("the Patch function is not implemented")
}

// ----- fakeWlsVerrazzanoV1beta1Client
// fakeWlsVerrazzanoV1beta1Client is used to interact with features provided by the verrazzano.io group.
type fakeWlsVerrazzanoV1beta1Client struct {
	operators map[string]*fakeWlsOperators
}

func newWlsVerrazzanoV1beta1Client() *fakeWlsVerrazzanoV1beta1Client {
	return &fakeWlsVerrazzanoV1beta1Client{
		operators: make(map[string]*fakeWlsOperators),
	}
}

func (c fakeWlsVerrazzanoV1beta1Client) RESTClient() rest.Interface {
	return &restfake.RESTClient{}
}

func (c fakeWlsVerrazzanoV1beta1Client) WlsOperators(namespace string) wlsoprclientsetbeta1.WlsOperatorInterface {
	if c.operators[namespace] == nil {
		c.operators[namespace] = &fakeWlsOperators{
			ops: make(map[string]*v1beta1.WlsOperator),
		}
	}
	return c.operators[namespace]
}

// ----- fakeHelidonVerrazzanoV1beta1Client
// fakeHelidonVerrazzanoV1beta1Client is used to interact with features provided by the verrazzano.io group.
type fakeHelidonVerrazzanoV1beta1Client struct {
	apps map[string]*fakeHelidonApps
}

func newHelidonVerrazzanoV1beta1() *fakeHelidonVerrazzanoV1beta1Client {
	return &fakeHelidonVerrazzanoV1beta1Client{
		apps: make(map[string]*fakeHelidonApps),
	}
}

func (c fakeHelidonVerrazzanoV1beta1Client) RESTClient() rest.Interface {
	return &restfake.RESTClient{}
}

func (c fakeHelidonVerrazzanoV1beta1Client) HelidonApps(namespace string) helidionclientsetbeta1.HelidonAppInterface {
	if c.apps[namespace] == nil {
		c.apps[namespace] = &fakeHelidonApps{
			apps: make(map[string]*helidionv1beta1.HelidonApp),
		}
	}
	return c.apps[namespace]
}

// ----- fakeCohVerrazzanoV1beta1Client
// fakeCohVerrazzanoV1beta1Client is used to interact with features provided by the verrazzano.io group.
type fakeCohVerrazzanoV1beta1Client struct {
	clusters map[string]*fakeCohClusters
}

func newCohVerrazzanoV1beta1() *fakeCohVerrazzanoV1beta1Client {
	return &fakeCohVerrazzanoV1beta1Client{
		clusters: make(map[string]*fakeCohClusters),
	}
}

func (c fakeCohVerrazzanoV1beta1Client) RESTClient() rest.Interface {
	return &restfake.RESTClient{}
}

func (c fakeCohVerrazzanoV1beta1Client) CohClusters(namespace string) cohoprclientsetbeta1.CohClusterInterface {
	if c.clusters[namespace] == nil {
		c.clusters[namespace] = &fakeCohClusters{
			clusters: make(map[string]*cohv1beta1.CohCluster),
		}
	}
	return c.clusters[namespace]
}

// FakeWlsOprClientset implements verrazzano-wko-operator Interface
type FakeWlsOprClientset struct {
	wlsVerrazzanoV1beta1 *fakeWlsVerrazzanoV1beta1Client
}

// NewWlsOprClientset returns a new WLS operator client set fake
func NewWlsOprClientset() *FakeWlsOprClientset {
	return &FakeWlsOprClientset{
		wlsVerrazzanoV1beta1: newWlsVerrazzanoV1beta1Client(),
	}
}

// Discovery retrieves the DiscoveryClient
func (c FakeWlsOprClientset) Discovery() discovery.DiscoveryInterface {
	return &discoveryfake.FakeDiscovery{}
}

// VerrazzanoV1beta1 returns the fake WlsVerrazzanoV1beta1Client
func (c FakeWlsOprClientset) VerrazzanoV1beta1() wlsoprclientsetbeta1.VerrazzanoV1beta1Interface {
	return c.wlsVerrazzanoV1beta1
}

// FakeHelidionClientset implements verrazzano-helidon-app-operator Interface
type FakeHelidionClientset struct {
	helidonVerrazzanoV1beta1 *fakeHelidonVerrazzanoV1beta1Client
}

// NewHelidionClientset returns a new Helidon client set fake
func NewHelidionClientset() *FakeHelidionClientset {
	return &FakeHelidionClientset{
		helidonVerrazzanoV1beta1: newHelidonVerrazzanoV1beta1(),
	}
}

// Discovery retrieves the DiscoveryClient
func (c FakeHelidionClientset) Discovery() discovery.DiscoveryInterface {
	return &discoveryfake.FakeDiscovery{}
}

// VerrazzanoV1beta1 returns the fake HelidonVerrazzanoV1beta1Client
func (c FakeHelidionClientset) VerrazzanoV1beta1() helidionclientsetbeta1.VerrazzanoV1beta1Interface {
	return c.helidonVerrazzanoV1beta1
}

// FakeCohOprClientset implements verrazzano-coh-cluster-operator Interface
type FakeCohOprClientset struct {
	cohVerrazzanoV1beta1 *fakeCohVerrazzanoV1beta1Client
}

// NewCohOprClientset returns a new Coherence operator client set fake
func NewCohOprClientset() *FakeCohOprClientset {
	return &FakeCohOprClientset{
		cohVerrazzanoV1beta1: newCohVerrazzanoV1beta1(),
	}
}

// Discovery retrieves the DiscoveryClient
func (c FakeCohOprClientset) Discovery() discovery.DiscoveryInterface {
	return &discoveryfake.FakeDiscovery{}
}

// VerrazzanoV1beta1 returns the fake CohVerrazzanoV1beta1Client
func (c FakeCohOprClientset) VerrazzanoV1beta1() cohoprclientsetbeta1.VerrazzanoV1beta1Interface {
	return c.cohVerrazzanoV1beta1
}
