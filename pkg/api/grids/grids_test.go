// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grids

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"testing"
)

var modelBindingPairs = map[string]*types.ModelBindingPair{
	"test-pair-1": testutil.GetModelBindingPair(),
}

// TestReturnAllGrids tests that all Grids are returned
// GIVEN that Grids are configured
// WHEN I call GetAllGrids
// THEN all the Grids should be returned
// AND that the Grids returned should be the correct amount and have valid values
func TestReturnAllGrids(t *testing.T) {
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	clusters := testutil.GetTestClusters()
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{
			name:    "verifyReturnAllGrids",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var myGrids []Grid
			Init(tt.listers)
			resp := httptest.NewRecorder()
			ReturnAllGrids(resp, httptest.NewRequest("GET", "/grids", nil))
			assert.Equal(t, http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myGrids)
			assert.Len(t, myGrids, 1, "expect returned Grid Array to have 1 entry")
			assert.Equal(t, "test-coherence", myGrids[0].Name, "expect Grid Name to be test-coherence")
			assert.Equal(t, "test-coherence-image:latest", myGrids[0].Image, "expect Grid Image Name to be test-coherence-image:latest")
		})
	}
}

// TestReturnSingleGrid tests that the Grid with ID=0 is returned
// GIVEN that Grid with ID=0 exists
// WHEN I call ReturnSingleGrid
// THEN all the Grid with ID=0 should be returned
// AND that the Grid returned should have the correct ID and have valid values
func TestReturnSingleGrid(t *testing.T) {
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	clusters := testutil.GetTestClusters()
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{
			name:    "verifyReturnSingleGrid",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{
				"id": "0",
			}
			var myGrid Grid
			Init(tt.listers)
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/grids", nil)
			req = mux.SetURLVars(req, vars)
			ReturnSingleGrid(resp, req)
			assert.Equal(t, http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myGrid)
			assert.Equal(t, "0", myGrid.ID, "expect the Grid with ID = 0 to be successfully returned")
			assert.Equal(t, "test-coherence", myGrid.Name, "expect Grid Name to be test-coherence")
			assert.Equal(t, "test-coherence-image:latest", myGrid.Image, "expect Grid Image Name to be test-coherence-image:latest")
		})
	}
}

// TestReturnSingleGridNotFound tests that a Grid that doesn't exist is not found
// GIVEN that Grid with ID=100 doesn't exist
// WHEN I call ReturnSingleGrid
// THEN all no Grid should be returned
// AND that the Grid returned should be empty
func TestReturnSingleGridNotFound(t *testing.T) {
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	clusters := testutil.GetTestClusters()
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{
			name:    "verifyReturnSingleGridNotFound",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{
				"id": "100",
			}
			var myGrid Grid
			Init(tt.listers)
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/grids", nil)
			req = mux.SetURLVars(req, vars)
			ReturnSingleGrid(resp, req)
			assert.Equal(t, http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myGrid)
			assert.Equal(t, "", myGrid.ID, "expect no Grid to be found for ID 100")
		})
	}
}
