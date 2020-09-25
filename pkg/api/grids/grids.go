// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grids

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

// Grid details of grid returned in API calls.
type Grid struct {
	ID        string `json:"id"`
	Cluster   string `json:"cluster"`
	Name      string `json:"name"`
	PodName   string `json:"podName"`
	Role      string `json:"role"`
	Image     string `json:"image"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

var (
	// Grids contains all grids.
	Grids     []Grid
	listerSet controller.Listers
)

// Init initialization for grids API.
func Init(listers controller.Listers) {
	listerSet = listers
	refreshGrids()
}

func refreshGrids() {

	// initialize grids as an empty list to avoid json encoding "nil"
	Grids = []Grid{}

	for _, mbp := range *listerSet.ModelBindingPairs {
		// grab all of the grids
		i := 0
		for _, grid := range mbp.Model.Spec.CoherenceClusters {
			Grids = append(Grids, Grid{
				ID: strconv.FormatInt(int64(i), 10),
				Cluster: func() string {
					return "NYI"
				}(),
				PodName:   "NYI",
				Name:      grid.Name,
				Role:      "NYI",
				Image:     grid.Image,
				Namespace: "NYI",
				Status: func() string {
					return "NYI"
				}(),
			})
			i++
		}
	}
}

// ReturnAllGrids returns all grids used by model and bindings.
func ReturnAllGrids(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /grids")

	refreshGrids()
	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Grids)), 10))
	json.NewEncoder(w).Encode(Grids)
}

// ReturnSingleGrid returns a single grid identified by the grid Kubernetes UID.
func ReturnSingleGrid(w http.ResponseWriter, r *http.Request) {
	refreshGrids()
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /grids/" + key)

	for _, grids := range Grids {
		if grids.ID == key {
			json.NewEncoder(w).Encode(grids)
		}
	}
}
