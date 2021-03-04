// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"strings"
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
	ContainerRuntime   string
	ManagedClusterName string
}

func getFilebeatConfig(clusterInfo ClusterInfo) string {
	if strings.HasPrefix(clusterInfo.ContainerRuntime, ContainerdContainerRuntimePrefix) {
		return FilebeatConfigDataContainerd
	}
	return FilebeatConfigDataDocker
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
