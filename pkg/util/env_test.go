// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetEnvValues  tests the various GetXXX env var methods
// GIVEN a request to get the requested Env value
// WHEN valid and invalid values
// THEN The expected value is returned when properly set
func TestGetEnvValues(t *testing.T) {

	const sizeValue = "1Gi"
	testVars := []struct {
		name   string
		value  string
		method func() string
	}{
		{name: "NODE_EXPORTER_IMAGE", value: "foo", method: GetNodeExporterImage},
		{name: "ES_MASTER_NODE_REQUEST_MEMORY", value: sizeValue, method: GetElasticsearchMasterNodeRequestMemory},
		{name: "ES_INGEST_NODE_REQUEST_MEMORY", value: sizeValue, method: GetElasticsearchIngestNodeRequestMemory},
		{name: "ES_DATA_NODE_REQUEST_MEMORY", value: sizeValue, method: GetElasticsearchDataNodeRequestMemory},
		{name: "GRAFANA_REQUEST_MEMORY", value: sizeValue, method: GetGrafanaRequestMemory},
		{name: "GRAFANA_DATA_STORAGE", value: sizeValue, method: GetGrafanaDataStorageSize},
		{name: "PROMETHEUS_REQUEST_MEMORY", value: sizeValue, method: GetPrometheusRequestMemory},
		{name: "PROMETHEUS_DATA_STORAGE", value: sizeValue, method: GetPrometheusDataStorageSize},
		{name: "KIBANA_REQUEST_MEMORY", value: sizeValue, method: GetKibanaRequestMemory},
		{name: "ES_DATA_STORAGE", value: sizeValue, method: GetElasticsearchDataStorageSize},
		{name: "ACCESS_CONTROL_ALLOW_ORIGIN", value: "foo", method: GetAccessControlAllowOrigin},
	}

	for _, tt := range testVars {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.name, tt.value)
			assert.Equal(t, tt.value, tt.method())
			os.Unsetenv(tt.name)
		})
	}

	testIntVars := []struct {
		name   string
		value  string
		intVal int32
		method func() int32
	}{
		{name: "ES_DATA_NODE_REPLICAS", value: "10", intVal: 10, method: GetElasticsearchDataNodeReplicas},
		{name: "ES_INGEST_NODE_REPLICAS", value: "11", intVal: 11, method: GetElasticsearchIngestNodeReplicas},
		{name: "ES_MASTER_NODE_REPLICAS", value: "12", intVal: 12, method: GetElasticsearchMasterNodeReplicas},
	}
	for _, tt := range testIntVars {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.name, tt.value)
			assert.Equal(t, tt.intVal, tt.method())
			os.Unsetenv(tt.name)
		})
	}

	testBoolVars := []struct {
		name    string
		value   string
		boolval bool
		method  func() bool
	}{
		{name: esEnabled, value: "true", boolval: true, method: GetElasticsearchEnabled},
		{name: esEnabled, value: "T", boolval: true, method: GetElasticsearchEnabled},
		{name: esEnabled, value: "false", boolval: false, method: GetElasticsearchEnabled},
		{name: esEnabled, value: "", boolval: true, method: GetElasticsearchEnabled},
		{name: esEnabled, value: "goo", boolval: true, method: GetElasticsearchEnabled},
		{name: promEnabled, value: "true", boolval: true, method: GetPrometheusEnabled},
		{name: promEnabled, value: "false", boolval: false, method: GetPrometheusEnabled},
		{name: kibanaEnabled, value: "true", boolval: true, method: GetKibanaEnabled},
		{name: kibanaEnabled, value: "false", boolval: false, method: GetKibanaEnabled},
		{name: grafanaEnabled, value: "true", boolval: true, method: GetGrafanaEnabled},
		{name: grafanaEnabled, value: "false", boolval: false, method: GetGrafanaEnabled},
	}
	for _, tt := range testBoolVars {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.name, tt.value)
			assert.Equal(t, tt.boolval, tt.method())
			os.Unsetenv(tt.name)
		})
	}
	assert.Equal(t, "container-registry.oracle.com/verrazzano/wl-frontend:324813", GetTestWlsFrontendImage())
}

// TestGetEnvReplicaCount  tests getEnvReplicaCount method
// GIVEN a request to getEnvReplicaCount
// WHEN valid and invalid values
// THEN The expected value is returned when properly set, otherwise the default is returned
func TestGetEnvReplicaCount(t *testing.T) {
	getEnvReplicaCount("foo", 0)
	tests := []struct {
		name          string
		value         string
		setVar        bool
		expectDefault bool
	}{
		{name: "Valid-Nonzero-int", value: "3", setVar: true, expectDefault: false},
		{name: "Zero-val", value: "0", setVar: true, expectDefault: false},
		{name: "Use-default", value: "blah", setVar: true, expectDefault: true},
		{name: "No-env-var", value: "5", setVar: false, expectDefault: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const envVarName = "foo"
			if tt.setVar {
				os.Setenv(envVarName, tt.value)
			}
			var defaultVal int32 = 100
			var expected = defaultVal
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

// TestGetBooleanVarNotSet  tests getBoolean method default if the value is not set; we should return true by default
// GIVEN a request to getBoolean
// WHEN with an env var name that's not set
// THEN True is returned
func TestGetBooleanVarNotSet(t *testing.T) {
	assert.True(t, getBoolean("notset"))
}
