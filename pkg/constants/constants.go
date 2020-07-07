// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import (
	"time"
)

type StorageOperationType string

// To be applied to all components related to Verrazzano
const VerrazzanoGroup = "verrazzano.io"

const ResyncPeriod = 15 * time.Second
const DefaultNamespace = "default"
const ServiceAppLabel = "app"
const K8SAppLabel = "k8s-app"

// Label values
const VerrazzanoBinding = "verrazzano.binding"
const VerrazzanoCluster = "verrazzano.cluster"

const VerrazzanoPrefix = "verrazzano"
const VerrazzanoSystem = "verrazzano-system"
const VerrazzanoSystemAdmin = VerrazzanoSystem + "-admin"
const VerrazzanoNamespace = VerrazzanoSystem

const ManifestFile = "manifest.json"

// Dashboard coordinates
var DefaultDashboards = []string{
	"dashboards/vmi_dashboard_provider.yml",
	"dashboards/weblogic/weblogic_dashboard.json",
	"dashboards/coherence/elastic-data-summary-dashboard.json",
	"dashboards/coherence/persistence-summary-dashboard.json",
	"dashboards/coherence/cache-details-dashboard.json",
	"dashboards/coherence/members-summary-dashboard.json",
	"dashboards/coherence/kubernetes-summary-dashboard.json",
	"dashboards/coherence/coherence-dashboard-main.json",
	"dashboards/coherence/caches-summary-dashboard.json",
	"dashboards/coherence/service-details-dashboard.json",
	"dashboards/coherence/proxy-servers-summary-dashboard.json",
	"dashboards/coherence/federation-details-dashboard.json",
	"dashboards/coherence/federation-summary-dashboard.json",
	"dashboards/coherence/services-summary-dashboard.json",
	"dashboards/coherence/http-servers-summary-dashboard.json",
	"dashboards/coherence/proxy-server-detail-dashboard.json",
	"dashboards/coherence/alerts-dashboard.json",
	"dashboards/coherence/member-details-dashboard.json",
	"dashboards/coherence/machines-summary-dashboard.json",
}

var SystemDashboards = []string{
	"dashboards/vmi_dashboard_provider.yml",
	"dashboards/system/system_dashboard.json",
}

// VMI related
const VmiSystemBindingName = "system"
const VmiUsername = "verrazzano"
const VmiSecretName = "verrazzano"

// Cert-Manager related
const AcmeDnsSecret = "verrazzano-cert-manager-acme-dns"
const AcmeDnsSecretKey = "acmedns.json"

// Monitoring related variables
const FilebeatName = "filebeat"
const JournalbeatName = "journalbeat"
const NodeExporterName = "node-exporter"

const MonitoringNamespace = "monitoring"
const LoggingNamespace = "logging"
