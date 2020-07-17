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

func CreateClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating cluster roles
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ClusterRoles").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Creating/updating ClusterRoles for VerrazzanoBinding %s", mbPair.Binding.Name)

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct ClusterRoles for each ManagedCluster
	for clusterName, _ := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct the set of expected ClusterRoles
		newClusterRoles := newClusterRoles(mbPair.Binding, clusterName)

		// Create or update ClusterRoles
		for _, newClusterRole := range newClusterRoles {
			existingClusterRole, err := managedClusterConnection.ClusterRoleLister.Get(newClusterRole.Name)
			if existingClusterRole != nil {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingClusterRole, newClusterRole)
				if specDiffs != "" {
					logger.Debug().Msgf("ClusterRole %s : Spec differences %s", newClusterRole.Name, specDiffs)
					logger.Info().Msgf("Updating ClusterRole %s in cluster %s", newClusterRole.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.RbacV1().ClusterRoles().Update(context.TODO(), newClusterRole, metav1.UpdateOptions{})
				}
			} else {
				logger.Info().Msgf("Creating ClusterRole %s in cluster %s", newClusterRole.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), newClusterRole, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CleanupOrphanedClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error {
	// Create log instance for creating orphaned cluster roles
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "OrphanedClusterRoles").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Cleaning up orphaned ClusterRoles for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(mbPair, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Get rid of any ClusterRoles with the specified binding
		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of ClusterRoles for this cluster and given binding
		existingClusterRolesList, err := managedClusterConnection.ClusterRoleLister.List(selector)
		if err != nil {
			return err
		}
		// Delete these ClusterRoles since none are expected on this cluster
		for _, role := range existingClusterRolesList {
			logger.Info().Msgf("Deleting ClusterRole %s in cluster %s", role.Name, clusterName)
			err := managedClusterConnection.KubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), role.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func DeleteClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	// Create log instance for deleting cluster roles
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ClusterRoles").Str("name", "Deletion").Logger()

	logger.Debug().Msgf("Deleting ClusterRole for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Delete ClusterRoles associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range filteredConnections {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var selector labels.Selector
		if bindingLabel {
			selector = labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name})
		} else {
			selector = labels.SelectorFromSet(util.GetManagedLabelsNoBinding(clusterName))
		}
		existingClusterRolesList, err := managedClusterConnection.ClusterRoleLister.List(selector)
		if err != nil {
			return err
		}
		for _, clusterRole := range existingClusterRolesList {
			if clusterRole.Name != constants.VerrazzanoSystem && clusterRole.Name != constants.VerrazzanoSystemAdmin {
				logger.Info().Msgf("Deleting ClusterRole %s", clusterRole.Name)
				err := managedClusterConnection.KubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRole.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Constructs the necessary ClusterRoles for the specified ManagedCluster in the given VerrazzanoBinding
func newClusterRoles(binding *v1beta1v8o.VerrazzanoBinding, clusterName string) []*rbacv1.ClusterRole {
	roleLabels := util.GetManagedLabelsNoBinding(clusterName)
	var clusterRoles []*rbacv1.ClusterRole

	// Append Cluster roles from system monitoring
	if binding.Name == constants.VmiSystemBindingName {
		clusterRoles = append(clusterRoles, monitoring.GetSystemClusterRoles(clusterName)...)
		return clusterRoles
	}

	// Only generate a cluster role for the managed cluster namespace of the binding (which will have Verrazzano deployments)
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   util.GetServiceAccountNameForSystem(),
			Labels: roleLabels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods/log", "serviceaccounts", "pods/portforward", "nodes"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "deletecollection", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/exec", "configmaps", "endpoints", "events", "namespaces", "persistentvolumeclaims", "secrets", "services"},
				Verbs:     []string{"*"},
			},
			{
				NonResourceURLs: []string{"/version/*"},
				Verbs:           []string{"get"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "daemonsets", "replicasets", "statefulsets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups:     []string{"apps"},
				Resources:     []string{"deployments/finalizers"},
				ResourceNames: []string{"coherence-operator"},
				Verbs:         []string{"update"},
			},
			{
				APIGroups: []string{"extensions"},
				Resources: []string{"daemonsets", "replicasets", "statefulsets", "podsecuritypolicies"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{"policy"},
				Resources: []string{"podsecuritypolicies"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "patch", "deletecollection"},
			},
			{
				APIGroups: []string{"authentication.k8s.io"},
				Resources: []string{"tokenreviews"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"monitoring.coreos.com"},
				Resources: []string{"servicemonitors"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{"verrazzano.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterroles", "clusterrolebindings", "roles", "rolebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{"weblogic.oracle"},
				Resources: []string{"domains"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "patch", "deletecollection"},
			},
			{
				APIGroups: []string{"weblogic.oracle"},
				Resources: []string{"domains/status"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{"coherence.oracle.com"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
	clusterRoles = append(clusterRoles, clusterRole)

	return clusterRoles
}
