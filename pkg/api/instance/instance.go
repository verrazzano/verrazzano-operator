// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	clusterPkg "github.com/verrazzano/verrazzano-operator/pkg/api/clusters"
)

type Instance struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	MgmtCluster   string `json:"mgmtCluster"`
	MgmtPlatform  string `json:"mgmtPlatform"`
	Status        string `json:"status"`
	Version       string `json:"version"`
	KeyCloakUrl   string `json:"keyCloakUrl"`
	RancherUrl    string `json:"rancherUrl"`
	VzApiUri      string `json:"vzApiUri"`
	ElasticUrl    string `json:"elasticUrl"`
	KibanaUrl     string `json:"kibanaUrl"`
	GrafanaUrl    string `json:"grafanaUrl"`
	PrometheusUrl string `json:"prometheusUrl"`
}

// This is global for the operator
var verrazzanoUri string

func SetVerrazzanoUri(s string) {
	verrazzanoUri = s
}

func ReturnSingleInstance(w http.ResponseWriter, r *http.Request) {
	// Create log instance for returning single instance
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Instance").Str("name", "Return").Logger()

	logger.Info().Msg("GET /instance")

	clusters, err := clusterPkg.GetClusters()
	if err != nil {
		msg := fmt.Sprintf("Error getting clusters : %s", err.Error())
		logger.Error().Msg(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	var mgmtCluster clusterPkg.Cluster
	for _, c := range clusters {
		if c.Name == "local" {
			mgmtCluster = c
			break
		}
	}

	instance := Instance{
		Id:            "0",
		Name:          GetVerrazzanoName(),
		MgmtCluster:   mgmtCluster.Name,
		MgmtPlatform:  mgmtCluster.Type,
		Status:        mgmtCluster.Status,
		Version:       getVersion(),
		VzApiUri:      deriveUrl("api"),
		RancherUrl:    deriveUrl("rancher"),
		ElasticUrl:    GetElasticUrl(),
		KibanaUrl:     GetKibanaUrl(),
		GrafanaUrl:    GetGrafanaUrl(),
		PrometheusUrl: GetPrometheusUrl(),
		KeyCloakUrl:   GetKeyCloakUrl(),
	}

	json.NewEncoder(w).Encode(instance)
}

func getVersion() string {
	return "0.1.0"
}

// Derive the URL from the verrazzano URI by replacing the 'api' segment
func deriveUrl(s string) string {
	return "https://" + s + "." + verrazzanoUri
}

func GetVerrazzanoName() string {
	segs := strings.Split(verrazzanoUri, ".")
	if len(segs) > 1 {
		return segs[0]
	}
	return ""
}

// Get KeyCloak Url
func GetKeyCloakUrl() string {
	return deriveUrl("keycloak")
}

// Get Kibana Url
func GetKibanaUrl() string {
	return deriveUrl("kibana.vmi.system")
}

// Get Grafana Url
func GetGrafanaUrl() string {
	return deriveUrl("grafana.vmi.system")
}

// Get Prometheus Url
func GetPrometheusUrl() string {
	return deriveUrl("prometheus.vmi.system")
}

// Get Elastic Search Url
func GetElasticUrl() string {
	return deriveUrl("elasticsearch.vmi.system")
}
