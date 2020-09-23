// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

func TestGetSystemClusterRoles(t *testing.T) {
	clusterName := "test-cluster"
	clusterRoles := GetSystemClusterRoles(clusterName)
	assert.NotNil(t, clusterRoles, "Expected the response to not be nil")
	assert.Len(t, clusterRoles, 3, "Expected three ClusterRoles to be returned")

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
			assert.Lenf(t, labels, 3, "Expected three labels in ClusterRole %s", clusterRole.Name)
			assert.Equalf(t, constants.FilebeatName, labels[constants.K8SAppLabel], "Expected label %s to be %s", constants.K8SAppLabel, constants.FilebeatName)

		case constants.JournalbeatName:
			journalBeatFound = true
			assert.Lenf(t, labels, 3, "Expected three labels in ClusterRole %s", clusterRole.Name)
			assert.Equalf(t, constants.JournalbeatName, labels[constants.K8SAppLabel], "Expected label %s to be %s", constants.K8SAppLabel, constants.JournalbeatName)

		case constants.NodeExporterName:
			nodeExporterFound = true
			assert.Lenf(t, labels, 3, "Expected three labels in ClusterRole %s", clusterRole.Name)
			assert.Equalf(t, constants.NodeExporterName, labels[constants.ServiceAppLabel], "Expected label %s to be %s", constants.ServiceAppLabel, constants.NodeExporterName)

			policyRules := clusterRole.Rules
			assert.Lenf(t, policyRules, 1, "Expected one policy rule created for ClusterRoles %s", clusterRole.Name)
			policyRule := policyRules[0]
			assert.Lenf(t, policyRule.APIGroups, 1, "Expected one APIGroup entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.APIGroups)
			assert.Contains(t, policyRule.APIGroups, "extensions")
			assert.Lenf(t, policyRule.Resources, 1, "Expected one Resources entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(t, policyRule.Resources, "podsecuritypolicies")
			assert.Lenf(t, policyRule.Verbs, 1, "Expected on Verbs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(t, policyRule.Verbs, "use")
			assert.Lenf(t, policyRule.ResourceNames, 1, "Expected one ResourceNames entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.ResourceNames)
			assert.Contains(t, policyRule.ResourceNames, "system-node-exporter")
			assert.Lenf(t, policyRule.NonResourceURLs, 0, "Expected no NonResourceURLs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.NonResourceURLs)

		default:
			unexpectedNamesFound = true
			unexpectedNames = append(unexpectedNames, clusterRole.Name)
		}

		// Common label checks
		assert.Equalf(t, labels[constants.VerrazzanoBinding], constants.VmiSystemBindingName, "Expected label %s to be %s for ClusterRole %s", constants.VerrazzanoBinding, constants.VmiSystemBindingName, clusterRole.Name)
		assert.Equalf(t, labels[constants.VerrazzanoCluster], clusterName, "Expected label %s to be %s for ClusterRole %s", constants.VerrazzanoCluster, clusterName, clusterRole.Name)

		// Policy checks that are common to FileBeat and JournalBeat
		if clusterRole.Name == constants.FilebeatName || clusterRole.Name == constants.JournalbeatName {
			policyRules := clusterRole.Rules
			assert.Lenf(t, policyRules, 1, "Expected one policy rule created for ClusterRoles %s", clusterRole.Name)
			policyRule := policyRules[0]
			assert.Lenf(t, policyRule.APIGroups, 1, "Expected one APIGroup entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.APIGroups)
			assert.Contains(t, policyRule.APIGroups, "")
			assert.Lenf(t, policyRule.Resources, 4, "Expected four Resources entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(t, policyRule.Resources, "pods")
			assert.Contains(t, policyRule.Resources, "namespaces")
			assert.Contains(t, policyRule.Resources, "services")
			assert.Contains(t, policyRule.Resources, "daemonsets")
			assert.Lenf(t, policyRule.Verbs, 3, "Expected three Verbs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.Resources)
			assert.Contains(t, policyRule.Verbs, "get")
			assert.Contains(t, policyRule.Verbs, "watch")
			assert.Contains(t, policyRule.Verbs, "list")
			assert.Lenf(t, policyRule.ResourceNames, 0, "Expected no ResourceNames entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.ResourceNames)
			assert.Lenf(t, policyRule.NonResourceURLs, 0, "Expected no NonResourceURLs entry for ClusterRole %s and found %q", clusterRole.Name, policyRule.NonResourceURLs)
		}
	}
	assert.Truef(t, fileBeatFound, "Expected to get a ClusterRole response for %s", constants.FilebeatName)
	assert.Truef(t, journalBeatFound, "Expected to get a ClusterRole response for %s", constants.JournalbeatName)
	assert.Truef(t, nodeExporterFound, "Expected to get a ClusterRole response for %s", constants.NodeExporterName)
	assert.Falsef(t, unexpectedNamesFound, "Unexpected ClusterRoles returned: %q", unexpectedNames)
}
