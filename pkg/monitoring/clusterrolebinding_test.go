// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

func TestGetSystemClusterRoleBindings(t *testing.T) {
	assert := assert.New(t)
	clusterName := "test-cluster"
	clusterRoleBindings := GetSystemClusterRoleBindings(clusterName)
	assert.NotNil(clusterRoleBindings, "Expected the response to not be nil")
	assert.Len(clusterRoleBindings, 3, "Expected three ClusterRoleBindings to be returned")

	// Validate that all the expected ClusterRoleBindings are returned
	fileBeatFound := false
	journalBeatFound := false
	nodeExporterFound := false
	unexpectedNamesFound := false
	var unexpectedNames []string
	for _, clusterRoleBinding := range clusterRoleBindings {
		labels := clusterRoleBinding.Labels

		// Common Subjects checks
		subjects := clusterRoleBinding.Subjects
		assert.Lenf(subjects, 1, "Expected one Subject rule created for ClusterRoleBinding %s", clusterRoleBinding.Name)
		subject := subjects[0]
		assert.Equalf("ServiceAccount", subject.Kind, "ClusterRoleBinding %s contains unexpected Kind for Subject", clusterRoleBinding.Name)

		// Common RoleRef checks
		roleRef := clusterRoleBinding.RoleRef
		assert.Equalf("rbac.authorization.k8s.io", roleRef.APIGroup, "ClusterRoleBinding %s contains unexpected APIGroup for RoleRef", clusterRoleBinding.Name)
		assert.Equalf("ClusterRole", roleRef.Kind, "ClusterRoleBinding %s contains unexpected Kind for RoleRef", clusterRoleBinding.Name)

		switch clusterRoleBinding.Name {
		case constants.FilebeatName:
			fileBeatFound = true
			assert.Lenf(labels, 3, "Expected three labels in ClusterRoleBinding %s", clusterRoleBinding.Name)
			assert.Equalf(constants.FilebeatName, labels[constants.K8SAppLabel], "Expected label %s to be %s", constants.K8SAppLabel, constants.FilebeatName)
			assert.Equalf(constants.FilebeatName, subject.Name, "ClusterRoleBinding %s contains unexpected Name for Subject", clusterRoleBinding.Name)
			assert.Equalf(constants.LoggingNamespace, subject.Namespace, "ClusterRoleBinding %s contains unexpected Namespace for Subject", clusterRoleBinding.Name)
			assert.Equalf(constants.FilebeatName, roleRef.Name, "ClusterRoleBinding %s contains unexpected Name for RoleRef", clusterRoleBinding.Name)

		case constants.JournalbeatName:
			journalBeatFound = true
			assert.Lenf(labels, 3, "Expected three labels in ClusterRoleBinding %s", clusterRoleBinding.Name)
			assert.Equalf(constants.JournalbeatName, labels[constants.K8SAppLabel], "Expected label %s to be %s", constants.K8SAppLabel, constants.JournalbeatName)
			assert.Equalf(constants.JournalbeatName, subject.Name, "ClusterRoleBinding %s contains unexpected Name for Subject", clusterRoleBinding.Name)
			assert.Equalf(constants.LoggingNamespace, subject.Namespace, "ClusterRoleBinding %s contains unexpected Namespace for Subject", clusterRoleBinding.Name)
			assert.Equalf(constants.JournalbeatName, roleRef.Name, "ClusterRoleBinding %s contains unexpected Name for RoleRef", clusterRoleBinding.Name)

		case constants.NodeExporterName:
			nodeExporterFound = true
			assert.Lenf(labels, 3, "Expected three labels in ClusterRoleBinding %s", clusterRoleBinding.Name)
			assert.Equalf(constants.NodeExporterName, labels[constants.ServiceAppLabel], "Expected label %s to be %s", constants.ServiceAppLabel, constants.NodeExporterName)
			assert.Equalf(constants.NodeExporterName, subject.Name, "ClusterRoleBinding %s contains unexpected Name for Subject", clusterRoleBinding.Name)
			assert.Equalf(constants.MonitoringNamespace, subject.Namespace, "ClusterRoleBinding %s contains unexpected Namespace for Subject", clusterRoleBinding.Name)
			assert.Equalf(constants.NodeExporterName, roleRef.Name, "ClusterRoleBinding %s contains unexpected Name for RoleRef", clusterRoleBinding.Name)

		default:
			unexpectedNamesFound = true
			unexpectedNames = append(unexpectedNames, clusterRoleBinding.Name)
		}

		// Common label checks
		assert.Equalf(labels[constants.VerrazzanoBinding], constants.VmiSystemBindingName, "Expected label %s to be %s for ClusterRoleBinding %s", constants.VerrazzanoBinding, constants.VmiSystemBindingName, clusterRoleBinding.Name)
		assert.Equalf(labels[constants.VerrazzanoCluster], clusterName, "Expected label %s to be %s for ClusterRoleBinding %s", constants.VerrazzanoCluster, clusterName, clusterRoleBinding.Name)
	}
	assert.Truef(fileBeatFound, "Expected to get a ClusterRoleBinding response for %s", constants.FilebeatName)
	assert.Truef(journalBeatFound, "Expected to get a ClusterRoleBinding response for %s", constants.JournalbeatName)
	assert.Truef(nodeExporterFound, "Expected to get a ClusterRoleBinding response for %s", constants.NodeExporterName)
	assert.Falsef(unexpectedNamesFound, "Unexpected ClusterRoleBinding returned: %q", unexpectedNames)
}
