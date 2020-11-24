// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

// FilebeatConfigData contains configuration used by Filebeats.
const FilebeatConfigData = `filebeat.config:
  inputs:
    # Mounted filebeat-inputs configmap:
    path: ${path.config}/inputs.d/*.yml
    # Reload inputs configs as they change:
    reload.enabled: false
  modules:
    path: ${path.config}/modules.d/*.yml
    # Reload module configs as they change:
    reload.enabled: false
name: ${NODENAME}
filebeat.autodiscover:
  providers:
	- type: kubernetes
	  hints.enabled: true
	  labels.dedot: true
	  annotations.dedot: true
filebeat.inputs:
- type: docker
  containers.ids:
  - "*"
  processors:
  - decode_json_fields:
      fields: ["message"]
      process_array: false
      max_depth: 1
      target: ""
      overwrite_keys: true
  - rename:
      fields:
       - from: "level"
         to: "log.level"
       - from: "caller"
         to: "log.caller"
       - from: "message"
         to: "log.message"
      ignore_missing: true
      fail_on_error: false
  - add_kubernetes_metadata:
      in_cluster: true
setup.template.name: "vmo-local-filebeat"
setup.template.enabled: true
setup.template.overwrite: true
setup.template.json.enabled: true
setup.template.json.path: "/etc/filebeat/es-index-template.json"
setup.template.json.name: "vmo-local-filebeat"
setup.template.pattern: "vmo-local-filebeat-*"
output.elasticsearch:
  hosts: ${ES_URL}
  username: ${ES_USER}
  password: ${ES_PASSWORD}
  index: ${INDEX_NAME}
`

// JournalbeatConfigData contains configuration used by Journalbeats.
const JournalbeatConfigData = `name: ${NODENAME}
journalbeat.inputs:
- paths: []
  seek: cursor
  cursor_seek_fallback: tail
  processors:
  - drop_event:
      when:
        contains:
          message: >-
            Loaded volume plugin "flexvolume-oracle/oci"
logging.to_files: false
setup.template.enabled: false
output.elasticsearch:
  hosts: ${ES_URL}
  username: ${ES_USER}
  password: ${ES_PASSWORD}
  index: ${INDEX_NAME}
`

// FilebeatInputData contains configuration used as inputs for Filebeats.
const FilebeatInputData = `- type: docker
  containers.ids:
  - "*"
  processors:
    - add_kubernetes_metadata:
        in_cluster: true
`

// FilebeatIndexTemplate contains Elasticsearch index template for Filebeats.
const FilebeatIndexTemplate = `{
  "index_patterns" : "vmo-local-filebeat-*",
  "version" : 60001,
  "settings" : {
    "index.refresh_interval" : "5s",
    "index.mapping.total_fields.limit" : "2000",
    "number_of_shards": 5
  },
  "mappings" : {
    "dynamic_templates" : [ {
      "message_field" : {
        "path_match" : "log.message",
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
        }, {
        "match" : "labels.app",
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
