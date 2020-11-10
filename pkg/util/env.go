// Copyright (C) 2020, Oracle and/or its affiliates.
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

const cohMicroImage = "COH_MICRO_IMAGE"
const helidonMicroImage = "HELIDON_MICRO_IMAGE"
const wlsMicroImage = "WLS_MICRO_IMAGE"
const prometheusPusherImage = "PROMETHEUS_PUSHER_IMAGE"
const nodeExporterImage = "NODE_EXPORTER_IMAGE"
const filebeatImage = "FILEBEAT_IMAGE"
const journalbeatImage = "JOURNALBEAT_IMAGE"
const weblogicOperatorImage = "WEBLOGIC_OPERATOR_IMAGE"
const fluentdImage = "FLUENTD_IMAGE"
const cohMicroRequestMemory = "COH_MICRO_REQUEST_MEMORY"
const helidonMicroRequestMemory = "HELIDON_MICRO_REQUEST_MEMORY"
const wlsMicroRequestMemory = "WLS_MICRO_REQUEST_MEMORY"
const esMasterNodeRequestMemory = "ES_MASTER_NODE_REQUEST_MEMORY"
const esIngestNodeRequestMemory = "ES_INGEST_NODE_REQUEST_MEMORY"
const esDataNodeRequestMemory = "ES_DATA_NODE_REQUEST_MEMORY"
const grafanaRequestMemory = "GRAFANA_REQUEST_MEMORY"
const prometheusRequestMemory = "PROMETHEUS_REQUEST_MEMORY"
const kibanaRequestMemory = "KIBANA_REQUEST_MEMORY"
const esMasterNodeReplicas = "ES_MASTER_NODE_REPLICAS"
const accessControlAllowOrigin = "ACCESS_CONTROL_ALLOW_ORIGIN"

func getCohMicroImage() string {
	return GetEnvFunc(cohMicroImage)
}

func getHelidonMicroImage() string {
	return GetEnvFunc(helidonMicroImage)
}

func getWlsMicroImage() string {
	return GetEnvFunc(wlsMicroImage)
}

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

// GetWeblogicOperatorImage returns the WebLogic Kubernetes Operator image.
func GetWeblogicOperatorImage() string {
	return GetEnvFunc(weblogicOperatorImage)
}

// GetFluentdImage returns the Fluentd image.
func GetFluentdImage() string {
	return GetEnvFunc(fluentdImage)
}

// GetCohMicroRequestMemory returns the Coherence micro operator memory request resource.
func GetCohMicroRequestMemory() string {
	return GetEnvFunc(cohMicroRequestMemory)
}

// GetHelidonMicroRequestMemory returns the Helidon App micro operator memory request resource.
func GetHelidonMicroRequestMemory() string {
	return GetEnvFunc(helidonMicroRequestMemory)
}

// GetWlsMicroRequestMemory returns the Weblogic micro operator memory request resource.
func GetWlsMicroRequestMemory() string {
	return GetEnvFunc(wlsMicroRequestMemory)
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

// GetGrafanaRequestMemory returns the Grafana memory request resource.
func GetGrafanaRequestMemory() string {
	return GetEnvFunc(grafanaRequestMemory)
}

// GetPrometheusRequestMemory returns the Prometheus memory request resource.
func GetPrometheusRequestMemory() string {
	return GetEnvFunc(prometheusRequestMemory)
}

// GetKibanaRequestMemory returns the Kibana memory request resource.
func GetKibanaRequestMemory() string {
	return GetEnvFunc(kibanaRequestMemory)
}

// GetElasticsearchMasterNodeReplicas returns the Elasticsearch master replicas.
func GetElasticsearchMasterNodeReplicas() int32 {
	value := GetEnvFunc(esMasterNodeReplicas)
	if len(value) != 0 {
		count, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			zap.S().Infof("%v is invalid.  The default value of 3 will be used as replicas of ES master node.", value)
		} else {
			return int32(count)
		}
	}
	return 3
}

// GetAccessControlAllowOrigin returns the additional allowed origins for API requests - these
// will be added to the Access-Control-Allow-Origins response header of the API
func GetAccessControlAllowOrigin() string {
	return GetEnvFunc(accessControlAllowOrigin)
}
