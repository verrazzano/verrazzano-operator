// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/golang/glog"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create all the cluster role bindings needed by Filebeats, Journalbeats, and NodeExporters in all the managed clusters
func GetSystemClusterRoleBindings(managedClusterName string) []*rbacv1.ClusterRoleBinding {
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding
	filebeatLabels := GetFilebeatLabels(managedClusterName)
	journalbeatLabels := GetJournalbeatLabels(managedClusterName)
	nodeExporterLabels := GetNodeExporterLabels(managedClusterName)

	fileabeatCRD, err := createSystemClusterRoleBinding(constants.LoggingNamespace, constants.FilebeatName, filebeatLabels)
	if err != nil {
		glog.V(6).Infof("New cluster role binding %s is giving error %s", constants.FilebeatName, err)
	}
	journalbeatCRD, err := createSystemClusterRoleBinding(constants.LoggingNamespace, constants.JournalbeatName, journalbeatLabels)
	if err != nil {
		glog.V(6).Infof("New cluster role binding %s is giving error %s", constants.JournalbeatName, err)
	}
	nodeExporterCRD, err := createSystemClusterRoleBinding(constants.MonitoringNamespace, constants.NodeExporterName, nodeExporterLabels)
	if err != nil {
		glog.V(6).Infof("New cluster role binding %s is giving error %s", constants.NodeExporterName, err)
	}
	clusterRoleBindings = append(clusterRoleBindings, fileabeatCRD, journalbeatCRD, nodeExporterCRD)
	return clusterRoleBindings
}

// Constructs the necessary cluster role binding
func createSystemClusterRoleBinding(namespace string, name string, labels map[string]string) (*rbacv1.ClusterRoleBinding, error) {
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
	return clusterRoleBinding, nil
}
