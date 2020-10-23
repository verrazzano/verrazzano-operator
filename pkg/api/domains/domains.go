// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package domains

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/managed"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

// Domain details of a WebLogic domain returned in API calls.
type Domain struct {
	ID                 string    `json:"id"`
	Style              string    `json:"style"`
	AdminServerAddress string    `json:"adminServerAddress"`
	T3Address          string    `json:"t3Address"`
	Credentials        string    `json:"credentials"`
	Namespace          string    `json:"namespace"`
	Status             string    `json:"status"`
	Servers            []Server  `json:"servers,omitempty"`
	Clusters           []Cluster `json:"clusters,omitempty"`
}

// Server details a WebLogic domain server returned in API calls.
type Server struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	ClusterName string `json:"clusterName"`
	NodeName    string `json:"nodeName"`
	Type        string `json:"type"`
}

// Cluster details a WebLogic domain cluster returned in API calls.
type Cluster struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	InitialSize string `json:"initialSize"`
	MaxSize     string `json:"maxSize"`
	CurrentSize string `json:"currentSize"`
}

var (
	// Domains contains all WebLogic domains.
	Domains   []Domain
	listerSet controller.Listers
)

// Init initialization for domains API.
func Init(listers controller.Listers) {
	listerSet = listers
	refreshDomains()
}

func refreshDomains() {

	// initialize domains as an empty list to avoid json encoding "nil"
	Domains = []Domain{}

	// Find all domains across all model/binding pairs
	for _, mbp := range *listerSet.ModelBindingPairs {

		// Find all domains across all managed clusters within this model/binding pair
		for _, mc := range mbp.ManagedClusters {
			// grab all of the domains
			for _, domain := range mc.WlsDomainCRs {
				Domains = append(Domains, Domain{
					ID: domain.Name,
					Style: func() string {
						if domain.Spec.DomainHomeInImage == true {
							return "domain-in-image"
						}
						return "domain-on-pv"
					}(),
					AdminServerAddress: func() string {
						for _, channel := range domain.Spec.AdminServer.AdminService.Channels {
							if channel.ChannelName == "default" {
								return fmt.Sprintf("http://%s-%s:%s)", domain.Name, managed.GetDomainAdminServerNameAsInAddress(), strconv.FormatInt(int64(channel.NodePort), 10))
							}
						}
						return ""
					}(),
					T3Address: func() string {
						for _, channel := range domain.Spec.AdminServer.AdminService.Channels {
							if channel.ChannelName == "T3Channel" {
								return fmt.Sprintf("t3://%s-%s:%s", domain.Name, managed.GetDomainAdminServerNameAsInAddress(), strconv.FormatInt(int64(channel.NodePort), 10))
							}
						}
						return ""
					}(),
					Credentials: domain.Spec.WebLogicCredentialsSecret.Name,
					Namespace: func() string {
						// find the namespace by locating the "placement" for this component
						for _, placement := range mbp.Binding.Spec.Placement {
							for _, namespace := range placement.Namespaces {
								for _, component := range namespace.Components {
									if component.Name == domain.Name {
										return namespace.Name
									}
								}
							}
						}
						return "unknown"
					}(),
					Status: func() string {
						return "NYI"
					}(),
					Servers: func() []Server {
						servers := []Server{}
						for _, server := range domain.Status.Servers {
							servers = append(servers, Server{
								ID:    server.ServerName,
								State: server.State,
								ClusterName: func() string {
									if len(server.ClusterName) > 0 {
										return server.ClusterName
									}
									return ""
								}(),
								NodeName: func() string {
									if len(server.NodeName) > 0 {
										return server.NodeName
									}
									return ""
								}(),
								Type: func() string {
									if strings.Contains(server.ServerName, "managed") {
										return "managed"
									} else if strings.Contains(server.ServerName, "admin") {
										return "admin"
									}
									return "unknown"
								}(),
							})
						}
						return servers
					}(),
					Clusters: func() []Cluster {
						clusters := []Cluster{}
						for _, cluster := range domain.Spec.Clusters {
							clusters = append(clusters, Cluster{
								ID:          cluster.ClusterName,
								InitialSize: strconv.FormatInt(int64(cluster.Replicas), 10),
							})
						}
						return clusters
					}(),
				})
			}
		}
	}
}

// ReturnAllDomains returns all domains used by model and bindings.
func ReturnAllDomains(w http.ResponseWriter, r *http.Request) {
	// Create log instance for returning all domains
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Domains").Str("name", "Return").Logger()

	refreshDomains()

	logger.Info().Msg("GET /domains")

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Domains)), 10))
	json.NewEncoder(w).Encode(Domains)
}

// ReturnSingleDomain returns a single domain identified by the domain Kubernetes UID.
func ReturnSingleDomain(w http.ResponseWriter, r *http.Request) {
	// Create log instance for returning single domain
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Domains").Str("name", "Return").Logger()

	refreshDomains()
	vars := mux.Vars(r)
	key := vars["id"]

	logger.Info().Msg("GET /domain/" + key)

	for _, domain := range Domains {
		if domain.ID == key {
			json.NewEncoder(w).Encode(domain)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)

}
