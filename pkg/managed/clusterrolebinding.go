// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"os"

	"github.com/rs/zerolog"
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

// CreateClusterRoleBindings creates/updates cluster role bindings needed for each managed cluster.
func CreateClusterRoleBindings(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating cluster role bindings
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ClusterRoleBindings").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Creating/updating ClusterRoleBindings for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Construct ClusterRoleBindings for each ManagedCluster
	for clusterName := range mbPair.ManagedClusters {
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
					logger.Debug().Msgf("ClusterRoleBinding %s : Spec differences %s", clusterRoleBinding.Name, specDiffs)
					logger.Info().Msgf("Updating ClusterRoleBinding %s in cluster %s", clusterRoleBinding.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{})
				}
			} else {
				logger.Debug().Msgf("Creating ClusterRoleBinding %s in cluster %s", clusterRoleBinding.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CleanupOrphanedClusterRoleBindings deletes cluster role bindings that have been orphaned.
func CleanupOrphanedClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating cluster role bindings
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "OrphanedClusterRoleBindings").Str("name", "Creation").Logger()

	logger.Info().Msgf("Cleaning up orphaned ClusterRoleBindings for VerrazzanoBinding %s", mbPair.Binding.Name)

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
			logger.Info().Msgf("Deleting ClusterRoleBinding %s in cluster %s", roleBinding.Name, clusterName)
			err := managedClusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), roleBinding.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteClusterRoleBindings deletes cluster role bindings for a given binding.
func DeleteClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	// Create log instance for creating cluster role bindings
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ClusterRoleBindings").Str("name", "Deletion").Logger()

	logger.Debug().Msgf("Deleting ClusterRoleBinding for VerrazzanoBinding %s", mbPair.Binding.Name)

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
				logger.Info().Msgf("Deleting ClusterRoleBinding %s", clusterRoleBinding.Name)
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
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding

	// Append Cluster role bindings from system monitoring
	if binding.Name == constants.VmiSystemBindingName {
		clusterRoleBindings = append(clusterRoleBindings, monitoring.GetSystemClusterRoleBindings(managedClusterName)...)
	} else {
		clusterRoleBindings = append(clusterRoleBindings, getManagedClusterRoleBinding(managedClusterName))
	}
	return clusterRoleBindings
}

// getClusterRoleBinding generates a cluster role binding for the managed cluster namespace
// of the binding (which will have Verrazzano deployments)
func getManagedClusterRoleBinding(managedClusterName string) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   util.GetServiceAccountNameForSystem(),
			Labels: util.GetManagedLabelsNoBinding(managedClusterName),
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
	return clusterRoleBinding
}
