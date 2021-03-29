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
	fluentdConfigMap := fluentdConfigMap(managedClusterName)
	if err != nil {
		zap.S().Debugf("New logging config map %s is giving error %s", constants.FluentdName, err)
	}
	configMaps = append(configMaps, fluentdConfigMap, indexconfig, filebeatindexconfig, filebeatconfig, filebeatinput, journalbeatindexconfig, journalbeatconfig)
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

func fluentdConfigMap(managedClusterName string) *corev1.ConfigMap {
	labels := GetFluentdLabels(managedClusterName)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.FluentdConfMapName,
			Namespace: constants.VerrazzanoNamespace,
			Labels:    labels,
		},
		Data: map[string]string{
			confKeyFluent:            confValFluent,
			confKeyFluentdStandalone: confValFluentdStandalone,
			confKeyGeneral:           confValGeneral,
			confKeyPrometheus:        confValPrometheus,
			confKeySystemdInput:      confValSystemdInput,
			confKeySystemdFilter:     confValSystemdFilter,
			confKeyKubernetesInput:   confValKubernetesInput,
			confKeyKubernetesFilter:  confValKubernetesFilter,
			confKeyOutPut:            confValOutPut,
			confKeyExtra:             confValExtra,
			confKeyElasticTemplate:   confValElasticTemplate,
		},
	}
}

const confKeyFluent = "fluent.conf"
const confValFluent = `
    # Use the config specified by the FLUENTD_CONFIG environment variable, or
    # default to fluentd-standalone.conf
    @include "#{ENV['FLUENTD_CONFIG'] || 'fluentd-standalone.conf'}"
`
const confKeyFluentdStandalone = "fluentd-standalone.conf"
const confValFluentdStandalone = `
    # Common config
    @include general.conf
    @include prometheus.conf

    # Input sources
    @include systemd-input.conf
    @include kubernetes-input.conf

    # Parsing/Filtering
    @include systemd-filter.conf
    @include kubernetes-filter.conf
    @include extra.conf

    # Send to storage
    @include output.conf
`
const confKeyGeneral = "general.conf"
const confValGeneral = `
    # Prevent fluentd from handling records containing its own logs. Otherwise
    # it can lead to an infinite loop, when error in sending one message generates
    # another message which also fails to be sent and so on.
    <match fluent.**>
      @type null
      @id out_fluent_null
    </match>

    # Used for health checking
    <source>
      @type http
      @id in_http
      port 9880
      bind 0.0.0.0
    </source>

    # Emits internal metrics to every minute, and also exposes them on port
    # 24220. Useful for determining if an output plugin is retryring/erroring,
    # or determining the buffer queue length.
    <source>
      @type monitor_agent
      @id in_monitor_agent
      bind 0.0.0.0
      port 24220
      tag fluentd.monitor.metrics
    </source>
`
const confKeyPrometheus = "prometheus.conf"
const confValPrometheus = `
    # input plugin that is required to expose metrics by other prometheus
    # plugins, such as the prometheus_monitor input below.
    <source>
      @type prometheus
      @id prometheus
      bind 0.0.0.0
      port 24231
      metrics_path /metrics
    </source>

    # input plugin that collects metrics from MonitorAgent and exposes them
    # as prometheus metrics
    <source>
      @type prometheus_monitor
      @id prometheus_monitor
      # update the metrics every 5 seconds
      interval 5
    </source>

    <source>
      @type prometheus_output_monitor
      @id prometheus_output_monitor
      interval 5
    </source>

    <source>
      @type prometheus_tail_monitor
      @id prometheus_tail_monitor
      interval 5
    </source>
`
const confKeySystemdInput = "systemd-input.conf"
const confValSystemdInput = `
    <source>
      @type systemd
      @id in_systemd
      read_from_head true
      tag systemd
      <storage>
        @type local
        persistent true
        path /tmp/journald_pos.json
        #path /var/log/journald_pos.json
      </storage>
      <entry>
        fields_strip_underscores true
      </entry>
    </source>
`
const confKeySystemdFilter = "systemd-filter.conf"
const confValSystemdFilter = `
    #<match systemd>
    #  @type rewrite_tag_filter
    #  @id systemd_rewrite_tag_filter
    #  # hostname_command sh /fluentd/etc/hostname.sh hostid
    #  hostname_command echo $HOSTNAME
    #  <rule>
    #    key SYSTEMD_UNIT
    #    pattern ^(.+).service$
    #    tag systemd.$1
    #  </rule>
    #  <rule>
    #    key SYSTEMD_UNIT
    #    pattern ^(.+).service$
    #    invert true
    #    tag systemd.unmatched
    #  </rule>
    #</match>

    <filter systemd.kubelet>
      @type parser
      @id systemd_kubelet_parser
      format kubernetes
      reserve_data true
      key_name MESSAGE
    </filter>

    <filter systemd.docker>
      @type parser
      @id systemd_docker_parser
      format /^time="(?<time>[^)]*)" level=(?<severity>[^ ]*) msg="(?<message>[^"]*)"( err="(?<error>[^"]*)")?( statusCode=($<status_code>\d+))?/
      reserve_data true
      key_name MESSAGE
    </filter>

    # Filter filter ssh logs since it's mostly bots trying to login
    <filter systemd.**>
      @type grep
      @id systemd_grep
      <exclude>
        key SYSTEMD_UNIT
        pattern (sshd@.*\.service)
      </exclude>
    </filter>
`
const confKeyKubernetesInput = "kubernetes-input.conf"
const confValKubernetesInput = `
    # Capture Kubernetes pod logs
    # The kubelet creates symlinks that capture the pod name, namespace,
    # container name & Docker container ID to the docker logs for pods in the
    # /var/log/containers directory on the host.
    <source>
      @type tail
      # @id in_tail
      path /var/log/containers/*.log
      pos_file /tmp/fluentd-containers.log.pos
      #pos_file /var/log/fluentd-containers.log.pos
      time_format %Y-%m-%dT%H:%M:%S.%NZ
      tag kubernetes.*
      format json
      read_from_head true
    </source>
`
const confKeyKubernetesFilter = "kubernetes-filter.conf"
const confValKubernetesFilter = `
    # Query the API for extra metadata.
    <filter kubernetes.**>
      @type kubernetes_metadata
      @id kubernetes_metadata
    </filter>

    # rewrite_tag_filter does not support nested fields like
    # kubernetes.container_name, so this exists to flatten the fields
    # so we can use them in our rewrite_tag_filter
    <filter kubernetes.**>
      @type record_transformer
      @id kubernetes_record_transformer
      enable_ruby true
      <record>
       #@target_index vmi-${record["kubernetes"]["namespace_name"]}-${time.strftime('%Y.%m.%d')}
        @target_index vmi-${record["kubernetes"]["namespace_name"]}
      </record>
      <record>
        kubernetes_namespace_container_name ${record["kubernetes"]["namespace_name"]}.${record["kubernetes"]["container_name"]}
      </record>
    </filter>

	# parse verrazzano operators' json log
    <filter kubernetes.**_verrazzano-**_**operator**>
      @type parser
      format json
      reserve_data true
      key_name log
      ignore_key_not_exist true
      replace_invalid_sequence true
      emit_invalid_record_to_error true
    </filter>

    # retag based on the container name of the log message
    #<match kubernetes.**>
    #  @type rewrite_tag_filter
    #  @id kubernetes_rewrite_tag_filter
    #  # hostname_command "echo $HOSTNAME"
    #  <rule>
    #    key kubernetes_namespace_container_name
    #    pattern ^(.+)$
    #    tag kube.$1
    #  </rule>
    #</match>

    # Remove the unnecessary field as the information is already available on
    # other fields.
    <filter kube.**>
      @type record_transformer
      @id kube_record_transformer
      remove_keys kubernetes_namespace_container_name
    </filter>

    <filter kube.kube-system.**>
      @type parser
      @id kube_parser
      format kubernetes
      reserve_data true
      key_name log
    </filter>

    <filter kube.**>
      @type parser
      key_name log
      reserve_data true
      remove_key_name_field false
      emit_invalid_record_to_error false
      hash_value_field wercker
      <parse>
        @type multi_format
        <pattern>
          format json
          time_format %Y-%m-%dT%H:%M:%S.%N%z
        </pattern>
        <pattern>
          format json
          time_format %Y-%m-%dT%H:%M:%S%z
        </pattern>
      </parse>
    </filter>
`
const confKeyOutPut = "output.conf"
const confValOutPut = `
    <match **>
      @type elasticsearch
      @id out_elasticsearch
      @log_level info
      logstash_format true
      include_tag_key true

      template_file /fluentd/etc/elasticsearch-template-vmi.json
      template_name elasticsearch-template-vmi.json
      template_overwrite true

      # time_as_integer needs to be set to false in order for fluentd
      # to convert timestamps to Fluent::EventTime objects instead of
      # integers. Thus allowing fluentd to preserve/transmit
      # nanosecond-precision values.
      time_as_integer false

      # Ensure that the aws-elasticsearch-service plugin preserves
      # nanosecond-precision when formatting Fluent::EventTime values
      # into a JSON payload destined for ElasticSearch.
      # time_key_format %Y-%m-%dT%H:%M:%S.%N%z

      # A value of 9 gives us nanosecond-precision in our timestamps. The default is
      # 0 which effectively gives us only second-precision; we don't want that. The
      # value here comes from Ruby's DateTime::iso8601 method and represents the
      # length of fractional seconds, e.g. 10^-9.
      time_precision 9

      # Set the chunk limit the same as for fluentd-gcp.
      buffer_chunk_limit 4M
      # Cap buffer memory usage to 2MiB/chunk * 32 chunks = 64 MiB
      buffer_queue_limit 32
      flush_interval 5s
      # Never wait longer than 5 minutes between retries.
      max_retry_wait 30
      # Disable the limit on the number of retries (retry forever).
      disable_retry_limit
      # Use multiple threads for processing.
      num_threads 8

      # Prevent reloading connections to AWS ES. This fixes the issues
      # described in https://github.com/wercker/infrastructure/pull/185
      reload_on_failure false
      reload_connections false

      hosts "#{ENV['ELASTICSEARCH_URL']}"
      ca_file /fluentd/secret/ca-bundle
      # ssl_version TLSv1_2
      user "#{ENV['ELASTICSEARCH_USER']}"
      password "#{ENV['ELASTICSEARCH_PASSWORD']}"
      target_index_key @target_index
    </match>
`
const confKeyExtra = "extra.conf"
const confValExtra = `
    # Example filter that adds an extra field "cluster_name" to all log
    # messages:
    # <filter **>
    #   @type record_transformer
    #   <record>
    #     cluster_name "your_cluster_name"
    #   </record>
    # </filter>
`
const confKeyElasticTemplate = "elasticsearch-template-vmi.json"
const confValElasticTemplate = `
    {
      "index_patterns" : "logstash-*",
      "version" : 60001,
      "settings" : {
        "index.refresh_interval" : "5s",
        "index.mapping.total_fields.limit" : "2000",
        "number_of_shards": 5
      },
      "mappings" : {
        "dynamic_templates" : [ {
          "message_field" : {
            "path_match" : "message",
            "match_mapping_type" : "string",
            "mapping" : {
              "type" : "text",
              "norms" : false
            }
          }
        }, {
          "string_fields" : {
            "match" : "*",
            "match_mapping_type" : "string",
            "mapping" : {
              "type" : "text", "norms" : false,
              "fields" : {
                "keyword" : { "type": "keyword", "ignore_above": 256 }
              }
            }
          }
        } ],
        "properties" : {
          "@timestamp": { "type": "date", "format": "strict_date_time||strict_date_optional_time||epoch_millis"},
          "@version": { "type": "keyword"},
          "geoip"  : {
            "dynamic": true,
            "properties" : {
              "ip": { "type": "ip" },
              "location" : { "type" : "geo_point" },
              "latitude" : { "type" : "half_float" },
              "longitude" : { "type" : "half_float" }
            }
          }
        }
      }
    }
`
