// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

// GetMonitoringComponentLabels returns labels for a given monitoring component.
func GetMonitoringComponentLabels(managedClusterName string, componentName string) map[string]string {
	if componentName == constants.NodeExporterName {
		return GetNodeExporterLabels(managedClusterName)
	}
	return nil
}

// GetMonitoringNamespace return namespace for a given monitoring component.
func GetMonitoringNamespace(componentName string) string {
	if componentName == constants.NodeExporterName {
		return constants.MonitoringNamespace
	}
	return ""
}

// GetMonitoringComponents returns list of monitoring components.
func GetMonitoringComponents() []string {
	var components []string
	components = append(components, constants.NodeExporterName)
	return components
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

func isManagedCluster(clusterInfo ClusterInfo) bool {
	if clusterInfo.ManagedClusterName != "" {
		return true
	}
	return false
}
