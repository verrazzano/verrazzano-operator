// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"

	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	appslistersv1 "k8s.io/client-go/listers/apps/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
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

