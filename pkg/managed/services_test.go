// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestCreateServicesAppBinding model/binding is for application binding - no service is created.
func TestCreateServicesAppBinding(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	err := CreateServices(modelBindingPair, clusterConnections)
	assert.Nil(err, "got an error from CreateServices: %v", err)

	services, err := clusterConnection.KubeClient.CoreV1().Services("test").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got an error listing services: %v", err)
	assert.Equal(1, len(services.Items), "expected exactly 1 service in the test namespace")
	assert.Equal("test-generic", services.Items[0].Name, "service name not equal to expected value")
	assert.Equal("test", services.Items[0].Namespace, "service namespace not equal to expected value")
	assert.Equal(util.GetManagedBindingLabels(modelBindingPair.Binding, "cluster1"), services.Items[0].Labels, "labels not equal to expected value")
	selector := map[string]string{"verrazzano.name": "test-generic"}
	assert.Equal(selector, services.Items[0].Spec.Selector, "selector not equal to expected value")
	assert.Equal(corev1.ServiceType("ClusterIP"), services.Items[0].Spec.Type, "service type not equal to expected value")
	assert.Equal("test-port", services.Items[0].Spec.Ports[0].Name, "port name not equal to expected value")
	assert.Equal(int32(8090), services.Items[0].Spec.Ports[0].Port, "port not equal to expected value")
	assert.Equal(intstr.FromInt(0), services.Items[0].Spec.Ports[0].TargetPort, "target port name not equal to expected value")
	assert.Equal(corev1.Protocol("TCP"), services.Items[0].Spec.Ports[0].Protocol, "protocol name not equal to expected value")
}

// TestCreateServicesVMIBinding model/binding for VMI system binding - Node Exporter service is created.
func TestCreateServicesVMIBinding(t *testing.T) {
	mbPairSystemBinding := testutil.GetModelBindingPair()
	mbPairSystemBinding.Model.Name = constants.VmiSystemBindingName
	mbPairSystemBinding.Binding.Name = constants.VmiSystemBindingName

	// Model/binding for VMI system binding - Node Exporter service is created.
	managedConnections := testutil.GetManagedClusterConnections()
	err := CreateServices(mbPairSystemBinding, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service: %v", err))
	}

	for clusterName := range mbPairSystemBinding.ManagedClusters {
		managedClusterConnection := managedConnections[clusterName]
		existingService, err := managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatal(fmt.Sprintf("can't list service for cluster %s: %v", err, clusterName))
		}
		assertCreateServiceVMI(t, existingService, clusterName)

		// Update the port value so when we call CreateServices again the update code is executed.
		existingService.Items[0].Spec.Ports[0].Port = 9200
		_, err = managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).Update(context.TODO(), &existingService.Items[0], metav1.UpdateOptions{})
		if err != nil {
			t.Fatal(fmt.Sprintf("can't update service for cluster %s: %v", err, clusterName))
		}
	}

	// Model/binding for VMI system binding - service already exist
	err = CreateServices(mbPairSystemBinding, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service: %v", err))
	}

	for clusterName := range mbPairSystemBinding.ManagedClusters {
		managedClusterConnection := managedConnections[clusterName]
		existingService, err := managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatal(fmt.Sprintf("can't list service for cluster %s: %v", err, clusterName))
		}
		assertCreateServiceVMI(t, existingService, clusterName)
	}
}

func assertCreateServiceVMI(t *testing.T, existingService *corev1.ServiceList, clusterName string) {
	labels := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}

	assert := assert.New(t)

	assert.Equal(1, len(existingService.Items), "one service should be found for cluster %s", clusterName)
	assert.Equal(constants.NodeExporterName, existingService.Items[0].Name)
	assert.Equal(constants.MonitoringNamespace, existingService.Items[0].Namespace)
	assert.Equal(monitoring.GetNodeExporterLabels(clusterName), existingService.Items[0].Labels)
	assert.Equal(labels, existingService.Items[0].Spec.Selector)
	assert.Equal(corev1.ServiceType("ClusterIP"), existingService.Items[0].Spec.Type)
	assert.Equal("metrics", existingService.Items[0].Spec.Ports[0].Name)
	assert.Equal(int32(9100), existingService.Items[0].Spec.Ports[0].Port)
	assert.Equal(intstr.FromInt(9100), existingService.Items[0].Spec.Ports[0].TargetPort)
	assert.Equal(corev1.Protocol("TCP"), existingService.Items[0].Spec.Ports[0].Protocol)
}
