// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

func TestGetSystemClusterRoles(t *testing.T) {
	assert := assert.New(t)
	clusterName := "test-cluster"
	clusterRoles := GetSystemClusterRoles(clusterName)
	assert.NotNil(clusterRoles, "Expected the response to not be nil")
	assert.Len(clusterRoles, 3, "Expected three ClusterRoles to be returned")

	// Validate that all the expected ClusterRoles were returned
	fileBeatFound := false
	journalBeatFound := false
	nodeExporterFound := false
	unexpectedNamesFound := false
	var unexpectedNames []string
	for _, clusterRole := range clusterRoles {
		labels := clusterRole.Labels
		switch clusterRole.Name {
		case constants.FilebeatName:
			fileBeatFound = true
			assert.Lenf(labels, 3, "Expected three labels in ClusterRole %s", clusterRole.Name)
			assert.Equalf(constants.FilebeatName, labels[constants.K8SAppLabel], "Expected label %s to be %s", constants.K8SAppLabel, constants.FilebeatName)

		case constants.JournalbeatName:
			journalBeatFound = true
			assert.Lenf(labels, 3, "Expected three labels in ClusterRole %s", clusterRole.Name)
			assert.Equalf(constants.JournalbeatName, labels[constants.K8SAppLabel], "Expected label %s to be %s", constants.K8SAppLabel, constants.JournalbeatName)

		case constants.NodeExporterName:
			nodeExporterFound = true
			assert.Lenf(labels, 3, "Expected three labels in ClusterRole %s", clusterRole.Name)
			assert.Equalf(constants.NodeExporterName, labels[constants.ServiceAppLabel], "Expected label %s to be %s", constants.ServiceAppLabel, constants.NodeExporterName)

			policyRules := clusterRole.Rules
			assert.Lenf(policyRules, 1, "Expected one policy rule created for ClusterRoles %s", clusterRole.Name)
			policyRule := policyRules[0]
			assert.Lenf(policyRule.APIGroups, 1, "Expected one APIGroup entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.APIGroups)
			assert.Contains(policyRule.APIGroups, "extensions")
			assert.Lenf(policyRule.Resources, 1, "Expected one Resources entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(policyRule.Resources, "podsecuritypolicies")
			assert.Lenf(policyRule.Verbs, 1, "Expected on Verbs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(policyRule.Verbs, "use")
			assert.Lenf(policyRule.ResourceNames, 1, "Expected one ResourceNames entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.ResourceNames)
			assert.Contains(policyRule.ResourceNames, "system-node-exporter")
			assert.Lenf(policyRule.NonResourceURLs, 0, "Expected no NonResourceURLs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.NonResourceURLs)

		default:
			unexpectedNamesFound = true
			unexpectedNames = append(unexpectedNames, clusterRole.Name)
		}

		// Common label checks
		assert.Equalf(labels[constants.VerrazzanoBinding], constants.VmiSystemBindingName, "Expected label %s to be %s for ClusterRole %s", constants.VerrazzanoBinding, constants.VmiSystemBindingName, clusterRole.Name)
		assert.Equalf(labels[constants.VerrazzanoCluster], clusterName, "Expected label %s to be %s for ClusterRole %s", constants.VerrazzanoCluster, clusterName, clusterRole.Name)

		// Policy checks that are common to FileBeat and JournalBeat
		if clusterRole.Name == constants.FilebeatName || clusterRole.Name == constants.JournalbeatName {
			policyRules := clusterRole.Rules
			assert.Lenf(policyRules, 1, "Expected one policy rule created for ClusterRoles %s", clusterRole.Name)
			policyRule := policyRules[0]
			assert.Lenf(policyRule.APIGroups, 1, "Expected one APIGroup entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.APIGroups)
			assert.Contains(policyRule.APIGroups, "")
			assert.Lenf(policyRule.Resources, 4, "Expected four Resources entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(policyRule.Resources, "pods")
			assert.Contains(policyRule.Resources, "namespaces")
			assert.Contains(policyRule.Resources, "services")
			assert.Contains(policyRule.Resources, "daemonsets")
			assert.Lenf(policyRule.Verbs, 3, "Expected three Verbs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(policyRule.Verbs, "get")
			assert.Contains(policyRule.Verbs, "watch")
			assert.Contains(policyRule.Verbs, "list")
			assert.Lenf(policyRule.ResourceNames, 0, "Expected no ResourceNames entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.ResourceNames)
			assert.Lenf(policyRule.NonResourceURLs, 0, "Expected no NonResourceURLs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.NonResourceURLs)
		}
	}
	assert.Truef(fileBeatFound, "Expected to get a ClusterRole response for %s", constants.FilebeatName)
	assert.Truef(journalBeatFound, "Expected to get a ClusterRole response for %s", constants.JournalbeatName)
	assert.Truef(nodeExporterFound, "Expected to get a ClusterRole response for %s", constants.NodeExporterName)
	assert.Falsef(unexpectedNamesFound, "Unexpected ClusterRoles returned: %q", unexpectedNames)
}
