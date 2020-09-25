// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/golang/glog"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSystemClusterRoles gets all the cluster roles needed by Filebeats, Journalbeats, and NodeExporters in
// all the clusters.
func GetSystemClusterRoles(managedClusterName string) []*rbacv1.ClusterRole {
	var clusterRoles []*rbacv1.ClusterRole
	filebeatLabels := GetFilebeatLabels(managedClusterName)
	journalbeatLabels := GetJournalbeatLabels(managedClusterName)
	nodeExporterLabels := GetNodeExporterLabels(managedClusterName)

	fileabeatCR, err := createLoggingClusterRoles(constants.FilebeatName, filebeatLabels)
	if err != nil {
		glog.V(6).Infof("New logging cluster role %s is giving error %s", constants.FilebeatName, err)
	}
	journalbeatCR, err := createLoggingClusterRoles(constants.JournalbeatName, journalbeatLabels)
	if err != nil {
		glog.V(6).Infof("New logging cluster role %s is giving error %s", constants.JournalbeatName, err)
	}
	nodeExporterCR, err := createMonitoringClusterRoles(constants.NodeExporterName, nodeExporterLabels)
	if err != nil {
		glog.V(6).Infof("New monitoring cluster role %s is giving error %s", constants.NodeExporterName, err)
	}
	clusterRoles = append(clusterRoles, fileabeatCR, journalbeatCR, nodeExporterCR)
	return clusterRoles
}

// Constructs the necessary cluster role
func createLoggingClusterRoles(name string, labels map[string]string) (*rbacv1.ClusterRole, error) {
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
	return clusterRole, nil
}

// Constructs the necessary ClusterRoles for the specified ManagedCluster
func createMonitoringClusterRoles(name string, labels map[string]string) (*rbacv1.ClusterRole, error) {
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
	return clusterRole, nil
}
