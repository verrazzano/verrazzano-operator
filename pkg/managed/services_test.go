// Copyright (c) 2020, Oracle and/or its affiliates.
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestCreateServicesAppBinding model/binding is for application binding - no service is created.
func TestCreateServicesAppBinding(t *testing.T) {
	err := CreateServices(testutil.GetModelBindingPair(), testutil.GetManagedClusterConnections())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service: %v", err))
	}

	for clusterName := range testutil.GetModelBindingPair().ManagedClusters {
		managedClusterConnection := testutil.GetManagedClusterConnections()[clusterName]
		existingServices, err := managedClusterConnection.ServiceLister.Services(constants.MonitoringNamespace).List(labels.Everything())
		if err != nil {
			t.Fatal(fmt.Sprintf("can't list service for cluster %s: %v", err, clusterName))
		}
		assert.Equal(t, 0, len(existingServices), "no services should be found for cluster %s", clusterName)
	}
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
		assertCreateService(t, existingService, clusterName)

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
		assertCreateService(t, existingService, clusterName)
	}
}

func assertCreateService(t *testing.T, existingService *corev1.ServiceList, clusterName string) {
	labels := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}

	assert.Equal(t, 1, len(existingService.Items), "one service should be found for cluster %s", clusterName)
	assert.Equal(t, constants.NodeExporterName, existingService.Items[0].Name)
	assert.Equal(t, constants.MonitoringNamespace, existingService.Items[0].Namespace)
	assert.Equal(t, monitoring.GetNodeExporterLabels(clusterName), existingService.Items[0].Labels)
	assert.Equal(t, labels, existingService.Items[0].Spec.Selector)
	assert.Equal(t, corev1.ServiceType("ClusterIP"), existingService.Items[0].Spec.Type)
	assert.Equal(t, "metrics", existingService.Items[0].Spec.Ports[0].Name)
	assert.Equal(t, int32(9100), existingService.Items[0].Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(9100), existingService.Items[0].Spec.Ports[0].TargetPort)
	assert.Equal(t, corev1.Protocol("TCP"), existingService.Items[0].Spec.Ports[0].Protocol)
}
