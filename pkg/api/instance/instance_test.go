// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const urlTemp = "https://%v.%v"

const vzURI = "abc.v8o.example.com"

// Test KeyCloak Url
func TestGetKeyCloakUrl(t *testing.T) {
	SetVerrazzanoUri(vzURI)
	keycloakURL := fmt.Sprintf(urlTemp, "keycloak", vzURI)
	assert.Equal(t, keycloakURL, GetKeyCloakURL(), "Expected KeyCloakUrl")

	vzURI2 := "xyz.v8o.example.com"
	SetVerrazzanoUri(vzURI2)
	keycloakURL = fmt.Sprintf(urlTemp, "keycloak", vzURI2)
	assert.Equal(t, keycloakURL, GetKeyCloakURL(), "Expected KeyCloakUrl")
}

// Test Kinana Url
func TestGetKibanaUrl(t *testing.T) {
	SetVerrazzanoUri(vzURI)
	expectedURL := fmt.Sprintf(urlTemp, "kibana.vmi.system", vzURI)
	assert.Equal(t, expectedURL, GetKibanaURL(), "Invalid Kibana Url")
}

// Test Grafana Url
func TestGetGrafanaUrl(t *testing.T) {
	SetVerrazzanoUri(vzURI)
	expectedURL := fmt.Sprintf(urlTemp, "grafana.vmi.system", vzURI)
	assert.Equal(t, expectedURL, GetGrafanaURL(), "Invalid Grafana Url")
}

// Test Prometheus Url
func TestGetPrometheusUrl(t *testing.T) {
	SetVerrazzanoUri(vzURI)
	expectedURL := fmt.Sprintf(urlTemp, "prometheus.vmi.system", vzURI)
	assert.Equal(t, expectedURL, GetPrometheusURL(), "Invalid Prometheus Url")
}

// Test Elastic Search Url
func TestGetElasticUrl(t *testing.T) {
	SetVerrazzanoUri(vzURI)
	expectedURL := fmt.Sprintf(urlTemp, "elasticsearch.vmi.system", vzURI)
	assert.Equal(t, expectedURL, GetElasticURL(), "Invalid Elastic Url")
}
