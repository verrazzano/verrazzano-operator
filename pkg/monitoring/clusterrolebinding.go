// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSystemClusterRoleBindings gets all the cluster role bindings needed by Filebeats, Journalbeats, and NodeExporters
// in all the managed clusters.
func GetSystemClusterRoleBindings(managedClusterName string) []*rbacv1.ClusterRoleBinding {
	filebeatLabels := GetFilebeatLabels(managedClusterName)
	journalBeatLabels := GetJournalbeatLabels(managedClusterName)
	nodeExporterLabels := GetNodeExporterLabels(managedClusterName)
	fluentdLabels := GetFluentdLabels(managedClusterName)
	return []*rbacv1.ClusterRoleBinding{
		createSystemClusterRoleBinding(constants.LoggingNamespace, constants.FilebeatName, filebeatLabels),
		createSystemClusterRoleBinding(constants.LoggingNamespace, constants.JournalbeatName, journalBeatLabels),
		createSystemClusterRoleBinding(constants.MonitoringNamespace, constants.NodeExporterName, nodeExporterLabels),
		createSystemClusterRoleBinding(constants.VerrazzanoNamespace, constants.FluentdName, fluentdLabels),
	}
}

// Constructs the necessary cluster role binding
func createSystemClusterRoleBinding(namespace string, name string, labels map[string]string) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
	}
	return clusterRoleBinding
}
