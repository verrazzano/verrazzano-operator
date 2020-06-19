// Copyright (c) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

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

func GetMonitoringNamespace(componentName string) string {
	if componentName == constants.FilebeatName {
		return constants.LoggingNamespace
	} else if componentName == constants.JournalbeatName {
		return constants.LoggingNamespace
	} else {
		return constants.MonitoringNamespace
	}
}

func GetMonitoringComponents() []string {
	var components []string
	components = append(components, constants.FilebeatName, constants.JournalbeatName, constants.NodeExporterName)
	return components
}

func GetFilebeatLabels(managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.FilebeatName, constants.VerrazzanoBinding: constants.VmiSystemBindingName, constants.VerrazzanoCluster: managedClusterName}
}

func GetJournalbeatLabels(managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.JournalbeatName, constants.VerrazzanoBinding: constants.VmiSystemBindingName, constants.VerrazzanoCluster: managedClusterName}
}

func GetNodeExporterLabels(managedClusterName string) map[string]string {
	return map[string]string{constants.ServiceAppLabel: constants.NodeExporterName, constants.VerrazzanoBinding: constants.VmiSystemBindingName, constants.VerrazzanoCluster: managedClusterName}
}
