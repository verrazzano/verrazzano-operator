// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operators

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

type Operator struct {
	Id          string `json:"id"`
	Cluster     string `json:"cluster"`
	Type        string `json:"type"`
	RestAddress string `json:"restAddress"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Status      string `json:"status"`
}

var (
	Operators []Operator
	listerSet controller.Listers
)

func Init(listers controller.Listers) {
	listerSet = listers
	refreshOperators()
}

func refreshOperators() {
	// initialize operators as an empty list to avoid json encoding "nil"
	Operators = []Operator{}

	for _, mbp := range *listerSet.ModelBindingPairs {

		// Find all the WebLogic operators for this model/binding pair.  Need to look at each managed cluster
		for _, mc := range mbp.ManagedClusters {
			// Each managed cluster can only have one WebLogic operator
			if mc.WlsOperator != nil {
				Operators = append(Operators, Operator{
					Name:    mc.WlsOperator.Spec.Name,
					Id:      string(mc.WlsOperator.UID),
					Cluster: mc.Name,
					Type:    "WebLogic",
					RestAddress: func() string {
						return "http://1.2.3.4:8080"
					}(),
					Namespace: mc.WlsOperator.Spec.Namespace,
					Status:    mc.WlsOperator.Status.State,
				},
				)
			}
		}
	}
}

func ReturnAllOperators(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /operators")

	refreshOperators()
	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Operators)), 10))
	json.NewEncoder(w).Encode(Operators)
}

func ReturnSingleOperator(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /operators/" + key)

	for _, operator := range Operators {
		if operator.Id == key {
			json.NewEncoder(w).Encode(operator)
		}
	}
}
