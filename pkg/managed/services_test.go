// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestCreateServicesVMIBinding tests the creation of services for a VMI system binding
// GIVEN a cluster which does not have any services
//  WHEN I call CreateServices
//  THEN there should be a Node Exporter service created
func TestCreateServicesVMIBinding(t *testing.T) {
	assert := assert.New(t)

	mbPairSystemBinding := testutil.GetModelBindingPair()
	mbPairSystemBinding.Model.Name = constants.VmiSystemBindingName
	mbPairSystemBinding.Binding.Name = constants.VmiSystemBindingName

	// Model/binding for VMI system binding - Node Exporter service is created.
	managedConnections := testutil.GetManagedClusterConnections()
	err := CreateServices(mbPairSystemBinding, managedConnections)
	assert.Nil(err, "got an error from CreateServices: %v", err)

	for clusterName := range mbPairSystemBinding.ManagedClusters {
		managedClusterConnection := managedConnections[clusterName]
		existingService, err := managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).List(context.TODO(), metav1.ListOptions{})
		assert.Nil(err, "got an error listing services: %v", err)
		assertCreateServiceVMI(t, existingService, clusterName)

		// Update the port value so when we call CreateServices again the update code is executed.
		existingService.Items[0].Spec.Ports[0].Port = 9200
		_, err = managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).Update(context.TODO(), &existingService.Items[0], metav1.UpdateOptions{})
		assert.Nil(err, "got an error updating services: %v", err)
	}

	// Model/binding for VMI system binding - service already exist
	err = CreateServices(mbPairSystemBinding, managedConnections)
	assert.Nil(err, "got an error from CreateServices: %v", err)

	for clusterName := range mbPairSystemBinding.ManagedClusters {
		managedClusterConnection := managedConnections[clusterName]
		existingService, err := managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).List(context.TODO(), metav1.ListOptions{})
		assert.Nil(err, "got an error listing services: %v", err)
		assertCreateServiceVMI(t, existingService, clusterName)
	}
}

// TestDeleteServices tests that an existing service is properly deleted.
// GIVEN a cluster which has an existing generic component service (test-generic)
//  WHEN I call DeleteServices
//  THEN we delete only the service (test-generic) for the generic component
func TestDeleteServices(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	err := CreateServices(modelBindingPair, clusterConnections)
	assert.Nil(err, "got an error from CreateServices: %v", err)

	services, err := clusterConnection.KubeClient.CoreV1().Services("test").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got an error listing services: %v", err)

	err = DeleteServices(modelBindingPair, clusterConnections)
	assert.Nil(err, "got an error from DeleteServices: %v", err)
	services, err = clusterConnection.KubeClient.CoreV1().Services("test").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got an error listing services: %v", err)
	assert.Equal(0, len(services.Items), "expected exactly 0 service in the test namespace")
}


func assertCreateServiceVMI(t *testing.T, existingService *corev1.ServiceList, clusterName string) {
	labels := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}

	assert := assert.New(t)

	assert.Equal(1, len(existingService.Items), "one service should be found for cluster %s", clusterName)
	assert.Equal(constants.NodeExporterName, existingService.Items[0].Name, "service name not equal to expected value")
	assert.Equal(constants.MonitoringNamespace, existingService.Items[0].Namespace, "service namespace not equal to expected value")
	assert.Equal(monitoring.GetNodeExporterLabels(clusterName), existingService.Items[0].Labels, "labels not equal to expected value")
	assert.Equal(labels, existingService.Items[0].Spec.Selector, "selector not equal to expected value")
	assert.Equal(corev1.ServiceType("ClusterIP"), existingService.Items[0].Spec.Type, "service type not equal to expected value")
	assert.Equal("metrics", existingService.Items[0].Spec.Ports[0].Name, "port name not equal to expected value")
	assert.Equal(int32(9100), existingService.Items[0].Spec.Ports[0].Port, "port not equal to expected value")
	assert.Equal(intstr.FromInt(9100), existingService.Items[0].Spec.Ports[0].TargetPort, "target port not equal to expected value")
	assert.Equal(corev1.Protocol("TCP"), existingService.Items[0].Spec.Ports[0].Protocol, "protocol name not equal to expected value")
}
