// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
)

// TestCreateAppServiceAccounts tests creation of service accounts when an application binding is used.
// GIVEN clusters which do not have service accounts for each namespace of an application binding
//  WHEN I call CreateServiceAccounts
//  THEN there should be a service account created for each namespace of an application binding
//   AND that service account should have the expected values
func TestCreateAppServiceAccounts(t *testing.T) {
	// Setup the needed structures to test creating service accounts for an application binding.
	imagePullSecrets := []corev1.LocalObjectReference{
		{
			Name: "test-imagePullSecret",
		},
	}
	managedClusters := getManagedClusters()
	managedConnections := getManagedConnections()
	createAppNamespaces(t, managedConnections)

	// Create the expected service accounts
	err := CreateServiceAccounts("testBinding", imagePullSecrets, managedClusters, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	// Validate the service accounts created for the "local" cluster
	managedClusterConnection := managedConnections["local"]
	serviceAccounts, err := managedClusterConnection.ServiceAccountLister.List(k8slabels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster local: %v", err))
	}
	assert.Equal(t, 2, len(serviceAccounts), "2 service accounts were expected for cluster local")
	for _, sa := range serviceAccounts {
		assertCreateAppServiceAccounts(t, sa, "local")
	}

	// Validate the service accounts created for the "cluster-1" cluster
	managedClusterConnection = managedConnections["cluster-1"]
	serviceAccounts, err = managedClusterConnection.ServiceAccountLister.List(k8slabels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster cluster-1: %v", err))
	}
	assert.Equal(t, 1, len(serviceAccounts), "1 service account was expected for cluster cluster-1")
	for _, sa := range serviceAccounts {
		assertCreateAppServiceAccounts(t, sa, "cluster-1")
	}

	// Update the imagePullSecret for service account we created in the "cluster-1" cluster.
	serviceAccounts[0].ImagePullSecrets[0].Name = ""
	_, err = managedClusterConnection.KubeClient.CoreV1().ServiceAccounts("ns-test3").Update(context.TODO(), serviceAccounts[0], metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't update service account for cluster cluster-1: %v", err))
	}

	// Call CreateServiceAccounts again, the update code will be executed to make the service account right.
	err = CreateServiceAccounts("testBinding", imagePullSecrets, managedClusters, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	// Validate the service account was updated for the "cluster-1" cluster
	serviceAccounts, err = managedClusterConnection.ServiceAccountLister.List(k8slabels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster cluster-1: %v", err))
	}
	assert.Equal(t, 1, len(serviceAccounts), "1 service account was expected for cluster cluster-1")
	for _, sa := range serviceAccounts {
		assertCreateAppServiceAccounts(t, sa, "cluster-1")
	}
}

// TestCreateVMIServiceAccounts tests creation of service accounts when the binding name is 'system'
// GIVEN clusters which do not have service accounts for each namespace of a VMI binding
//  WHEN I call CreateServiceAccounts
//  THEN there should be a service account created for each namespace of a VMI binding
//   AND that service account should have the expected values
func TestCreateVMIServiceAccounts(t *testing.T) {
	assert := assert.New(t)

	// Setup the needed structures to test creating service accounts for a VMI binding.
	managedClusters := getManagedClusters()
	managedConnections := getManagedConnections()

	// Create the expected service accounts
	err := CreateServiceAccounts(constants.VmiSystemBindingName, []corev1.LocalObjectReference{}, managedClusters, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	// Validate the service accounts created for the "local" cluster
	managedClusterConnection := managedConnections["local"]
	serviceAccounts, err := managedClusterConnection.ServiceAccountLister.List(k8slabels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster local: %v", err))
	}
	assert.Equal(3, len(serviceAccounts), "3 service accounts were expected for cluster local")

	for i, sa := range serviceAccounts {
		assertCreateVMIServiceAccounts(t, sa, i, "local")
	}

	// Validate the service accounts created for the "cluster-1" cluster
	managedClusterConnection = managedConnections["cluster-1"]
	serviceAccounts, err = managedClusterConnection.ServiceAccountLister.List(k8slabels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster local: %v", err))
	}
	assert.Equal(3, len(serviceAccounts), "3 service accounts were expected for cluster cluster-1")

	for i, sa := range serviceAccounts {
		assertCreateVMIServiceAccounts(t, sa, i, "cluster-1")
	}
}

// getManagedClusters returns the managed clusters needed for tests.
func getManagedClusters() map[string]*types.ManagedCluster {
	return map[string]*types.ManagedCluster{
		"local": {
			Name:       "local",
			Namespaces: []string{"ns-test1", "ns-test2"},
		},
		"cluster-1": {
			Name:       "cluster-1",
			Namespaces: []string{"ns-test3"},
		},
	}
}

// getManagedConnections returns the managed connections needed for tests.
func getManagedConnections() map[string]*util.ManagedClusterConnection {
	return map[string]*util.ManagedClusterConnection{
		"local":     testutil.GetManagedClusterConnection("local"),
		"cluster-1": testutil.GetManagedClusterConnection("cluster-1"),
	}
}

// createAppNamespaces creates the namespaces needed for an application binding.
func createAppNamespaces(t *testing.T, managedConnections map[string]*util.ManagedClusterConnection) {
	var ns = corev1.Namespace{}

	clusterConnection := managedConnections["local"]
	ns.Name = "ns-test1"
	_, err := clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create namespace %s: %v", ns.Name, err))
	}
	ns.Name = "ns-test2"
	_, err = clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create namespace %s: %v", ns.Name, err))
	}

	clusterConnection = managedConnections["cluster-1"]
	ns.Name = "ns-test3"
	_, err = clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create namespace %s: %v", ns.Name, err))
	}
}

// assertCreateAppServiceAccounts asserts that the given service account for an application binding has
// the expected values.
func assertCreateAppServiceAccounts(t *testing.T, sa *corev1.ServiceAccount, clusterName string) {
	assert := assert.New(t)

	labels := util.GetManagedLabelsNoBinding(clusterName)

	assert.Equal(util.GetServiceAccountNameForSystem(), sa.Name, "service account name for namespace %s was not as expected", sa.Namespace)
	assert.Equal(labels, sa.Labels, "service account label for namespace %s was not as expected", sa.Namespace)
	assert.Equal("test-imagePullSecret", sa.ImagePullSecrets[0].Name, "service account image pull secret for namespace %s was not as expected", sa.Namespace)
}

// assertCreateVMIServiceAccounts asserts that the given service account for a VMI binding has the expected values.
func assertCreateVMIServiceAccounts(t *testing.T, sa *corev1.ServiceAccount, index int, clusterName string) {
	assert := assert.New(t)

	labels := monitoring.GetMonitoringComponentLabels(clusterName, sa.Name)

	assert.Equal(labels, sa.Labels, "service account label for namespace %s was not as expected", sa.Namespace)
	assert.Equal([]corev1.LocalObjectReference{}, sa.ImagePullSecrets, "imagePullSecrets for service account should be empty")
	if index == 0 {
		assert.Equal(constants.FilebeatName, sa.Name, "service account has wrong name")
	} else if index == 1 {
		assert.Equal(constants.JournalbeatName, sa.Name, "service account has wrong name")
	} else if index == 2 {
		assert.Equal(constants.NodeExporterName, sa.Name, "service account has wrong name")
	}
}
