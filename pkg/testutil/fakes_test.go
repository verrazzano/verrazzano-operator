// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"context"
	"fmt"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSimplePodLister(t *testing.T) {
	assert := asserts.New(t)

	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	l := simplePodLister{
		clusterConnection.KubeClient,
	}
	s := labels.Everything()
	pods, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pods: %v", err))
	}
	assert.Equal(4, len(pods))

	namespaceLister := l.Pods("test")

	pods, err = namespaceLister.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pods: %v", err))
	}
	assert.Equal(1, len(pods))

	pod, err := namespaceLister.Get("test-pod")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pod: %v", err))
	}
	assert.Equal("test-pod", pod.Name)
	assert.Equal("test", pod.Namespace)
}

func TestSimpleConfigMapLister(t *testing.T) {
	assert := asserts.New(t)

	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	createConfigMap(t, "cm1", clusterConnection)
	createConfigMap(t, "cm2", clusterConnection)
	createConfigMap(t, "cm3", clusterConnection)

	l := simpleConfigMapLister{
		clusterConnection.KubeClient,
	}
	s := labels.Everything()
	configMaps, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pods: %v", err))
	}
	assert.Equal(3, len(configMaps))

	namespaceLister := l.ConfigMaps("test")

	configMaps, err = namespaceLister.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pods: %v", err))
	}
	assert.Equal(3, len(configMaps))

	configMap, err := namespaceLister.Get("cm1")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get config map: %v", err))
	}
	assert.Equal("cm1", configMap.Name)
	assert.Equal("test", configMap.Namespace)
}

func TestSimpleNamespaceLister(t *testing.T) {
	assert := asserts.New(t)

	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	l := simpleNamespaceLister{
		clusterConnection.KubeClient,
	}
	s := labels.Everything()
	namespaces, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get namespaces: %v", err))
	}
	assert.Equal(6, len(namespaces))

	namespace, err := l.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get namespace: %v", err))
	}
	assert.Equal("test", namespace.Name)
}

// Test SimpleSecretLister
func TestSimpleSecretLister(t *testing.T) {
	assert := asserts.New(t)
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	testNamespace1 := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "",
			Name:      "test-namespace-1",
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	testSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace-1",
			Name:      "test-secret-1"},
		Immutable: nil,
		Data:      nil,
		StringData: map[string]string{
			"test-secret-data-key-1": "test-secret-data-value-1",
		},
		Type: "test-secret-type",
	}

	testSecret2 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace-1",
			Name:      "test-secret-2"},
		Immutable: nil,
		Data:      nil,
		StringData: map[string]string{
			"test-secret-data-key-2": "test-secret-data-value-2",
		},
		Type: "test-secret-type",
	}

	namespace, err := clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace1, metav1.CreateOptions{})
	assert.NoError(err, "Should not err creating namespace.")
	assert.Equal(&testNamespace1, namespace, "Created namespace should match original.")

	secrets, err := clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(err, "Should not err for empty list.")
	assert.Len(secrets, 1, "List should have one entry.")

	secret, err := clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err, "Creating secret should not err.")
	assert.Equal(&testSecret1, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(err, "Should not err for non-empty list.")
	assert.Len(secrets, 2, "List should have two entries.")

	secret, err = clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret2, metav1.CreateOptions{})
	assert.NoError(err, "Creating secret should not err.")
	assert.Equal(&testSecret2, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(err, "Should not err for non-empty list.")
	assert.Len(secrets, 3, "List should have three entries.")
	assert.NotEqual(findByName(secrets, "test-secret-1"), findByName(secrets, "test-secret-2"), "Secrets should be different.")
}

// Test simpleSecretNamespaceLister List and Get
func TestSimpleSecretNamespaceLister(t *testing.T) {
	assert := asserts.New(t)
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	testSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace-1",
			Name:      "test-secret-1"},
		Immutable: nil,
		Data:      nil,
		StringData: map[string]string{
			"test-secret-data-key-1": "test-secret-data-value-1",
		},
		Type: "test-secret-type",
	}

	secrets, err := clusterConnection.SecretLister.Secrets("test-namespace-1").List(labels.Everything())
	assert.NoError(err, "Should not err for empty list.")
	assert.Nil(secrets, "List should be empty/nil.")

	secret, err := clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err, "Creating secret should not err.")
	assert.Equal(&testSecret1, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.Secrets("test-namespace-1").List(labels.Everything())
	assert.NoError(err, "Should not err for non-empty list.")
	assert.Len(secrets, 1, "List should have one entry.")

	secret, err = clusterConnection.SecretLister.Secrets("test-namespace-1").Get("test-secret-1")
	assert.NoError(err, "Should not err for an existing secret.")
	assert.Equal(&testSecret1, secret, "Get should return same secret.")

	secret, err = clusterConnection.SecretLister.Secrets("test-namespace-1").Get("test-secret-invalid")
	assert.Error(err, "Should err for a non-existing secret.")
	assert.Nil(secret, "Should return nil for a non-existing secret.")
}

func TestSimpleServiceLister(t *testing.T) {
	assert := asserts.New(t)

	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "metrics",
					Port:       9100,
					TargetPort: intstr.FromInt(9100),
					Protocol:   "TCP",
				},
			},
			Type: "ClusterIP",
		},
	}
	_, err := clusterConnection.KubeClient.CoreV1().Services("test").Create(context.TODO(), &svc, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service: %v", err))
	}

	l := simpleServiceLister{
		clusterConnection.KubeClient,
	}

	s := labels.Everything()
	services, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get services: %v", err))
	}
	// testdata includes the istio-ingressgateway service so expect 2 services
	assert.Equal(2, len(services))

	namespaceLister := l.Services("test")

	services, err = namespaceLister.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get services: %v", err))
	}
	assert.Equal(1, len(services))

	service, err := namespaceLister.Get("test-service")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service: %v", err))
	}
	assert.Equal("test-service", service.Name)
	assert.Equal("test", service.Namespace)
}

// TestSimpleServiceAccountLister tests the functionality of simpleServiceAccountLister.
// GIVEN a cluster which has no existing service accounts
//  WHEN I create one service account and a simpleServiceAccountLister
//  THEN the lister should list one service account given an everything selector
//   AND the lister should list exactly one specific service account given a label selector
//   AND the lister should get a specific service account when requested by name
func TestSimpleServiceAccountLister(t *testing.T) {
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	assert := asserts.New(t)

	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-serviceAccount",
			Namespace: "test",
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test-imagePullSecret",
			},
		},
	}
	_, err := clusterConnection.KubeClient.CoreV1().ServiceAccounts("test").Create(context.TODO(), &sa, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	l := simpleServiceAccountLister{
		clusterConnection.KubeClient,
	}

	s := labels.Everything()
	serviceAccounts, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service accounts: %v", err))
	}
	assert.Equal(1, len(serviceAccounts))

	namespaceLister := l.ServiceAccounts("test")

	serviceAccounts, err = namespaceLister.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service accounts: %v", err))
	}
	assert.Equal(1, len(serviceAccounts))

	serviceAccount, err := namespaceLister.Get("test-serviceAccount")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service account: %v", err))
	}
	assert.Equal("test-serviceAccount", serviceAccount.Name)
	assert.Equal("test", serviceAccount.Namespace)
}

// TestSimpleClusterRoleLister tests the functionality of simpleClusterRoleLister
// GIVEN a cluster which has no existing cluster roles
//  WHEN I create two cluster roles and a simpleClusterRoleLister
//  THEN the lister should list exactly two cluster roles given an everything selector
//   AND the lister should list exactly one specific cluster role given a label selector
//   AND the lister should get a specific cluster role when requested by name
func TestSimpleClusterRoleLister(t *testing.T) {
	assert := asserts.New(t)

	// create a map of empty test cluster connections that use fakes
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// add some cluster roles through the fake client set on the test cluster
	cr := v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				"label1": "foo",
			},
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to a create cluster role: %v", err)
	}

	cr = v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test2",
		},
	}
	_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to create cluster role: %v", err)
	}

	// create a simpleClusterRoleLister using the fake client set from the test cluster
	l := simpleClusterRoleLister{
		clusterConnection.KubeClient,
	}

	// list all the cluster roles through the lister
	s := labels.Everything()
	clusterRoles, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to list the cluster roles: %v", err))
	}
	assert.Len(clusterRoles, 2, "expected 2 cluster roles in the list")
	expectedRoleSet := map[string]struct{}{"test": {}, "test2": {}}
	for _, clusterRole := range clusterRoles {
		delete(expectedRoleSet, clusterRole.Name)
	}
	assert.Len(expectedRoleSet, 0, "not all of the expected roles were returned: %v", expectedRoleSet)

	// list cluster roles through the lister with a label selector
	requirement, err := labels.NewRequirement("label1", selection.Equals, []string{"foo"})
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to create a requirement: %v", err))
	}
	s = labels.NewSelector().Add(*requirement)
	clusterRoles, err = l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to list the cluster roles: %v", err))
	}
	assert.Len(clusterRoles, 1, "expected 1 cluster role in the list")
	assert.Equal("test", clusterRoles[0].Name, "expected the cluster to be named test")

	// get specific cluster roles through the lister
	clusterRole, err := l.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to get a cluster role: %v", err))
	}
	assert.Equal("test", clusterRole.Name, "expected the cluster to be named test")

	clusterRole, err = l.Get("test2")
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to get a cluster role: %v", err))
	}
	assert.Equal("test2", clusterRole.Name, "expected the cluster to be named test")
}

// TestSimpleClusterRoleBindingLister tests the functionality of simpleClusterRoleBindingLister
// GIVEN a cluster which has no existing cluster role bindings
//  WHEN I create two cluster role bindings and a simpleClusterRoleBindingLister
//  THEN the lister should list exactly two cluster role bindings given an everything selector
//   AND the lister should list exactly one specific cluster role binding given a label selector
//   AND the lister should get a specific cluster role binding when requested by name
func TestSimpleClusterRoleBindingLister(t *testing.T) {
	assert := asserts.New(t)

	// create a map of empty test cluster connections that use fakes
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// add some cluster role bindings through the fake client set on the test cluster
	cr := v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				"label1": "foo",
			},
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), &cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to a create cluster role binding: %v", err)
	}

	cr = v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test2",
		},
	}
	_, err = clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), &cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to create cluster role binding: %v", err)
	}

	// create a simpleClusterRoleBindingLister using the fake client set from the test cluster
	l := simpleClusterRoleBindingLister{
		clusterConnection.KubeClient,
	}

	// list all the cluster role bindings through the lister
	s := labels.Everything()
	clusterRoleBindings, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to list the cluster role bindings: %v", err))
	}
	assert.Len(clusterRoleBindings, 2, "expected 2 cluster role bindings in the list")
	expectedBindingSet := map[string]struct{}{"test": {}, "test2": {}}
	for _, clusterRoleBinding := range clusterRoleBindings {
		delete(expectedBindingSet, clusterRoleBinding.Name)
	}
	assert.Len(expectedBindingSet, 0, "not all of the expected role bindings were returned: %v", expectedBindingSet)

	// list cluster role bindings through the lister with a label selector
	requirement, err := labels.NewRequirement("label1", selection.Equals, []string{"foo"})
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to create a requirement: %v", err))
	}
	s = labels.NewSelector().Add(*requirement)
	clusterRoleBindings, err = l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to list the cluster role bindings: %v", err))
	}
	assert.Len(clusterRoleBindings, 1, "expected 1 cluster role binding in the list")
	assert.Equal("test", clusterRoleBindings[0].Name, "expected the cluster role binding to be named test")

	// get specific cluster role bindings through the lister
	clusterRoleBinding, err := l.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to get a cluster role binding: %v", err))
	}
	assert.Equal("test", clusterRoleBinding.Name, "expected the cluster role binding to be named test")

	clusterRoleBinding, err = l.Get("test2")
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to get a cluster role binding: %v", err))
	}
	assert.Equal("test2", clusterRoleBinding.Name, "expected the cluster role binding to be named test")
}

// TestSimpleDaemonSetLister tests the functionality of simpleDaemonSetLister
// GIVEN a cluster which has no existing daemon sets
//  WHEN I create two daemon sets and a simpleDaemonSetLister
//  THEN the lister should list exactly two daemon sets given an everything selector
//   AND the lister should list exactly one specific daemon set given a label selector
//   AND the lister should get a specific daemon set when requested by name
func TestSimpleDaemonSetLister(t *testing.T) {
	assert := asserts.New(t)

	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	ds := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels: map[string]string{
				"label1": "foo",
			},
		},
	}
	_, err := clusterConnection.KubeClient.AppsV1().DaemonSets("test").Create(context.TODO(), &ds, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create daemon sets: %v", err))
	}

	ds = appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2",
			Namespace: "test2",
		},
	}
	_, err = clusterConnection.KubeClient.AppsV1().DaemonSets("test2").Create(context.TODO(), &ds, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create daemon sets: %v", err))
	}

	l := simpleDaemonSetLister{
		clusterConnection.KubeClient,
	}
	selector := labels.Everything()
	// get the daemon set list for all namespaces
	sets, err := l.List(selector)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get daemon sets: %v", err))
	}
	assert.Len(sets, 2, "expected 2 daemon sets")

	// list daemon sets through the lister with a label selector
	requirement, err := labels.NewRequirement("label1", selection.Equals, []string{"foo"})
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to create a requirement: %v", err))
	}
	selector = labels.NewSelector().Add(*requirement)
	sets, err = l.List(selector)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list daemon sets: %v", err))
	}
	assert.Len(sets, 1, "expected 1 daemon set")
	assert.Equal("test", sets[0].Name, "unexpected daemon set name")

	// get the list for the 'test' namespace
	namespaceLister := l.DaemonSets("test")
	daemonSets, err := namespaceLister.List(selector)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get daemon sets: %v", err))
	}
	assert.Len(daemonSets, 1, "expected 1 daemon set")
	assert.Equal("test", sets[0].Name, "unexpected daemon set name")

	// get a daemon set by name
	pod, err := namespaceLister.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get daemon sets: %v", err))
	}
	assert.Equal("test", pod.Name, "unexpected daemon set name")
	assert.Equal("test", pod.Namespace, "unexpected daemon set namespace")
}

// TestSimpleDeploymentLister tests the functionality of simpleDeploymentLister
// GIVEN a cluster which has no existing deployments
//  WHEN I create two deployments and a simpleDeploymentLister
//  THEN the lister should list exactly two deployments given an everything selector
//   AND the lister should list exactly one specific deployment given a label selector
//   AND the lister should get a specific deployment when requested by name
func TestSimpleDeploymentLister(t *testing.T) {
	assert := asserts.New(t)

	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	ds := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels: map[string]string{
				"label1": "foo",
			},
		},
	}
	_, err := clusterConnection.KubeClient.AppsV1().Deployments("test").Create(context.TODO(), &ds, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create deployments: %v", err))
	}

	ds = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2",
			Namespace: "test2",
		},
	}
	_, err = clusterConnection.KubeClient.AppsV1().Deployments("test2").Create(context.TODO(), &ds, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create deployments: %v", err))
	}

	l := simpleDeploymentLister{
		clusterConnection.KubeClient,
	}
	selector := labels.Everything()
	// get the deployment list for all namespaces
	sets, err := l.List(selector)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get deployments: %v", err))
	}
	assert.Len(sets, 2, "expected 2 deployments")

	// list deployments through the lister with a label selector
	requirement, err := labels.NewRequirement("label1", selection.Equals, []string{"foo"})
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to create a requirement: %v", err))
	}
	selector = labels.NewSelector().Add(*requirement)
	sets, err = l.List(selector)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list deployments: %v", err))
	}
	assert.Len(sets, 1, "expected 1 deployment")
	assert.Equal("test", sets[0].Name, "unexpected deployment name")

	// get the list for the 'test' namespace
	namespaceLister := l.Deployments("test")
	deployments, err := namespaceLister.List(selector)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get deployments: %v", err))
	}
	assert.Len(deployments, 1, "expected 1 deployment")
	assert.Equal("test", sets[0].Name, "unexpected deployment name")

	// get a deployment by name
	pod, err := namespaceLister.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get deployments: %v", err))
	}
	assert.Equal("test", pod.Name, "unexpected deployment name")
	assert.Equal("test", pod.Namespace, "unexpected deployment namespace")
}

// TestFakeSecrets tests the functionality of FakeSecrets.
// GIVEN a FakeSecrets instance
//  WHEN initialized with 3 secrets
//  THEN I should be able to get a secret by name
//   AND I should be able to create a new secret on the FakeSecrets
//   AND I should be able to update an existing secret
//   AND I should be able to list secrets with a selector
//   AND I should be able to delete a secret
func TestFakeSecrets(t *testing.T) {
	assert := asserts.New(t)

	secrets := &FakeSecrets{Secrets: map[string]*corev1.Secret{
		"secret1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret1",
				Namespace: "test",
				Labels:    map[string]string{"foo": "aaa", "bar": "bbb"},
			},
		},
		"secret2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret2",
				Namespace: "test",
				Labels:    map[string]string{"foo": "aaa", "bar": "ccc"},
			},
		},
		"secret3": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret3",
				Namespace: "test2",
				Labels:    map[string]string{"foo": "ddd", "bar": "eee"},
			},
		},
	}}

	// test Get
	secret, err := secrets.Get("secret1")
	assert.Nil(err, "error getting secret")
	assert.Equal("secret1", secret.Name, "unexpected secret name")
	assert.Equal("test", secret.Namespace, "unexpected secret namespace")

	// test Create
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret4",
			Namespace: "test",
			Labels:    map[string]string{"foo": "ddd", "bar": "fff"},
		},
	}
	secret, err = secrets.Create(secret)
	assert.Nil(err, "error creating secret")
	assert.Equal("secret4", secret.Name, "unexpected secret name")
	assert.Equal("test", secret.Namespace, "unexpected secret namespace")

	// test Update
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret4",
			Namespace: "test2",
			Labels:    map[string]string{"foo": "ddd", "bar": "fff"},
		},
	}
	secret, err = secrets.Update(secret)
	assert.Nil(err, "error creating secret")
	assert.Equal("secret4", secret.Name, "unexpected secret name")
	assert.Equal("test2", secret.Namespace, "unexpected secret namespace")

	// test List
	selector := labels.Everything()
	list, err := secrets.List("test", selector)
	assert.Nil(err, "error listing secrets")
	expectedNames := map[string]struct{}{"secret1": {}, "secret2": {}}
	assert.Len(list, len(expectedNames), "list of secrets does not match expected")
	for _, secret := range list {
		assert.Contains(expectedNames, secret.Name, "expected secret not found")
	}

	// list secrets  with a label selector
	requirement, err := labels.NewRequirement("foo", selection.Equals, []string{"ddd"})
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to create a requirement: %v", err))
	}
	selector = labels.NewSelector().Add(*requirement)
	list, err = secrets.List("test2", selector)
	assert.Nil(err, "error listing secrets")
	expectedNames = map[string]struct{}{"secret3": {}, "secret4": {}}
	assert.Len(list, len(expectedNames), "list of secrets does not match expected")
	for _, secret := range list {
		assert.Contains(expectedNames, secret.Name, "expected secret not found")
	}

	// test GetVmiPassword
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSecretName,
		},
		Data: map[string][]byte{
			"password": []byte("TestTest"),
		},
	}
	secret, err = secrets.Create(secret)
	assert.Nil(err, "error creating secret")
	assert.Equal(constants.VmiSecretName, secret.Name, "unexpected secret name")
	p, err := secrets.GetVmiPassword()
	assert.Nil(err, "error calling GetVmiPassword")
	assert.Equal("TestTest", p, "unexpected test value for VmiPassword")

	// test Delete
	err = secrets.Delete("test2", "secret4")
	assert.Nil(err, "error deleting secret")
	secret, err = secrets.Get("secret4")
	assert.Nil(err, "error getting secret")
	assert.Nil(secret, "expected secret to be deleted")
}

func createConfigMap(t *testing.T, name string, clusterConnection *util.ManagedClusterConnection) {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
			Labels: map[string]string{
				"verrazzano.binding": "testBinding",
				"verrazzano.cluster": "cluster3",
			},
		},
	}
	_, err := clusterConnection.KubeClient.CoreV1().ConfigMaps("test").Create(context.TODO(), &cm, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("can't create config map: %v", err)
	}
}

//findByName returns the secret with the given name
func findByName(secrets []*corev1.Secret, name string) *corev1.Secret {
	for _, secret := range secrets {
		if secret.Name == name {
			return secret
		}
	}
	return nil
}
