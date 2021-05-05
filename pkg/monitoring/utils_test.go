// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

func TestGetMonitoringComponentLabels(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	labels := GetMonitoringComponentLabels(clusterName, constants.NodeExporterName)
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
	ns := GetMonitoringNamespace(constants.NodeExporterName)
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
	assert.Lenf(comps, 1, "Wrong number of monitoring components")
	assert.Contains(comps, constants.NodeExporterName)
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
