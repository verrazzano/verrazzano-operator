package grids

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"net/http"
	"net/http/httptest"
	"testing"
)

var modelBindingPairs = map[string]*types.ModelBindingPair{
	"test-pair-1": testutil.GetModelBindingPair(),
}


//  TestReturnAllGrids verifies that the single test-coherence grid entry is returned and valid
func TestReturnAllGrids(t *testing.T) {
	clusters := testutil.GetTestClusters()
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		listers controller.Listers
	}{
		{
			name: "verifyReturnAllGrids",
			listers: testutilcontroller.NewControllerListers(clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var myGrids     []Grid
			Init(tt.listers)
			resp := httptest.NewRecorder()
			ReturnAllGrids(resp, httptest.NewRequest("GET", "/grids", nil))
			assert.Equal(t, http.StatusOK, resp.Code , "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myGrids)
			assert.Len(t, myGrids, 1 , "expect returned Grid Array to have 1 entry")
			assert.Equal(t, "test-coherence", myGrids[0].Name , "expect Grid Name to be test-coherence")
			assert.Equal(t, "test-coherence-image:latest", myGrids[0].Image , "expect Grid Image Name to be test-coherence-image:latest")
		})
	}
}

//  TestReturnSingleGrid verifies that the grid entry with ID=0 is returned and valid
func TestReturnSingleGrid(t *testing.T) {
	clusters := testutil.GetTestClusters()
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		listers controller.Listers
	}{
		{
			name: "verifyReturnSingleGrid",
			listers: testutilcontroller.NewControllerListers(clusters, &modelBindingPairs),
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
			assert.Equal(t, http.StatusOK, resp.Code , "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myGrid)
			assert.Equal(t, "0", myGrid.ID , "expect the Grid with ID = 0 to be successfully returned")
			assert.Equal(t, "test-coherence", myGrid.Name , "expect Grid Name to be test-coherence")
			assert.Equal(t, "test-coherence-image:latest", myGrid.Image , "expect Grid Image Name to be test-coherence-image:latest")
		})
	}
}


//  TestReturnSingleGridNotFpund verifies that the grid entry with ID=100 is not found
func TestReturnSingleGridNotFound(t *testing.T) {
	clusters := testutil.GetTestClusters()
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		listers controller.Listers
	}{
		{
			name: "verifyReturnSingleGridNotFound",
			listers: testutilcontroller.NewControllerListers(clusters, &modelBindingPairs),
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
			assert.Equal(t, http.StatusOK, resp.Code , "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myGrid)
			assert.Equal(t, "", myGrid.ID , "expect no Grid to be found for ID 100")
		})
	}
}

