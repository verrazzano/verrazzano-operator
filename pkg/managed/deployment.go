// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding

package managed

import (
	"context"
	"errors"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CreateDeployments creates/updates deployments needed for each managed cluster.
func CreateDeployments(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection, verrazzanoURI string, sec monitoring.Secrets) error {
	zap.S().Infof("Creating/updating Deployments for VerrazzanoBinding %s", vzSynMB.Location.Name)

	// Construct deployments for each ManagedCluster
	for clusterName, managedClusterObj := range vzSynMB.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var deployments []*appsv1.Deployment
		var err error
		if vzSynMB.Location.Name == constants.VmiSystemBindingName {
			deployments, err = monitoring.GetSystemDeployments(clusterName, verrazzanoURI, util.GetManagedLabelsNoBinding(clusterName), sec)
			if err != nil {
				zap.S().Errorf("Error getting the monitoring system deployments %v", err)
				continue
			}
		} else {
			deployments, err = newSystemDeployments(vzSynMB.Location, managedClusterObj, verrazzanoURI, sec)
			if err != nil {
				zap.S().Errorf("Error creating new deployments %v", err)
				continue
			}
			// Add deployments from genericComponents
			for _, deployment := range managedClusterObj.Deployments {
				deployments = append(deployments, deployment)
			}
		}

		// Create/Update Deployments
		for _, newDeployment := range deployments {
			existingDeployment, err := managedClusterConnection.DeploymentLister.Deployments(newDeployment.Namespace).Get(newDeployment.Name)
			if existingDeployment != nil {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingDeployment, newDeployment)
				if specDiffs != "" {
					zap.S().Debugf("Deployment %s : Spec differences %s", newDeployment.Name, specDiffs)
					zap.S().Infof("Updating deployment %s:%s in cluster %s", newDeployment.Namespace, newDeployment.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.AppsV1().Deployments(newDeployment.Namespace).Update(context.TODO(), newDeployment, metav1.UpdateOptions{})
				}
			} else {
				zap.S().Infof("Creating deployment %s:%s in cluster %s", newDeployment.Namespace, newDeployment.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.AppsV1().Deployments(newDeployment.Namespace).Create(context.TODO(), newDeployment, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteDeployments deletes deployments for a given binding.
func DeleteDeployments(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	zap.S().Infof("Deleting Deployments for VerrazzanoBinding %s", vzSynMB.Location.Name)

	// Delete Deployments associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range filteredConnections {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(util.GetManagedBindingLabels(vzSynMB.Location, clusterName))

		existingDeploymentList, err := managedClusterConnection.DeploymentLister.List(selector)
		if err != nil {
			return err
		}
		for _, deployment := range existingDeploymentList {
			zap.S().Infof("Deleting Deployment %s:%s", deployment.Namespace, deployment.Name)
			err := managedClusterConnection.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CleanupOrphanedDeployments deletes deployments that have been orphaned.   Deployments can be orphaned when a binding
// has been changed to not require a deployment or the deployment was moved to a different cluster.
func CleanupOrphanedDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	zap.S().Infof("Cleaning up orphaned Deployments for VerrazzanoBinding %s", vzSynMB.Location.Name)

	// Get the managed clusters that this binding applies to
	matchedClusters, err := util.GetManagedClustersForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	for clusterName, mc := range vzSynMB.ManagedClusters {
		managedClusterConnection := matchedClusters[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.Location.Name, constants.VerrazzanoCluster: clusterName})

		// Get the set of expected Deployment names
		var deploymentNames []string
		for _, deployment := range mc.Deployments {
			deploymentNames = append(deploymentNames, deployment.Name)
		}

		// Get list of Deployments that exist for this cluster and given binding
		existingDeploymentList, err := managedClusterConnection.DeploymentLister.List(selector)
		if err != nil {
			return err
		}

		// Delete any Deployments not expected on this cluster
		for _, deployment := range existingDeploymentList {
			if !util.Contains(deploymentNames, deployment.Name) {
				zap.S().Infof("Deleting Deployment %s:%s in cluster %s", deployment.Namespace, deployment.Name, clusterName)
				err := managedClusterConnection.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})
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

		// Get rid of any Deployments with the specified binding
		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.Location.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of Deployments for this cluster and given binding
		existingDeploymentList, err := managedClusterConnection.DeploymentLister.List(selector)
		if err != nil {
			return err
		}

		// Delete these Deployments since they are no longer needed on this cluster.
		for _, deployment := range existingDeploymentList {
			zap.S().Infof("Deleting Deployment %s:%s in cluster %s", deployment.Namespace, deployment.Name, clusterName)
			err := managedClusterConnection.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Constructs the necessary Verrazzano system deployments for the specified ManagedCluster in the given VerrazzanoBinding
func newSystemDeployments(binding *types.ResourceLocation, managedCluster *types.ManagedCluster, verrazzanoURI string, sec monitoring.Secrets) ([]*appsv1.Deployment, error) {
	deployPromPusher := true //temporary variable to create pusher deployment
	depLabels := util.GetManagedLabelsNoBinding(managedCluster.Name)
	var deployments []*appsv1.Deployment

	// Does a Prometheus pusher need to be deployed to this cluster?
	if deployPromPusher == true {
		if verrazzanoURI == "" {
			return nil, errors.New("Verrazzano URI cannot be empty for prometheus pusher deployment")
		}
		deployment, err := monitoring.CreateDeployment(constants.MonitoringNamespace, binding.Name, depLabels, sec)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}
