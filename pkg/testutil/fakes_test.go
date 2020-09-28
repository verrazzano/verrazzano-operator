// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"context"
	"fmt"
	"testing"

	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/stretchr/testify/assert"
	istio "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSimplePodLister(t *testing.T) {
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
	assert.Equal(t, 4, len(pods))

	nsl := l.Pods("test")

	pods, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pods: %v", err))
	}
	assert.Equal(t, 1, len(pods))

	pod, err := nsl.Get("test-pod")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pod: %v", err))
	}
	assert.Equal(t, "test-pod", pod.Name)
	assert.Equal(t, "test", pod.Namespace)
}

func TestSimpleConfigMapLister(t *testing.T) {
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
	assert.Equal(t, 3, len(configMaps))

	nsl := l.ConfigMaps("test")

	configMaps, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get pods: %v", err))
	}
	assert.Equal(t, 3, len(configMaps))

	configMap, err := nsl.Get("cm1")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get config map: %v", err))
	}
	assert.Equal(t, "cm1", configMap.Name)
	assert.Equal(t, "test", configMap.Namespace)
}

func TestSimpleNamespaceLister(t *testing.T) {
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
	assert.Equal(t, 4, len(namespaces))

	namespace, err := l.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get namespace: %v", err))
	}
	assert.Equal(t, "test", namespace.Name)
}

func TestSimpleGatewayLister(t *testing.T) {
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	gw := istio.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gateway",
			Namespace: "test",
		},
	}
	_, err := clusterConnection.IstioClientSet.NetworkingV1alpha3().Gateways("test").Create(context.TODO(), &gw, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create gateway: %v", err))
	}

	l := simpleGatewayLister{
		clusterConnection.KubeClient,
		clusterConnection.IstioClientSet,
	}
	s := labels.Everything()
	gateways, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get gateways: %v", err))
	}
	assert.Equal(t, 1, len(gateways))

	nsl := l.Gateways("test")

	gateways, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get gateways: %v", err))
	}
	assert.Equal(t, 1, len(gateways))

	gateway, err := nsl.Get("test-gateway")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get gateway: %v", err))
	}
	assert.Equal(t, "test-gateway", gateway.Name)
	assert.Equal(t, "test", gateway.Namespace)
}

func TestSimpleVirtualServiceLister(t *testing.T) {
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	vs := istio.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test",
		},
	}
	_, err := clusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices("test").Create(context.TODO(), &vs, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service: %v", err))
	}

	l := simpleVirtualServiceLister{
		clusterConnection.KubeClient,
		clusterConnection.IstioClientSet,
	}
	s := labels.Everything()
	services, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get services: %v", err))
	}
	assert.Equal(t, 1, len(services))

	nsl := l.VirtualServices("test")

	services, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get services: %v", err))
	}
	assert.Equal(t, 1, len(services))

	service, err := nsl.Get("test-service")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service: %v", err))
	}
	assert.Equal(t, "test-service", service.Name)
	assert.Equal(t, "test", service.Namespace)
}

func TestSimpleServiceEntryLister(t *testing.T) {
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	se := istio.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-entry",
			Namespace: "test",
		},
	}
	_, err := clusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries("test").Create(context.TODO(), &se, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create entry: %v", err))
	}

	l := simpleServiceEntryLister{
		clusterConnection.KubeClient,
		clusterConnection.IstioClientSet,
	}
	s := labels.Everything()
	entries, err := l.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get entries: %v", err))
	}
	assert.Equal(t, 1, len(entries))

	nsl := l.ServiceEntries("test")

	entries, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get entries: %v", err))
	}
	assert.Equal(t, 1, len(entries))

	entry, err := nsl.Get("test-entry")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get entry: %v", err))
	}
	assert.Equal(t, "test-entry", entry.Name)
	assert.Equal(t, "test", entry.Namespace)
}

// Test simpleSecretLister
func TestSimpleSecretLister(t *testing.T) {
	assert := assert.New(t)
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
	assert.Nil(secrets, "List should be empty/nil.")

	secret, err := clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err, "Creating secret should not err.")
	assert.Equal(&testSecret1, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(err, "Should not err for non-empty list.")
	assert.Len(secrets, 1, "List should have one entry.")

	secret, err = clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret2, metav1.CreateOptions{})
	assert.NoError(err, "Creating secret should not err.")
	assert.Equal(&testSecret2, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(err, "Should not err for non-empty list.")
	assert.Len(secrets, 2, "List should have one entry.")
	assert.NotEqual(secrets[0], secrets[1], "Secrets should be different.")
}

// Test simpleSecretNamespaceLister List and Get
func TestSimpleSecretNamespaceLister(t *testing.T) {
	assert := assert.New(t)
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
	assert.Equal(t, 2, len(services))

	nsl := l.Services("test")

	services, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get services: %v", err))
	}
	assert.Equal(t, 1, len(services))

	service, err := nsl.Get("test-service")
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service: %v", err))
	}
	assert.Equal(t, "test-service", service.Name)
	assert.Equal(t, "test", service.Namespace)
}

func TestSimpleServiceAccountLister(t *testing.T) {
	clusterConnections := GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	assert := assert.New(t)

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

	nsl := l.ServiceAccounts("test")

	serviceAccounts, err = nsl.List(s)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't get service accounts: %v", err))
	}
	assert.Equal(1, len(serviceAccounts))

	serviceAccount, err := nsl.Get("test-serviceAccount")
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
	assert.Len(t, clusterRoles, 2, "expected 2 cluster roles in the list")
	expectedRoleSet := map[string]struct{}{"test": {}, "test2": {}}
	for _, clusterRole := range clusterRoles {
		delete(expectedRoleSet, clusterRole.Name)
	}
	assert.Len(t, expectedRoleSet, 0, "not all of the expected roles were returned: %v", expectedRoleSet)

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
	assert.Len(t, clusterRoles, 1, "expected 1 cluster role in the list")
	assert.Equal(t, "test", clusterRoles[0].Name, "expected the cluster to be named test")

	// get specific cluster roles through the lister
	clusterRole, err := l.Get("test")
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to get a cluster role: %v", err))
	}
	assert.Equal(t, "test", clusterRole.Name, "expected the cluster to be named test")

	clusterRole, err = l.Get("test2")
	if err != nil {
		t.Fatal(fmt.Sprintf("got an error trying to get a cluster role: %v", err))
	}
	assert.Equal(t, "test2", clusterRole.Name, "expected the cluster to be named test")
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
