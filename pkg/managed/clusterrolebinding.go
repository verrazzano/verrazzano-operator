// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func CreateClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {

	glog.V(6).Infof("Creating/updating ClusterRoleBindings for VerrazzanoBinding %s", mbPair.Binding.Name)

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct ClusterRoleBindings for each ManagedCluster
	for clusterName, _ := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct the set of expected ClusterRoleBindings
		newClusterRoleBindings := newClusterRoleBindings(mbPair.Binding, clusterName)

		// Create or update ClusterRoleBindings
		for _, clusterRoleBinding := range newClusterRoleBindings {
			existingClusterRoleBinding, err := managedClusterConnection.ClusterRoleBindingLister.Get(clusterRoleBinding.Name)
			if existingClusterRoleBinding != nil {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingClusterRoleBinding, clusterRoleBinding)
				if specDiffs != "" {
					glog.V(6).Infof("ClusterRoleBinding %s : Spec differences %s", clusterRoleBinding.Name, specDiffs)
					glog.V(4).Infof("Updating ClusterRoleBinding %s in cluster %s", clusterRoleBinding.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{})
				}
			} else {
				glog.V(4).Infof("Creating ClusterRoleBinding %s in cluster %s", clusterRoleBinding.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CleanupOrphanedClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error {
	glog.V(6).Infof("Cleaning up orphaned ClusterRoleBindings for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(mbPair, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Get rid of any ClusterRoleBindings with the specified binding
		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of ClusterRoleBindings for this cluster and given binding
		existingClusterRoleBindingsList, err := managedClusterConnection.ClusterRoleBindingLister.List(selector)
		if err != nil {
			return err
		}
		// Delete these ClusterRoleBindings since none are expected on this cluster
		for _, roleBinding := range existingClusterRoleBindingsList {
			glog.V(4).Infof("Deleting ClusterRoleBinding %s in cluster %s", roleBinding.Name, clusterName)
			err := managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), roleBinding.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func DeleteClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	glog.V(6).Infof("Deleting ClusterRoleBinding for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Delete ClusterRoleBindings associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range filteredConnections {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var selector labels.Selector
		if bindingLabel {
			selector = labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name})
		} else {
			selector = labels.SelectorFromSet(util.GetManagedLabelsNoBinding(clusterName))
		}
		existingClusterRoleBindingsList, err := managedClusterConnection.ClusterRoleBindingLister.List(selector)
		if err != nil {
			return err
		}
		for _, clusterRoleBinding := range existingClusterRoleBindingsList {
			if clusterRoleBinding.Name != constants.VerrazzanoSystem && clusterRoleBinding.Name != constants.VerrazzanoSystemAdmin {
				glog.V(4).Infof("Deleting ClusterRoleBinding %s", clusterRoleBinding.Name)
				err := managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), clusterRoleBinding.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Constructs the necessary Roles for the specified ManagedCluster in the given VerrazzanoBinding
func newClusterRoleBindings(binding *v1beta1v8o.VerrazzanoBinding, managedClusterName string) []*rbacv1.ClusterRoleBinding {
	roleLabels := util.GetManagedLabelsNoBinding(managedClusterName)
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding

	// Append Cluster role bindings from system monitoring
	if binding.Name == constants.VmiSystemBindingName {
		clusterRoleBindings = append(clusterRoleBindings, monitoring.GetSystemClusterRoleBindings(managedClusterName)...)
		return clusterRoleBindings
	}

	// Only generate a cluster role binding for the managed cluster namespace of the binding (which will have Verrazzano deployments)
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   util.GetServiceAccountNameForSystem(),
			Labels: roleLabels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      util.GetServiceAccountNameForSystem(),
				Namespace: util.GetManagedClusterNamespaceForSystem(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     util.GetServiceAccountNameForSystem(),
		},
	}
	clusterRoleBindings = append(clusterRoleBindings, clusterRoleBinding)

	return clusterRoleBindings
}
