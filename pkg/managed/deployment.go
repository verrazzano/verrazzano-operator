// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding

package managed

import (
	"context"

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
			deployments = monitoring.GetSystemDeployments(clusterName, verrazzanoURI, sec)
		} else {
			deployments = newDeployments(mbPair.Binding, managedClusterObj, manifest, verrazzanoURI, sec)
		}

		// Create/Update Deployments
		var deploymentNames []string
		for _, newDeployment := range deployments {
			deploymentNames = append(deploymentNames, newDeployment.Name)
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

// Constructs the necessary Deployments for the specified ManagedCluster in the given VerrazzanoBinding
func newDeployments(binding *v1beta1v8o.VerrazzanoBinding, managedCluster *types.ManagedCluster, manifest *util.Manifest, verrazzanoURI string, sec monitoring.Secrets) []*appsv1.Deployment {
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
		deployment := helidonapp.CreateDeployment(managedNamespace, depLabels, manifest.HelidonAppOperatorImage)
		deployments = append(deployments, deployment)
	}

	// Does a Prometheus pusher need to be deployed to this cluster?
	if deployPromPusher == true {
		if verrazzanoURI == "" {
			glog.V(4).Infof("Verrazzano URI must not be empty for prometheus pusher to work")
		} else {
			deployment := monitoring.CreateDeployment(constants.MonitoringNamespace, binding.Name, depLabels, sec)
			deployments = append(deployments, deployment)
		}
	}

	return deployments
}
