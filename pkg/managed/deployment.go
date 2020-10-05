// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding

package managed

import (
	"context"
	"errors"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/cohoperator"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/helidonapp"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	"github.com/verrazzano/verrazzano-operator/pkg/wlsopr"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CreateDeployments creates/updates deployments needed for each managed cluster.
func CreateDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, manifest *util.Manifest, verrazzanoURI string, sec monitoring.Secrets) error {
	glog.V(6).Infof("Creating/updating Deployments for VerrazzanoBinding %s", mbPair.Binding.Name)

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct deployments for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var deployments []*appsv1.Deployment
		if mbPair.Binding.Name == constants.VmiSystemBindingName {
			deployments, err = monitoring.GetSystemDeployments(clusterName, verrazzanoURI, util.GetManagedLabelsNoBinding(clusterName), sec)
			if err != nil {
				glog.Errorf("Error getting the monitoring system deployments %v", err)
				continue
			}
		} else {
			deployments, err = newSystemDeployments(mbPair.Binding, managedClusterObj, manifest, verrazzanoURI, sec)
			if err != nil {
				glog.Errorf("Error creating new deployments %v", err)
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
					glog.V(6).Infof("Deployment %s : Spec differences %s", newDeployment.Name, specDiffs)
					glog.V(4).Infof("Updating deployment %s:%s in cluster %s", newDeployment.Namespace, newDeployment.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.AppsV1().Deployments(newDeployment.Namespace).Update(context.TODO(), newDeployment, metav1.UpdateOptions{})
				}
			} else {
				glog.V(4).Infof("Creating deployment %s:%s in cluster %s", newDeployment.Namespace, newDeployment.Name, clusterName)
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
func DeleteDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	glog.V(6).Infof("Deleting Deployments for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Delete Deployments associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range filteredConnections {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(util.GetManagedBindingLabels(mbPair.Binding, clusterName))

		existingDeploymentList, err := managedClusterConnection.DeploymentLister.List(selector)
		if err != nil {
			return err
		}
		for _, deployment := range existingDeploymentList {
			glog.V(4).Infof("Deleting Deployment %s:%s", deployment.Namespace, deployment.Name)
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
func CleanupOrphanedDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	glog.V(6).Infof("Cleaning up orphaned Deployments for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Get the managed clusters that this binding applies to
	matchedClusters, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	for clusterName, mc := range mbPair.ManagedClusters {
		managedClusterConnection := matchedClusters[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

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
				glog.V(4).Infof("Deleting Deployment %s:%s in cluster %s", deployment.Namespace, deployment.Name, clusterName)
				err := managedClusterConnection.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(mbPair, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Get rid of any Deployments with the specified binding
		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of Deployments for this cluster and given binding
		existingDeploymentList, err := managedClusterConnection.DeploymentLister.List(selector)
		if err != nil {
			return err
		}

		// Delete these Deployments since they are no longer needed on this cluster.
		for _, deployment := range existingDeploymentList {
			glog.V(4).Infof("Deleting Deployment %s:%s in cluster %s", deployment.Namespace, deployment.Name, clusterName)
			err := managedClusterConnection.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Constructs the necessary Verrazzano system deployments for the specified ManagedCluster in the given VerrazzanoBinding
func newSystemDeployments(binding *v1beta1v8o.VerrazzanoBinding, managedCluster *types.ManagedCluster, manifest *util.Manifest, verrazzanoURI string, sec monitoring.Secrets) ([]*appsv1.Deployment, error) {
	managedNamespace := util.GetManagedClusterNamespaceForSystem()
	deployPromPusher := true //temporary variable to create pusher deployment
	depLabels := util.GetManagedLabelsNoBinding(managedCluster.Name)
	var deployments []*appsv1.Deployment

	// Does a WebLogic application need to be deployed to this cluster?  If so, deploy the micro-operator that will manage it.
	if managedCluster.WlsOperator != nil {
		deployment := wlsopr.CreateDeployment(managedNamespace, binding.Name, depLabels, manifest.WlsMicroOperatorImage)
		deployments = append(deployments, deployment)
	}

	// Does a Coherence applications need to be deployed to this cluster?  If so, deploy the micro-operator that will manage it.
	if managedCluster.CohOperatorCRs != nil {
		deployment := cohoperator.CreateDeployment(managedNamespace, binding.Name, depLabels, manifest.CohClusterOperatorImage)
		deployments = append(deployments, deployment)
	}

	// Does a Helidon Application need to be deployed to this cluster?  If so, deploy the micro-operator that will manage it.
	if managedCluster.HelidonApps != nil && manifest.HelidonAppOperatorImage != "" {
		deployment := helidonapp.CreateAppOperatorDeployment(managedNamespace, depLabels,
			manifest.HelidonAppOperatorImage, util.GetHelidonMicroRequestMemory())
		deployments = append(deployments, deployment)
	}

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
