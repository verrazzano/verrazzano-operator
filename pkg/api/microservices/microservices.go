// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package microservices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

// Microservice details of microservice returned in API calls.
type Microservice struct {
	ID        string `json:"id"`
	Cluster   string `json:"cluster"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
}

var (
	// Microservices contains all microservices.
	Microservices []Microservice
	listerSet     controller.Listers
)

// Init initialization for microservices API.
func Init(listers controller.Listers) {
	listerSet = listers
	refreshMicroservices()
}

func refreshMicroservices() {

	// initialize microservices as an empty list to avoid json encoding "nil"
	Microservices = []Microservice{}

	for _, mbp := range *listerSet.ModelBindingPairs {
		// Find all HelidonApps across all managed clusters within this model/binding pair
		for _, mc := range mbp.ManagedClusters {
			for _, helidonApp := range mc.HelidonApps {
				Microservices = append(Microservices, Microservice{
					ID:        helidonApp.Spec.Name,
					Name:      helidonApp.Spec.Name,
					Cluster:   mc.Name,
					Type:      "Helidon",
					Namespace: helidonApp.Spec.Namespace,
					Status:    helidonApp.Status.State,
				})
			}
		}
	}
}

// ReturnAllMicroservices returns all microservices used by model and bindings.
func ReturnAllMicroservices(w http.ResponseWriter, r *http.Request) {
	refreshMicroservices()
	glog.V(4).Info("GET /microservices")
	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Microservices)), 10))
	json.NewEncoder(w).Encode(Microservices)
}

// ReturnSingleMicroservice returns a single microservice identified by the microservice Kubernetes UID.
func ReturnSingleMicroservice(w http.ResponseWriter, r *http.Request) {
	refreshMicroservices()
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /microservices/" + key)

	foundApplication := false
	for _, microservices := range Microservices {
		if microservices.ID == key {
			foundApplication = true
			json.NewEncoder(w).Encode(microservices)
		}
	}

	if !foundApplication {
		msg := fmt.Sprintf("Microservice with ID %v not found", key)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
}
