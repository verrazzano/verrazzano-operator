// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import (
	"time"
)

// VerrazzanoGroup label to be applied to all components related to Verrazzano
const VerrazzanoGroup = "verrazzano.io"

// ResyncPeriod (re-list time period) for controller
const ResyncPeriod = 15 * time.Second

// DefaultNamespace constant for default namespace
const DefaultNamespace = "default"

// ServiceAppLabel label name for service app
const ServiceAppLabel = "app"

// K8SAppLabel label name for k8s app
const K8SAppLabel = "k8s-app"

// VerrazzanoBinding label for Verrazzano binding
const VerrazzanoBinding = "verrazzano.binding"

// VerrazzanoCluster label for Verrazzano cluster
const VerrazzanoCluster = "verrazzano.cluster"

// VerrazzanoPrefix prefix for Verrazzano
const VerrazzanoPrefix = "verrazzano"

// VerrazzanoSystem name for Verrazzano system
const VerrazzanoSystem = "verrazzano-system"

// VerrazzanoSystemAdmin name for Verrazzano system admin
const VerrazzanoSystemAdmin = VerrazzanoSystem + "-admin"

// VerrazzanoNamespace namespace name for Verrazzano system
const VerrazzanoNamespace = VerrazzanoSystem

// VerrazzanoOperatorServiceAccount name for Verrazzano operator service account
const VerrazzanoOperatorServiceAccount = "verrazzano-operator"

// ManifestFile file name for Verrazzano operator manifest
const ManifestFile = "manifest.json"

// DefaultDashboards list of default dashboard coordinates
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

// SystemDashboards list of system dashboard coordinates
var SystemDashboards = []string{
	"dashboards/vmi_dashboard_provider.yml",
	"dashboards/system/system_dashboard.json",
}

// VmiSystemBindingName system binding name for Verrazzano Monitoring Instance
const VmiSystemBindingName = "system"

// VmiUsername user name for Verrazzano Monitoring Instance
const VmiUsername = "verrazzano"

// VmiSecretName secret name for Verrazzano Monitoring Instance
const VmiSecretName = "verrazzano"

// AcmeDNSSecret secret name for use with Cert Manager
const AcmeDNSSecret = "verrazzano-cert-manager-acme-dns"

// AcmeDNSSecretKey secret key for use with Cert Manager
const AcmeDNSSecretKey = "acmedns.json"

// FilebeatName name for Filebeats
const FilebeatName = "filebeat"

// JournalbeatName name for Journalbeats
const JournalbeatName = "journalbeat"

// NodeExporterName name for Node Exporter
const NodeExporterName = "node-exporter"

// MonitoringNamespace name of the monitoring namespace
const MonitoringNamespace = "monitoring"

// LoggingNamespace name for the logging namespace
const LoggingNamespace = "logging"
