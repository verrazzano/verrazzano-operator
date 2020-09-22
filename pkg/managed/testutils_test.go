// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	istio "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestContains(t *testing.T) {
	s := []string{"foo", "bar", "baz"}
	assert.True(t, contains(s, "foo"))
	assert.True(t, contains(s, "bar"))
	assert.True(t, contains(s, "baz"))
	assert.False(t, contains(s, "biz"))
}

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

<<<<<<< HEAD
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

=======
>>>>>>> master
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
	assert.NoError(t, err, "Should not err creating namespace.")
	assert.Equal(t, &testNamespace1, namespace, "Created namespace should match original.")

	secrets, err := clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(t, err, "Should not err for empty list.")
	assert.Nil(t, secrets, "List should be empty/nil.")

	secret, err := clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(t, err, "Creating secret should not err.")
	assert.Equal(t, &testSecret1, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(t, err, "Should not err for non-empty list.")
	assert.Len(t, secrets, 1, "List should have one entry.")

	secret, err = clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret2, metav1.CreateOptions{})
	assert.NoError(t, err, "Creating secret should not err.")
	assert.Equal(t, &testSecret2, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.List(labels.Everything())
	assert.NoError(t, err, "Should not err for non-empty list.")
	assert.Len(t, secrets, 2, "List should have one entry.")
	assert.NotEqual(t, secrets[0], secrets[1], "Secrets should be different.")
}

// Test simpleSecretNamespaceLister List and Get
func TestSimpleSecretNamespaceLister(t *testing.T) {
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
	assert.NoError(t, err, "Should not err for empty list.")
	assert.Nil(t, secrets, "List should be empty/nil.")

	secret, err := clusterConnection.KubeClient.CoreV1().Secrets("test-namespace-1").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(t, err, "Creating secret should not err.")
	assert.Equal(t, &testSecret1, secret, "Create should return same secret.")

	secrets, err = clusterConnection.SecretLister.Secrets("test-namespace-1").List(labels.Everything())
	assert.NoError(t, err, "Should not err for non-empty list.")
	assert.Len(t, secrets, 1, "List should have one entry.")

	secret, err = clusterConnection.SecretLister.Secrets("test-namespace-1").Get("test-secret-1")
	assert.NoError(t, err, "Should not err for an existing secret.")
	assert.Equal(t, &testSecret1, secret, "Get should return same secret.")

	secret, err = clusterConnection.SecretLister.Secrets("test-namespace-1").Get("test-secret-invalid")
	assert.Error(t, err, "Should err for a non-existing secret.")
	assert.Nil(t, secret, "Should return nil for a non-existing secret.")
}
