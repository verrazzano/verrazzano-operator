// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package util

import "os"

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
const fluentdKubernetesDaemonsetImage = "FLUENTD_KUBERNETES_DAEMONSET_IMAGE"

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

func GetFluentdKubernetesDaemonsetImage() string {
	return os.Getenv(fluentdKubernetesDaemonsetImage)
}
