// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoggingConfigMaps gets all the config maps needed by Filebeats and Journalbeats in all the managed cluster.
func LoggingConfigMaps(managedClusterName string, clusterInfo ClusterInfo) []*corev1.ConfigMap {
	filebeatLabels := GetFilebeatLabels(managedClusterName)
	journalbeatLabels := GetJournalbeatLabels(managedClusterName)
	var configMaps []*corev1.ConfigMap

	indexconfig, err := createLoggingConfigMap(constants.LoggingNamespace, "index-config", "index_name", "vmo-demo-filebeat", filebeatLabels)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", indexconfig.Name, err)
	}
	filebeatindexconfig, err := createLoggingConfigMap(constants.LoggingNamespace, "filebeat-index-config", "filebeat-index-name", "vmo-"+managedClusterName+"-filebeat-%{+yyyy.MM.dd}", filebeatLabels)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", filebeatindexconfig.Name, err)
	}

	filebeatconfig, err := createLoggingConfigMap(constants.LoggingNamespace, "filebeat-config", "filebeat.yml", getFilebeatConfig(clusterInfo), filebeatLabels)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", filebeatconfig.Name, err)
	}
	filebeatconfig.Data["es-index-template.json"] = FilebeatIndexTemplate
	filebeatinput, err := createLoggingConfigMap(constants.LoggingNamespace, "filebeat-inputs", "kubernetes.yml", getFilebeatInput(clusterInfo), filebeatLabels)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", filebeatinput.Name, err)
	}
	journalbeatindexconfig, err := createLoggingConfigMap(constants.LoggingNamespace, "journalbeat-index-config", "journalbeat-index-name", "vmo-"+managedClusterName+"-journalbeat-%{+yyyy.MM.dd}", journalbeatLabels)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", journalbeatindexconfig.Name, err)
	}
	journalbeatconfig, err := createLoggingConfigMap(constants.LoggingNamespace, "journalbeat-config", "journalbeat.yml", getJournalbeatConfig(clusterInfo), journalbeatLabels)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", journalbeatconfig.Name, err)
	}
	configMaps = append(configMaps, indexconfig, filebeatindexconfig, filebeatconfig, filebeatinput, journalbeatindexconfig, journalbeatconfig)
	return configMaps
}

// Constructs the necessary ConfigMaps for logging
func createLoggingConfigMap(namespace string, cmname string, cmfile string, cmdata string, labels map[string]string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmname,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string]string{cmfile: cmdata},
	}
	return configMap, nil
}
