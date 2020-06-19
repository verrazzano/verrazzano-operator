// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package microservices

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

type Microservice struct {
	Id        string `json:"id"`
	Cluster   string `json:"cluster"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
}

var (
	Microservices []Microservice
	listerSet     controller.Listers
)

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
					Id:        helidonApp.Spec.Name,
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

func ReturnAllMicroservices(w http.ResponseWriter, r *http.Request) {
	refreshMicroservices()
	glog.V(4).Info("GET /microservices")
	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Microservices)), 10))
	json.NewEncoder(w).Encode(Microservices)
}

func ReturnSingleMicroservice(w http.ResponseWriter, r *http.Request) {
	refreshMicroservices()
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /microservices/" + key)

	for _, microservices := range Microservices {
		if microservices.Id == key {
			json.NewEncoder(w).Encode(microservices)
		}
	}
}
