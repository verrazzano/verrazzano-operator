// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"go.uber.org/zap"
	"os"
	"strconv"
)

// This file contains all of the env vars used by the Verrazzano Operator

// GetEnvFunc stores the GetEnv function in a variable, so that tests can override it
var GetEnvFunc = os.Getenv

// Define the ENV vars

const prometheusPusherImage = "PROMETHEUS_PUSHER_IMAGE"
const nodeExporterImage = "NODE_EXPORTER_IMAGE"
const filebeatImage = "FILEBEAT_IMAGE"
const journalbeatImage = "JOURNALBEAT_IMAGE"
const fluentdImage = "FLUENTD_IMAGE"
const esMasterNodeRequestMemory = "ES_MASTER_NODE_REQUEST_MEMORY"
const esIngestNodeRequestMemory = "ES_INGEST_NODE_REQUEST_MEMORY"
const esDataNodeRequestMemory = "ES_DATA_NODE_REQUEST_MEMORY"
const grafanaRequestMemory = "GRAFANA_REQUEST_MEMORY"
const grafanaDataStorageSize = "GRAFANA_DATA_STORAGE"
const prometheusRequestMemory = "PROMETHEUS_REQUEST_MEMORY"
const prometheusDataStorageSize = "PROMETHEUS_DATA_STORAGE"
const kibanaRequestMemory = "KIBANA_REQUEST_MEMORY"
const esDataNodeReplicas = "ES_DATA_NODE_REPLICAS"
const esIngestNodeReplicas = "ES_INGEST_NODE_REPLICAS"
const esIngestNodeReplicasDefault = 1
const esMasterNodeReplicas = "ES_MASTER_NODE_REPLICAS"
const esMasterNodeReplicasDefault = 3
const esDataStorageSize = "ES_DATA_STORAGE"
const esDataNodeReplicasDefault = 2

const accessControlAllowOrigin = "ACCESS_CONTROL_ALLOW_ORIGIN"
const sharedVMIDefault = "USE_SYSTEM_VMI"

// GetPromtheusPusherImage returns the Prometheus Pusher image.
func GetPromtheusPusherImage() string {
	return GetEnvFunc(prometheusPusherImage)
}

// GetNodeExporterImage returns the Node Exporter image.
func GetNodeExporterImage() string {
	return GetEnvFunc(nodeExporterImage)
}

// GetFilebeatImage returns the Filebeats image.
func GetFilebeatImage() string {
	return GetEnvFunc(filebeatImage)
}

// GetJournalbeatImage returns the Journabeats image.
func GetJournalbeatImage() string {
	return GetEnvFunc(journalbeatImage)
}

// GetTestWlsFrontendImage returns a dummy application image for tests.
func GetTestWlsFrontendImage() string {
	return "container-registry.oracle.com/verrazzano/wl-frontend:324813"
}

// GetFluentdImage returns the Fluentd image.
func GetFluentdImage() string {
	return GetEnvFunc(fluentdImage)
}

// GetElasticsearchMasterNodeRequestMemory returns the Elasticsearch master memory request resource.
func GetElasticsearchMasterNodeRequestMemory() string {
	return GetEnvFunc(esMasterNodeRequestMemory)
}

// GetElasticsearchIngestNodeRequestMemory returns the Elasticsearch injest memory request resource.
func GetElasticsearchIngestNodeRequestMemory() string {
	return GetEnvFunc(esIngestNodeRequestMemory)
}

// GetElasticsearchDataNodeRequestMemory returns the Elasticsearch data memory request resource.
func GetElasticsearchDataNodeRequestMemory() string {
	return GetEnvFunc(esDataNodeRequestMemory)
}

// GetElasticsearchDataStorageSize returns the Elasticsearch storage size request
func GetElasticsearchDataStorageSize() string {
	return GetEnvFunc(esDataStorageSize)
}

// GetGrafanaRequestMemory returns the Grafana memory request resource.
func GetGrafanaRequestMemory() string {
	return GetEnvFunc(grafanaRequestMemory)
}

// GetGrafanaDataStorageSize returns the Prometheus storage size request
func GetGrafanaDataStorageSize() string {
	return GetEnvFunc(grafanaDataStorageSize)
}

// GetPrometheusRequestMemory returns the Prometheus memory request resource.
func GetPrometheusRequestMemory() string {
	return GetEnvFunc(prometheusRequestMemory)
}

// GetPrometheusDataStorageSize returns the Prometheus storage size request
func GetPrometheusDataStorageSize() string {
	return GetEnvFunc(prometheusDataStorageSize)
}

// GetKibanaRequestMemory returns the Kibana memory request resource.
func GetKibanaRequestMemory() string {
	return GetEnvFunc(kibanaRequestMemory)
}

// getEnvReplicaCount gets a replica count from the named env var, returning the provided default if not present
func getEnvReplicaCount(envVarName string, defaultValue int32) int32 {
	value := GetEnvFunc(envVarName)
	if len(value) != 0 {
		count, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			zap.S().Infof("%s: %v is invalid.  The default value of %v will be used for node replicas.", envVarName, value, defaultValue)
		} else {
			return int32(count)
		}
	}
	return defaultValue
}

// GetElasticsearchMasterNodeReplicas returns the Elasticsearch master node replicas.
func GetElasticsearchMasterNodeReplicas() int32 {
	return getEnvReplicaCount(esMasterNodeReplicas, esMasterNodeReplicasDefault)
}

// GetElasticsearchDataNodeReplicas returns the Elasticsearch data node replicas.
func GetElasticsearchDataNodeReplicas() int32 {
	return getEnvReplicaCount(esDataNodeReplicas, esDataNodeReplicasDefault)
}

// GetElasticsearchIngestNodeReplicas returns the Elasticsearch ingest node replicas.
func GetElasticsearchIngestNodeReplicas() int32 {
	return getEnvReplicaCount(esIngestNodeReplicas, esIngestNodeReplicasDefault)
}

// GetAccessControlAllowOrigin returns the additional allowed origins for API requests - these
// will be added to the Access-Control-Allow-Origins response header of the API
func GetAccessControlAllowOrigin() string {
	return GetEnvFunc(accessControlAllowOrigin)
}

// GetRancherURL returns Rancher URL
func GetRancherURL() string {
	return GetEnvFunc("rancherURL")
}

// GetRancherHost returns optional host name to use in host headers when accessing Rancher
func GetRancherHost() string {
	return GetEnvFunc("rancherHost")
}
