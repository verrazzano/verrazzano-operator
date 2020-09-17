// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package util

import (
	"github.com/golang/glog"
	"os"
	"strconv"
)

// This file contains all of the env vars used by the Verrazzano Operator
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

func getCohMicroImage() string {
	return os.Getenv(cohMicroImage)
}

func getHelidonMicroImage() string {
	return os.Getenv(helidonMicroImage)
}

func getWlsMicroImage() string {
	return os.Getenv(wlsMicroImage)
}

func GetPromtheusPusherImage() string {
	return os.Getenv(prometheusPusherImage)
}

func GetNodeExporterImage() string {
	return os.Getenv(nodeExporterImage)
}

func GetFilebeatImage() string {
	return os.Getenv(filebeatImage)
}

func GetJournalbeatImage() string {
	return os.Getenv(journalbeatImage)
}

func GetTestWlsFrontendImage() string {
	return "container-registry.oracle.com/verrazzano/wl-frontend:324813"
}

func GetWeblogicOperatorImage() string {
	return os.Getenv(weblogicOperatorImage)
}

func GetFluentdImage() string {
	return os.Getenv(fluentdImage)
}

func GetCohMicroRequestMemory() string {
	return os.Getenv(cohMicroRequestMemory)
}

func GetHelidonMicroRequestMemory() string {
	return os.Getenv(helidonMicroRequestMemory)
}

func GetWlsMicroRequestMemory() string {
	return os.Getenv(wlsMicroRequestMemory)
}

func GetElasticsearchMasterNodeRequestMemory() string {
	return os.Getenv(esMasterNodeRequestMemory)
}

func GetElasticsearchIngestNodeRequestMemory() string {
	return os.Getenv(esIngestNodeRequestMemory)
}

func GetElasticsearchDataNodeRequestMemory() string {
	return os.Getenv(esDataNodeRequestMemory)
}

func GetGrafanaRequestMemory() string {
	return os.Getenv(grafanaRequestMemory)
}

func GetPrometheusRequestMemory() string {
	return os.Getenv(prometheusRequestMemory)
}

func GetKibanaRequestMemory() string {
	return os.Getenv(kibanaRequestMemory)
}

func GetElasticsearchMasterNodeReplicas() int32 {
	value := os.Getenv(esMasterNodeReplicas)
	if len(value) != 0 {
		count, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			glog.V(5).Infof("%v is invalid.  The default value of 3 will be used as replicas of ES master node.", value)
		} else {
			return int32(count)
		}
	}
	return 3
}
