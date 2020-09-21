// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	istio "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"testing"
)

func TestContains(t *testing.T) {
	s := []string{"foo", "bar", "baz"}
	assert.True(t, contains(s, "foo"))
	assert.True(t, contains(s, "bar"))
	assert.True(t, contains(s, "baz"))
	assert.False(t, contains(s, "biz"))
}

func TestSimplePodLister(t *testing.T) {
	clusterConnections := getManagedClusterConnections()
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

func TestSimpleGatewayLister(t *testing.T) {
	clusterConnections := getManagedClusterConnections()
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
	clusterConnections := getManagedClusterConnections()
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
	clusterConnections := getManagedClusterConnections()
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
