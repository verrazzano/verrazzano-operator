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
