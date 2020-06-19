// Copyright (c) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const urlTemp = "https://%v.%v"

const vzUri = "abc.v8o.oracledx.com"

// Test KeyCloak Url
func TestGetKeyCloakUrl(t *testing.T) {
	SetVerrazzanoUri(vzUri)
	keycloakUrl := fmt.Sprintf(urlTemp, "keycloak", vzUri)
	assert.Equal(t, keycloakUrl, GetKeyCloakUrl(), "Expected KeyCloakUrl")

	vzUri2 := "xyz.v8o.oracledx.com"
	SetVerrazzanoUri(vzUri2)
	keycloakUrl = fmt.Sprintf(urlTemp, "keycloak", vzUri2)
	assert.Equal(t, keycloakUrl, GetKeyCloakUrl(), "Expected KeyCloakUrl")
}

// Test Kinana Url
func TestGetKibanaUrl(t *testing.T) {
	SetVerrazzanoUri(vzUri)
	expectedUrl := fmt.Sprintf(urlTemp, "kibana.vmi.system", vzUri)
	assert.Equal(t, expectedUrl, GetKibanaUrl(), "Invalid Kibana Url")
}

// Test Grafana Url
func TestGetGrafanaUrl(t *testing.T) {
	SetVerrazzanoUri(vzUri)
	expectedUrl := fmt.Sprintf(urlTemp, "grafana.vmi.system", vzUri)
	assert.Equal(t, expectedUrl, GetGrafanaUrl(), "Invalid Grafana Url")
}

// Test Prometheus Url
func TestGetPrometheusUrl(t *testing.T) {
	SetVerrazzanoUri(vzUri)
	expectedUrl := fmt.Sprintf(urlTemp, "prometheus.vmi.system", vzUri)
	assert.Equal(t, expectedUrl, GetPrometheusUrl(), "Invalid Prometheus Url")
}

// Test Elastic Search Url
func TestGetElasticUrl(t *testing.T) {
	SetVerrazzanoUri(vzUri)
	expectedUrl := fmt.Sprintf(urlTemp, "elasticsearch.vmi.system", vzUri)
	assert.Equal(t, expectedUrl, GetElasticUrl(), "Invalid Elastic Url")
}
