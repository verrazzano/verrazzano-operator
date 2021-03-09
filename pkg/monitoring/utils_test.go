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

func Test_getFilebeatConfig(t *testing.T) {
	// not a managed cluster
	config := getFilebeatConfig(ClusterInfo{ContainerRuntime: "docker://123"})
	assert.Equal(t, FilebeatConfigDataDocker, config, "expected filebeat config")
	config = getFilebeatConfig(ClusterInfo{ContainerRuntime: "containerd://123", ManagedClusterName: ""})
	assert.Equal(t, FilebeatConfigDataContainerd, config, "expected filebeat config")

	// managed cluster, without ca bundle
	config = getFilebeatConfig(ClusterInfo{ContainerRuntime: "docker://123", ManagedClusterName: "cluster1"})
	assert.Equal(t, FilebeatConfigDataDocker, config, "expected filebeat config")
	config = getFilebeatConfig(ClusterInfo{ContainerRuntime: "containerd://123", ManagedClusterName: "cluster1"})
	assert.Equal(t, FilebeatConfigDataContainerd, config, "expected filebeat config")

	// managed cluster, with ca bundle
	config = getFilebeatConfig(ClusterInfo{ContainerRuntime: "docker://123", ManagedClusterName: "cluster1", ElasticsearchCABundle: []byte("testCABundle")})
	assert.Equal(t, FilebeatConfigDataDocker+FileBeatCABundleSetting, config, "expected filebeat config")
	config = getFilebeatConfig(ClusterInfo{ContainerRuntime: "containerd://123", ManagedClusterName: "cluster1", ElasticsearchCABundle: []byte("testCABundle")})
	assert.Equal(t, FilebeatConfigDataContainerd+FileBeatCABundleSetting, config, "expected filebeat config")
}

func Test_getJournalbeatConfig(t *testing.T) {
	// not a managed cluster
	config := getJournalbeatConfig(ClusterInfo{})
	assert.Equal(t, JournalbeatConfigData, config, "expected journalbeat config")

	// managed cluster, without ca bundle
	config = getJournalbeatConfig(ClusterInfo{ManagedClusterName: "cluster1"})
	assert.Equal(t, JournalbeatConfigData, config, "expected journalbeat config")

	// managed cluster, with ca bundle
	config = getJournalbeatConfig(ClusterInfo{ManagedClusterName: "cluster1", ElasticsearchCABundle: []byte("testCABundle")})
	assert.Equal(t, JournalbeatConfigData+JournalBeatCABundleSetting, config, "expected journalbeat config")
}

func Test_getElasticsearchURL(t *testing.T) {
	testURL := "testURL"
	url := getElasticsearchURL(ClusterInfo{ManagedClusterName: "", ElasticsearchURL: testURL})
	assert.Equal(t, "http://vmi-system-es-ingest.verrazzano-system.svc.cluster.local", url, "expected elasticsearch url")
	url = getElasticsearchURL(ClusterInfo{ManagedClusterName: "cluster1", ElasticsearchURL: testURL})
	assert.Equal(t, testURL, url, "expected elasticsearch url")
}

func Test_getFilebeatInput(t *testing.T) {
	input := getFilebeatInput(ClusterInfo{ContainerRuntime: "docker://123"})
	assert.Equal(t, FilebeatInputDataDocker, input, "expected filebeat input")
	input = getFilebeatInput(ClusterInfo{ContainerRuntime: "containerd://123"})
	assert.Equal(t, FilebeatInputDataContainerd, input, "expected filebeat input")
}

func Test_getFilebeatLogHostPath(t *testing.T) {
	path := getFilebeatLogHostPath(ClusterInfo{ContainerRuntime: "docker://123"})
	assert.Equal(t, FilebeatLogHostPathDocker, path, "expected filebeat log host path")
	path = getFilebeatLogHostPath(ClusterInfo{ContainerRuntime: "containerd://123"})
	assert.Equal(t, FilebeatLogHostPathContainerd, path, "expected filebeat log host path")
}
