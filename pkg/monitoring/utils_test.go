// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"testing"
)

func TestGetMonitoringComponentLabels(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	labels := GetMonitoringComponentLabels(clusterName, constants.FilebeatName)
	assert.Equal(labels, GetFilebeatLabels(clusterName))
	labels = GetMonitoringComponentLabels(clusterName, constants.JournalbeatName)
	assert.Equal(labels, GetJournalbeatLabels(clusterName))
	labels = GetMonitoringComponentLabels(clusterName, constants.NodeExporterName)
	assert.Equal(labels, GetNodeExporterLabels(clusterName))
}

func TestGetMonitoringComponentLabelsBadComp(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	labels := GetMonitoringComponentLabels(clusterName, "")
	assert.Empty(labels)
}

func TestGetMonitoringNamespace(t *testing.T) {
	assert := assert.New(t)
	ns := GetMonitoringNamespace(constants.FilebeatName)
	assert.Equal(ns, constants.LoggingNamespace)
	ns = GetMonitoringNamespace(constants.JournalbeatName)
	assert.Equal(ns, constants.LoggingNamespace)
	ns = GetMonitoringNamespace(constants.NodeExporterName)
	assert.Equal(ns, constants.MonitoringNamespace)
}

func TestGetMonitoringNamespaceBadComp(t *testing.T) {
	assert := assert.New(t)
	ns := GetMonitoringNamespace("")
	assert.Empty(ns)
}

func TestGetMonitoringComponents(t *testing.T) {
	assert := assert.New(t)
	comps := GetMonitoringComponents()
	assert.Lenf(comps, 3, "Wrong number of monitoring components")
	assert.Contains(comps, constants.FilebeatName)
	assert.Contains(comps, constants.JournalbeatName)
	assert.Contains(comps, constants.NodeExporterName)
}

func TestGetFilebeatLabels(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	expected := map[string]string{
		constants.K8SAppLabel:       constants.FilebeatName,
		constants.VerrazzanoBinding: constants.VmiSystemBindingName,
		constants.VerrazzanoCluster: clusterName}

	assert.Equal(expected, GetFilebeatLabels(clusterName))
}

func TestGetJournalbeatLabels(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	expected := map[string]string{
		constants.K8SAppLabel:       constants.JournalbeatName,
		constants.VerrazzanoBinding: constants.VmiSystemBindingName,
		constants.VerrazzanoCluster: clusterName}

	assert.Equal(expected, GetJournalbeatLabels(clusterName))
}

func TestGetNodeExporterLabels(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	expected := map[string]string{
		constants.ServiceAppLabel:   constants.NodeExporterName,
		constants.VerrazzanoBinding: constants.VmiSystemBindingName,
		constants.VerrazzanoCluster: clusterName}

	assert.Equal(expected, GetNodeExporterLabels(clusterName))
}
