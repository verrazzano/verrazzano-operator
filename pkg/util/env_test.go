// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
)

func TestGetEnvValues(t *testing.T) {
	assert := assert.New(t)
	const sizeValue = "1Gi"

	testVars := []struct {
		name  string
		value string
	}{
		{name: "COH_MICRO_IMAGE", value: "foo"},
		{name: "HELIDON_MICRO_IMAGE", value: "foo"},
		{name: "WLS_MICRO_IMAGE", value: "foo"},
		{name: "PROMETHEUS_PUSHER_IMAGE", value: "foo"},
		{name: "NODE_EXPORTER_IMAGE", value: "foo"},
		{name: "FILEBEAT_IMAGE", value: "foo"},
		{name: "JOURNALBEAT_IMAGE", value: "foo"},
		{name: "WEBLOGIC_OPERATOR_IMAGE", value: "foo"},
		{name: "FLUENTD_IMAGE", value: "foo"},
		{name: "COH_MICRO_REQUEST_MEMORY", value: "foo"},
		{name: "HELIDON_MICRO_REQUEST_MEMORY", value: sizeValue},
		{name: "WLS_MICRO_REQUEST_MEMORY", value: sizeValue},
		{name: "ES_MASTER_NODE_REQUEST_MEMORY", value: sizeValue},
		{name: "ES_INGEST_NODE_REQUEST_MEMORY", value: sizeValue},
		{name: "ES_DATA_NODE_REQUEST_MEMORY", value: sizeValue},
		{name: "GRAFANA_REQUEST_MEMORY", value: sizeValue},
		{name: "GRAFANA_DATA_STORAGE", value: sizeValue},
		{name: "PROMETHEUS_REQUEST_MEMORY", value: sizeValue},
		{name: "PROMETHEUS_DATA_STORAGE", value: sizeValue},
		{name: "KIBANA_REQUEST_MEMORY", value: sizeValue},
		{name: "ES_DATA_NODE_REPLICAS", value: "1"},
		{name: "ES_INGEST_NODE_REPLICAS", value: "1"},
		{name: "ES_MASTER_NODE_REPLICAS", value: "1"},
		{name: "ES_DATA_STORAGE", value: sizeValue},
		{name: "ACCESS_CONTROL_ALLOW_ORIGIN", value: "foo"},
	}

	for _, envVar := range testVars {
		os.Setenv(envVar.name, envVar.value)
	}

	assert.NotNil(GetAccessControlAllowOrigin(),
		"GetElasticsearchDataStorageSize is not null")

	assert.NotNil(GetCohMicroRequestMemory(),
		"GetCohMicroRequestMemory is not null")

	assert.NotNil(GetElasticsearchDataNodeReplicas(),
		"GetElasticsearchDataNodeReplicas is not null")

	assert.NotNil(GetElasticsearchDataNodeRequestMemory(),
		"GetElasticsearchDataNodeRequestMemory is not null")

	assert.NotNil(GetElasticsearchIngestNodeReplicas(),
		"GetElasticsearchIngestNodeReplicas is not null")

	assert.NotNil(GetElasticsearchDataStorageSize(),
		"GetElasticsearchDataStorageSize is not null")

	assert.NotNil(GetElasticsearchIngestNodeRequestMemory(),
		"GetElasticsearchIngestNodeRequestMemory is not null")

	assert.NotNil(GetElasticsearchMasterNodeReplicas(),
		"GetElasticsearchMasterNodeReplicas is not null")

	assert.NotNil(GetElasticsearchMasterNodeRequestMemory(),
		"GetElasticsearchMasterNodeRequestMemory is not null")

	assert.NotNil(GetFilebeatImage(),
		"GetFilebeatImage is not null")

	assert.NotNil(GetFluentdImage(),
		"GetFluentdImage is not null")

	assert.NotNil(GetGrafanaDataStorageSize(),
		"GetGrafanaDataStorageSize is not null")

	assert.NotNil(GetGrafanaRequestMemory(),
		"GetGrafanaRequestMemory is not null")

	assert.NotNil(GetHelidonMicroRequestMemory(),
		"GetHelidonMicroRequestMemory is not null")

	assert.NotNil(GetJournalbeatImage(),
		"GetJournalbeatImage is not null")

	assert.NotNil(GetKibanaRequestMemory(),
		"GetKibanaRequestMemory is not null")

	assert.NotNil(GetNodeExporterImage(),
		"GetNodeExporterImage is not null")

	assert.NotNil(GetPrometheusDataStorageSize(),
		"GetPrometheusDataStorageSize is not null")

	assert.NotNil(GetPrometheusRequestMemory(),
		"GetPrometheusRequestMemory is not null")

	assert.NotNil(GetPromtheusPusherImage(),
		"GetPromtheusPusherImage is not null")

	assert.NotNil(GetTestWlsFrontendImage(),
		"GetTestWlsFrontendImage is not null")

	assert.NotNil(GetWeblogicOperatorImage(),
		"GetWeblogicOperatorImage is not null")

	assert.NotNil(GetWlsMicroRequestMemory(),
		"GetWlsMicroRequestMemory is not null")

	for _, envVar := range testVars {
		os.Unsetenv(envVar.name)
	}
}

func TestGetEnvReplicaCount(t *testing.T) {
	assert := assert.New(t)
	getEnvReplicaCount("foo", 0)
	tests := []struct {
		name          string
		value         string
		setVar        bool
		expectDefault bool
	}{
		{
			name:          "Valid-Nonzero-int",
			value:         "3",
			setVar:        true,
			expectDefault: false,
		},
		{
			name:          "Zero-val",
			value:         "0",
			setVar:        true,
			expectDefault: false,
		},
		{
			name:          "Use-default",
			value:         "blah",
			setVar:        true,
			expectDefault: true,
		},
		{
			name:          "No-env-var",
			value:         "5",
			setVar:        false,
			expectDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const envVarName = "foo"
			if tt.setVar {
				os.Setenv(envVarName, tt.value)
			}
			var defaultVal int32 = 100
			var expected int32 = defaultVal
			if !tt.expectDefault {
				val, _ := strconv.ParseInt(tt.value, 10, 32)
				expected = int32(val)
			}
			actual := getEnvReplicaCount("foo", defaultVal)
			os.Unsetenv(envVarName)
			assert.Equal(t, int32(expected), actual)
		})
	}
}
