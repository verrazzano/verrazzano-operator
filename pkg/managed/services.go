// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"

	"github.com/golang/glog"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateServices(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {

	glog.V(6).Infof("Creating/updating Service for VerrazzanoBinding %s", mbPair.Binding.Name)

	// If binding is not System binding, skip creating Services
	if mbPair.Binding.Name != constants.VmiSystemBindingName {
		glog.V(6).Infof("Skip creating Services for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)
		return nil
	}

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct services for each ManagedCluster
	for clusterName, _ := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct the set of expected Services
		newServices, err := newServices(clusterName)

		if err != nil {
			return err
		}

		// Create or update Services
		for _, newService := range newServices {
			existingService, err := managedClusterConnection.ServiceLister.Services(newService.Namespace).Get(newService.Name)
			if existingService != nil {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingService, newService)
				if specDiffs != "" {
					glog.V(6).Infof("Service %s : Spec differences %s", newService.Name, specDiffs)
					glog.V(4).Infof("Updating Service %s in cluster %s", newService.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.CoreV1().Services(newService.Namespace).Update(context.TODO(), newService, metav1.UpdateOptions{})
				}
			} else {
				glog.V(4).Infof("Creating Service %s in cluster %s", newService.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.CoreV1().Services(newService.Namespace).Create(context.TODO(), newService, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Constructs the necessary Service for the specified ManagedCluster
func newServices(managedClusterName string) ([]*corev1.Service, error) {
	roleLabels := monitoring.GetNodeExporterLabels(managedClusterName)
	labels := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}
	var Services []*corev1.Service

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.NodeExporterName,
			Labels:    roleLabels,
			Namespace: constants.MonitoringNamespace,
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
			Selector: labels,
			Type:     "ClusterIP",
		},
	}
	Services = append(Services, service)

	return Services, nil
}
