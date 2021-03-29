// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSystemClusterRoles gets all the cluster roles needed by Filebeats, Journalbeats, and NodeExporters in
// all the clusters.
func GetSystemClusterRoles(managedClusterName string) []*rbacv1.ClusterRole {
	return []*rbacv1.ClusterRole{
		createLoggingClusterRoles(constants.FilebeatName, GetFilebeatLabels(managedClusterName)),
		createLoggingClusterRoles(constants.JournalbeatName, GetJournalbeatLabels(managedClusterName)),
		createMonitoringClusterRoles(constants.NodeExporterName, GetNodeExporterLabels(managedClusterName)),
		createLoggingClusterRoles(constants.FluentdName, GetFluentdLabels(managedClusterName)),
	}
}

// Constructs the necessary cluster role
func createLoggingClusterRoles(name string, labels map[string]string) *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "namespaces", "services", "daemonsets"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	return clusterRole
}

// Constructs the necessary ClusterRoles for the specified ManagedCluster
func createMonitoringClusterRoles(name string, labels map[string]string) *rbacv1.ClusterRole {
	// Only generate a cluster role for the managed cluster namespace
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"extensions"},
				Resources:     []string{"podsecuritypolicies"},
				Verbs:         []string{"use"},
				ResourceNames: []string{"system-node-exporter"},
			},
		},
	}
	return clusterRole
}
