// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

const (
	//
	// istio related
	//

	// IstioNamespace namespace name for istio system
	IstioNamespace = "istio-system"
	// IstioPrometheusCMName config map name for istio prometheus
	IstioPrometheusCMName = "prometheus"
	// PrometheusYml Prometheus config file
	PrometheusYml = "prometheus.yml"
	// PrometheusLabel app label for Prometheus
	PrometheusLabel = "app=prometheus"

	//
	// general prometheus scrape config labels
	//

	// PrometheusScrapeConfigsLabel label for Prometheus scrape config
	PrometheusScrapeConfigsLabel = "scrape_configs"
	// PrometheusJobNameLabel label for Prometheus job name
	PrometheusJobNameLabel = "job_name"
	// BasicAuthLabel label for Prometheus basic auth
	BasicAuthLabel = "basic_auth"
	// UsernameLabel label for Prometheus username
	UsernameLabel = "username"
	// PasswordLabel label for Prometheus password
	PasswordLabel = "password"

	//
	// prometheus scrape config placeholders
	//

	// JobNameHolder placeholder for job name
	JobNameHolder = "##JOB_NAME##"
	// NamespaceHolder placeholder for namespace
	NamespaceHolder = "##NAMESPACE##"
	// KeepSourceLabelsHolder placeholder for keep source labels
	KeepSourceLabelsHolder = "##KEEP_SOURCE_LABELS##"
	// KeepSourceLabelsRegexHolder placeholder for keep source labels regex
	KeepSourceLabelsRegexHolder = "##KEEP_SOURCE_LABELS_REGEX##"
	// ComponentBindingNameHolder placeholder for component binding name
	ComponentBindingNameHolder = "##COMPONENT_BINDING_NAME##"

	//
	// node exporter related
	//

	// NodeExporterName name for system Node Exporter
	NodeExporterName = "system-node-exporter"
	// NodeExporterNamespace namespace for Node Exporter
	NodeExporterNamespace = "monitoring"
	// NodeExporterKeepSourceLabels keep source labels for Node Exporter
	NodeExporterKeepSourceLabels = "__meta_kubernetes_pod_label_app"
	// NodeExporterKeepSourceLabelsRegex keep source labels regex for Node Exporter
	NodeExporterKeepSourceLabelsRegex = "node-exporter"

	//
	// weblogic binding related
	//

	// WeblogicOperatorName name for WebLogic operator
	WeblogicOperatorName = "weblogic-operator"
	// WeblogicOperatorKeepSourceLabels keep source labels for WebLogic operator
	WeblogicOperatorKeepSourceLabels = "__meta_kubernetes_pod_label_app"
	// WeblogicOperatorKeepSourceLabelsRegex keep source labels regex for WebLogic operator
	WeblogicOperatorKeepSourceLabelsRegex = "weblogic-operator"
	// WeblogicName name for WebLogic
	WeblogicName = "weblogic"
	// WeblogicKeepSourceLabels keep source labels for WebLogic
	WeblogicKeepSourceLabels = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_weblogic_domainName"
	// WeblogicKeepSourceLabelsRegex keep source labels regex for WebLogic
	WeblogicKeepSourceLabelsRegex = "true;##COMPONENT_BINDING_NAME##"

	//
	// coherence binding related
	//

	// CoherenceOperatorName name for Coherence operator
	CoherenceOperatorName = "coherence-operator"
	// CoherenceOperatorKeepSourceLabels keep source labels for Coherence operator
	CoherenceOperatorKeepSourceLabels = "__meta_kubernetes_pod_label_app,__meta_kubernetes_pod_container_port_name"
	// CoherenceOperatorKeepSourceLabelsRegex keep source labels regex for Coherence operator
	CoherenceOperatorKeepSourceLabelsRegex = "coherence-operator;(metrics|oper-metrics)"
	// CoherenceName name for Coherence
	CoherenceName = "coherence"
	// CoherenceKeepSourceLabels keep source labels for Coherence
	CoherenceKeepSourceLabels = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_coherenceCluster"
	// CoherenceKeepSourceLabelsRegex keep source labels regex for Coherence
	CoherenceKeepSourceLabelsRegex = "true;##COMPONENT_BINDING_NAME##"

	//
	// helidon binding related
	//

	// HelidonName name for Helidon
	HelidonName = "helidon"
	// HelidonKeepSourceLabels keep source labels for Helidon
	HelidonKeepSourceLabels = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_app"
	// HelidonKeepSourceLabelsRegex keep source labels regex for Helidon
	HelidonKeepSourceLabelsRegex = "true;##COMPONENT_BINDING_NAME##"

	//
	// Generic Component binding related
	//

	// Name name for Generic Component
	GenericName = "generic"
	// GenericKeepSourceLabels keep source labels for Helidon
	GenericKeepSourceLabels = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_app"
	// GenericKeepSourceLabelsRegex keep source labels regex for Helidon
	GenericKeepSourceLabelsRegex = "true;##COMPONENT_BINDING_NAME##"
)

// PrometheusScrapeConfigTemplate configuration for general prometheus scrape target template.
const PrometheusScrapeConfigTemplate = `
job_name: ##JOB_NAME##
kubernetes_sd_configs:
- role: pod
  namespaces:
    names:
      - ##NAMESPACE##
relabel_configs:
- action: keep
  source_labels: [##KEEP_SOURCE_LABELS##]
  regex: ##KEEP_SOURCE_LABELS_REGEX##
- source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
  action: replace
  target_label: __metrics_path__
  regex: (.+)
- source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
  action: replace
  regex: ([^:]+)(?::\d+)?;(\d+)
  replacement: $1:$2
  target_label: __address__
- action: labelmap
  regex: __meta_kubernetes_pod_label_(.+)
- source_labels: [__meta_kubernetes_pod_name]
  action: replace
  target_label: pod_name
- regex: '(controller_revision_hash)'
  action: labeldrop
- source_labels: [name]
  regex: '.*/(.*)$'
  replacement: $1
  target_label: webapp
`
