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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDaemonSets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, verrazzanoUri string) error {

	glog.V(6).Infof("Creating/updating daemonset for VerrazzanoBinding %s", mbPair.Binding.Name)

	// If binding is not System binding, skip creating Daemon sets
	if mbPair.Binding.Name != constants.VmiSystemBindingName {
		glog.V(6).Infof("Skip creating Daemon sets for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)
		return nil
	}

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct deployments for each ManagedCluster
	for clusterName := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct DaemonSet for each ManagedCluster
		newDaemonSets, err := newDaemonSet(mbPair.Binding.Name, clusterName, verrazzanoUri)
		if err != nil {
			return err
		}

		for _, newDaemonSet := range newDaemonSets {
			existingcm, err := managedClusterConnection.DaemonSetLister.DaemonSets(newDaemonSet.Namespace).Get(newDaemonSet.Name)
			if existingcm == nil {
				glog.V(4).Infof("Creating DaemonSet %s in cluster %s", newDaemonSet.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.AppsV1().DaemonSets(newDaemonSet.Namespace).Create(context.TODO(), newDaemonSet, metav1.CreateOptions{})
				if err != nil {
					return err
				}
				continue
			}
			//If  daemonset already exists, check the spec differences
			specDiffs := diff.CompareIgnoreTargetEmpties(existingcm, newDaemonSet)
			if specDiffs != "" {
				glog.V(6).Infof("DaemonSet %s : Spec differences %s", newDaemonSet.Name, specDiffs)
				glog.V(4).Infof("Updating DaemonSet %s in cluster %s", newDaemonSet.Name, clusterName)
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
func newDaemonSet(bindingName string, managedClusterName string, verrazzanoUri string) ([]*appsv1.DaemonSet, error) {
	var daemonSets []*appsv1.DaemonSet
	daemonSets = monitoring.SystemDaemonSets(managedClusterName, verrazzanoUri)
	return daemonSets, nil
}
