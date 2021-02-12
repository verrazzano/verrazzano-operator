// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateDaemonSets creates/updates daemon sets needed for each managed cluster.
func CreateDaemonSets(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection, verrazzanoURI string, containerRuntime string) error {
	zap.S().Debugf("Creating/updating daemonset for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// If binding is not System binding, skip creating Daemon sets
	if vzSynMB.SynBinding.Name != constants.VmiSystemBindingName {
		zap.S().Debugf("Skip creating Daemon sets for VerrazzanoApplicationBinding %s", vzSynMB.SynBinding.Name)
		return nil
	}

	// Construct deployments for each ManagedCluster
	for clusterName := range vzSynMB.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct DaemonSet for each ManagedCluster
		newDaemonSets, err := newDaemonSet(clusterName, verrazzanoURI, containerRuntime)
		if err != nil {
			return err
		}

		for _, newDaemonSet := range newDaemonSets {
			existingcm, err := managedClusterConnection.DaemonSetLister.DaemonSets(newDaemonSet.Namespace).Get(newDaemonSet.Name)
			if existingcm == nil {
				zap.S().Infof("Creating DaemonSet %s in cluster %s", newDaemonSet.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.AppsV1().DaemonSets(newDaemonSet.Namespace).Create(context.TODO(), newDaemonSet, metav1.CreateOptions{})
				if err != nil {
					return err
				}
				continue
			}
			//If  daemonset already exists, check the spec differences
			specDiffs := diff.CompareIgnoreTargetEmpties(existingcm, newDaemonSet)
			if specDiffs != "" {
				zap.S().Debugf("DaemonSet %s : Spec differences %s", newDaemonSet.Name, specDiffs)
				zap.S().Infof("Updating DaemonSet %s in cluster %s", newDaemonSet.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.AppsV1().DaemonSets(newDaemonSet.Namespace).Update(context.TODO(), newDaemonSet, metav1.UpdateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Constructs the necessary Daemonset for the specified ManagedCluster
func newDaemonSet(managedClusterName string, verrazzanoURI string, containerRuntime string) ([]*appsv1.DaemonSet, error) {
	var daemonSets []*appsv1.DaemonSet
	daemonSets = monitoring.SystemDaemonSets(managedClusterName, verrazzanoURI, containerRuntime)
	return daemonSets, nil
}
