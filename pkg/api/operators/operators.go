// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operators

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

// Operator details of operator returned in API calls.
type Operator struct {
	ID          string `json:"id"`
	Cluster     string `json:"cluster"`
	Type        string `json:"type"`
	RestAddress string `json:"restAddress"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Status      string `json:"status"`
}

var (
	// Operators contains all operators.
	Operators []Operator
	listerSet controller.Listers
)

// Init initialization for operators API.
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
					ID:      string(mc.WlsOperator.UID),
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

// ReturnAllOperators returns all operators used by model and bindings.
func ReturnAllOperators(w http.ResponseWriter, r *http.Request) {
	// Create log instance for returning all operators
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Operators").Str("name", "Return").Logger()

	logger.Info().Msg("GET /operators")

	refreshOperators()
	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Operators)), 10))
	json.NewEncoder(w).Encode(Operators)
}

// ReturnSingleOperator returns a single operator identified by the operator Kubernetes UID.
func ReturnSingleOperator(w http.ResponseWriter, r *http.Request) {
	// Create log instance for returning single operator
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Operators").Str("name", "Return").Logger()

	vars := mux.Vars(r)
	key := vars["id"]

	logger.Info().Msg("GET /operators/" + key)

	var found bool
	for _, operator := range Operators {
		if operator.ID == key {
			json.NewEncoder(w).Encode(operator)
			found = true
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
	}
}
