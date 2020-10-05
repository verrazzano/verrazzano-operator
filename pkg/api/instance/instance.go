// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	clusterPkg "github.com/verrazzano/verrazzano-operator/pkg/api/clusters"
)

// Instance details of instance returned in API calls.
type Instance struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MgmtCluster   string `json:"mgmtCluster"`
	MgmtPlatform  string `json:"mgmtPlatform"`
	Status        string `json:"status"`
	Version       string `json:"version"`
	KeyCloakURL   string `json:"keyCloakUrl"`
	RancherURL    string `json:"rancherUrl"`
	VzAPIURL      string `json:"vzApiUri"`
	ElasticURL    string `json:"elasticUrl"`
	KibanaURL     string `json:"kibanaUrl"`
	GrafanaURL    string `json:"grafanaUrl"`
	PrometheusURL string `json:"prometheusUrl"`
}

// This is global for the operator
var verrazzanoURI string

// SetVerrazzanoURI set the verrazzanoURI variable
func SetVerrazzanoURI(s string) {
	verrazzanoURI = s
}

// ReturnSingleInstance returns a single instance identified by the secret Kubernetes UID.
func ReturnSingleInstance(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /instance")

	clusters, err := clusterPkg.GetClusters()
	if err != nil {
		msg := fmt.Sprintf("Error getting clusters : %s", err.Error())
		glog.Error(msg)
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
		ID:            "0",
		Name:          GetVerrazzanoName(),
		MgmtCluster:   mgmtCluster.Name,
		MgmtPlatform:  mgmtCluster.Type,
		Status:        mgmtCluster.Status,
		Version:       getVersion(),
		VzAPIURL:      deriveURL("api"),
		RancherURL:    deriveURL("rancher"),
		ElasticURL:    GetElasticURL(),
		KibanaURL:     GetKibanaURL(),
		GrafanaURL:    GetGrafanaURL(),
		PrometheusURL: GetPrometheusURL(),
		KeyCloakURL:   GetKeyCloakURL(),
	}

	json.NewEncoder(w).Encode(instance)
}

func getVersion() string {
	return "0.1.0"
}

// Derive the URL from the verrazzano URI by prefixing with the given URL segment
func deriveURL(s string) string {
	if len(strings.TrimSpace(verrazzanoURI)) > 0 {
		return "https://" + s + "." + verrazzanoURI
	}
	return ""
}

// GetVerrazzanoName returns the environment name portion of the verrazzanoUri
func GetVerrazzanoName() string {
	segs := strings.Split(verrazzanoURI, ".")
	if len(segs) > 1 {
		return segs[0]
	}
	return ""
}

// GetKeyCloakURL returns Keycloak URL
func GetKeyCloakURL() string {
	return deriveURL("keycloak")
}

// GetKibanaURL returns Kibana URL
func GetKibanaURL() string {
	return deriveURL("kibana.vmi.system")
}

// GetGrafanaURL returns Grafana URL
func GetGrafanaURL() string {
	return deriveURL("grafana.vmi.system")
}

// GetPrometheusURL returns Prometheus URL
func GetPrometheusURL() string {
	return deriveURL("prometheus.vmi.system")
}

// GetElasticURL returns Elasticsearch URL
func GetElasticURL() string {
	return deriveURL("elasticsearch.vmi.system")
}

// GetConsoleURL returns the Verrazzano Console URL
func GetConsoleURL() string {
	return deriveURL("console")
}
