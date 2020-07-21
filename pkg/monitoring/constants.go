// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

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
filebeat.inputs:
- type: docker
  containers.ids:
  - "*"
  processors:
  - add_kubernetes_metadata:
      in_cluster: true
setup.template.enabled: false
output.elasticsearch:
  hosts: ${ES_URL}
  username: ${ES_USER}
  password: ${ES_PASSWORD}
  index: ${INDEX_NAME}
`

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

const FilebeatInputData = `- type: docker
  containers.ids:
  - "*"
  processors:
    - add_kubernetes_metadata:
        in_cluster: true
`
