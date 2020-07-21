// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

const (
	// istio related
	IstioNamespace        = "istio-system"
	IstioPrometheusCMName = "prometheus"
	PrometheusYml         = "prometheus.yml"
	PrometheusLabel       = "app=prometheus"

	// general prometheus scrape config labels
	PrometheusScrapeConfigsLabel = "scrape_configs"
	PrometheusJobNameLabel       = "job_name"
	BasicAuthLabel               = "basic_auth"
	UsernameLabel                = "username"
	PasswordLabel                = "password"

	// prometheus scrape config placeholders
	JobNameHolder               = "##JOB_NAME##"
	NamespaceHolder             = "##NAMESPACE##"
	KeepSourceLabelsHolder      = "##KEEP_SOURCE_LABELS##"
	KeepSourceLabelsRegexHolder = "##KEEP_SOURCE_LABELS_REGEX##"
	ComponentBindingNameHolder  = "##COMPONENT_BINDING_NAME##"

	// node exporter related
	NodeExporterName                  = "system-node-exporter"
	NodeExporterNamespace             = "monitoring"
	NodeExporterKeepSourceLabels      = "__meta_kubernetes_pod_label_app"
	NodeExporterKeepSourceLabelsRegex = "node-exporter"

	// weblogic binding related
	WeblogicOperatorName                  = "weblogic-operator"
	WeblogicOperatorKeepSourceLabels      = "__meta_kubernetes_pod_label_app"
	WeblogicOperatorKeepSourceLabelsRegex = "weblogic-operator"
	WeblogicName                          = "weblogic"
	WeblogicKeepSourceLabels              = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_weblogic_domainName"
	WeblogicKeepSourceLabelsRegex         = "true;##COMPONENT_BINDING_NAME##"

	// coherence binding related
	CoherenceOperatorName                  = "coherence-operator"
	CoherenceOperatorKeepSourceLabels      = "__meta_kubernetes_pod_label_app,__meta_kubernetes_pod_container_port_name"
	CoherenceOperatorKeepSourceLabelsRegex = "coherence-operator;(metrics|oper-metrics)"
	CoherenceName                          = "coherence"
	CoherenceKeepSourceLabels              = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_coherenceCluster"
	CoherenceKeepSourceLabelsRegex         = "true;##COMPONENT_BINDING_NAME##"

	// helidon binding related
	HelidonName                  = "helidon"
	HelidonKeepSourceLabels      = "__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_app"
	HelidonKeepSourceLabelsRegex = "true;##COMPONENT_BINDING_NAME##"
)

// general prometheus scrape target template
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
