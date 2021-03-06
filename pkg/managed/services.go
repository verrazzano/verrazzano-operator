// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/verrazzano/pkg/diff"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateServices creates/updates services needed for each managed cluster.
func CreateServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	zap.S().Debugf("Creating/updating Service for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// Construct services for each ManagedCluster
	for clusterName, mc := range vzSynMB.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var services []*corev1.Service
		// Construct the set of expected Services
		if vzSynMB.SynBinding.Name == constants.VmiSystemBindingName {
			services = newServices(clusterName)
		} else {
			// Add services from genericComponents
			for _, service := range mc.Services {
				services = append(services, service)
			}
		}

		// Create or update Services
		for _, service := range services {
			existingService, err := managedClusterConnection.ServiceLister.Services(service.Namespace).Get(service.Name)
			if existingService != nil {
				specDiffs := diff.Diff(existingService, service)
				if specDiffs != "" {
					zap.S().Debugf("Service %s : Spec differences %s", service.Name, specDiffs)
					zap.S().Infof("Updating Service %s in cluster %s", service.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.CoreV1().Services(service.Namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
				}
			} else {
				zap.S().Infof("Creating Service %s:%s in cluster %s", service.Namespace, service.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.CoreV1().Services(service.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteServices deletes services for a given binding.
func DeleteServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	zap.S().Infof("Deleting Services for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// Delete Services associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range filteredConnections {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(util.GetManagedBindingLabels(vzSynMB.SynBinding, clusterName))

		existingServiceList, err := managedClusterConnection.ServiceLister.List(selector)
		if err != nil {
			return err
		}
		for _, service := range existingServiceList {
			zap.S().Infof("Deleting Service %s:%s", service.Namespace, service.Name)
			err := managedClusterConnection.KubeClient.CoreV1().Services(service.Namespace).Delete(context.TODO(), service.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CleanupOrphanedServices deletes services that have been orphaned.  Services can be orphaned when a binding
// has been changed to not require a service or the service was moved to a different cluster.
func CleanupOrphanedServices(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	zap.S().Infof("Cleaning up orphaned Services for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// Get the managed clusters that this binding applies to
	matchedClusters, err := util.GetManagedClustersForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)
	if err != nil {
		return nil
	}
	for clusterName, mc := range vzSynMB.ManagedClusters {
		managedClusterConnection := matchedClusters[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.SynBinding.Name, constants.VerrazzanoCluster: clusterName})

		// Get the set of expected Service names
		var serviceNames []string
		for _, service := range mc.Services {
			serviceNames = append(serviceNames, service.Name)
		}

		// Get list of Services that exist for this cluster and given binding
		existingServiceList, err := managedClusterConnection.ServiceLister.List(selector)
		if err != nil {
			return err
		}

		// Delete any Services not expected on this cluster
		for _, service := range existingServiceList {
			if !util.Contains(serviceNames, service.Name) {
				zap.S().Infof("Deleting Service %s:%s in cluster %s", service.Namespace, service.Name, clusterName)
				err := managedClusterConnection.KubeClient.CoreV1().Services(service.Namespace).Delete(context.TODO(), service.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Get rid of any Services with the specified binding
		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.SynBinding.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of Services for this cluster and given binding
		existingServiceList, err := managedClusterConnection.ServiceLister.List(selector)
		if err != nil {
			return err
		}

		// Delete these Services since they are no longer needed on this cluster.
		for _, service := range existingServiceList {
			zap.S().Infof("Deleting Service %s:%s in cluster %s", service.Namespace, service.Name, clusterName)
			err := managedClusterConnection.KubeClient.CoreV1().Services(service.Namespace).Delete(context.TODO(), service.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Constructs the necessary Service for the specified ManagedCluster
func newServices(managedClusterName string) []*corev1.Service {
	roleLabels := monitoring.GetNodeExporterLabels(managedClusterName)
	labels := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}
	var services []*corev1.Service

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
	services = append(services, service)

	return services
}
