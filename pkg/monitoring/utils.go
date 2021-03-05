// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

// GetMonitoringComponentLabels returns labels for a given monitoring component.
func GetMonitoringComponentLabels(managedClusterName string, componentName string) map[string]string {
	if componentName == constants.FilebeatName {
		return GetFilebeatLabels(managedClusterName)
	} else if componentName == constants.JournalbeatName {
		return GetJournalbeatLabels(managedClusterName)
	} else if componentName == constants.NodeExporterName {
		return GetNodeExporterLabels(managedClusterName)
	}
	return nil
}

// GetMonitoringNamespace return namespace for a given monitoring component.
func GetMonitoringNamespace(componentName string) string {
	if componentName == constants.FilebeatName {
		return constants.LoggingNamespace
	} else if componentName == constants.JournalbeatName {
		return constants.LoggingNamespace
	} else if componentName == constants.NodeExporterName {
		return constants.MonitoringNamespace
	} else {
		return ""
	}
}

// GetMonitoringComponents returns list of monitoring components.
func GetMonitoringComponents() []string {
	var components []string
	components = append(components, constants.FilebeatName, constants.JournalbeatName, constants.NodeExporterName)
	return components
}

// GetFilebeatLabels returns labels for Filebeats.
func GetFilebeatLabels(managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.FilebeatName, constants.VerrazzanoBinding: constants.VmiSystemBindingName, constants.VerrazzanoCluster: managedClusterName}
}

// GetJournalbeatLabels returns labels for Journalbeats.
func GetJournalbeatLabels(managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.JournalbeatName, constants.VerrazzanoBinding: constants.VmiSystemBindingName, constants.VerrazzanoCluster: managedClusterName}
}

// GetNodeExporterLabels returns labels for Node Exporter.
func GetNodeExporterLabels(managedClusterName string) map[string]string {
	return map[string]string{constants.ServiceAppLabel: constants.NodeExporterName, constants.VerrazzanoBinding: constants.VmiSystemBindingName, constants.VerrazzanoCluster: managedClusterName}
}

// ClusterInfo has info like ContainerRuntime and managed cluster name
type ClusterInfo struct {
	ContainerRuntime      string
	ManagedClusterName    string
	ElasticsearchURL      string
	ElasticsearchUsername string
	ElasticsearchPassword string
	ElasticsearchCABundle []byte
}

func getFilebeatConfig(clusterInfo ClusterInfo) string {
	config := FilebeatConfigDataDocker
	if strings.HasPrefix(clusterInfo.ContainerRuntime, ContainerdContainerRuntimePrefix) {
		config = FilebeatConfigDataContainerd
	}
	if isManagedCluster(clusterInfo) && len(clusterInfo.ElasticsearchCABundle) > 0 {
		config = config + FileBeatCABundleSetting
	}
	return config
}

func getJournalbeatConfig(clusterInfo ClusterInfo) string {
	config := JournalbeatConfigData
	if isManagedCluster(clusterInfo) && len(clusterInfo.ElasticsearchCABundle) > 0 {
		config = config + JournalBeatCABundleSetting
	}
	return config
}

func getFilebeatInput(clusterInfo ClusterInfo) string {
	if strings.HasPrefix(clusterInfo.ContainerRuntime, ContainerdContainerRuntimePrefix) {
		return FilebeatInputDataContainerd
	}
	return FilebeatInputDataDocker
}

func getFilebeatLogHostPath(clusterInfo ClusterInfo) string {
	if strings.HasPrefix(clusterInfo.ContainerRuntime, ContainerdContainerRuntimePrefix) {
		return FilebeatLogHostPathContainerd
	}
	return FilebeatLogHostPathDocker
}

func isManagedCluster(clusterInfo ClusterInfo) bool {
	if clusterInfo.ManagedClusterName != "" {
		return true
	}
	return false
}

func getElasticsearchURL(clusterInfo ClusterInfo) string {
	if isManagedCluster(clusterInfo) {
		return clusterInfo.ElasticsearchURL
	}
	return fmt.Sprintf("http://vmi-system-es-ingest.%s.svc.cluster.local", constants.VerrazzanoNamespace)
}
