// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package microservices

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
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

// TestReturnAllMicroservices tests that all Microservices are returned
// GIVEN that Microservices are configured
// WHEN I call GetAllMicroservices
// THEN all the Microservices should be returned
// AND that the Microservices returned should be the correct amount and have valid values
func TestReturnAllMicroservices(t *testing.T) {
	var modelBindingPairs = map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{
			name:    "verifyReturnAllMicroservices",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			var myMicroservices []Microservice
			Init(tt.listers)
			resp := httptest.NewRecorder()
			ReturnAllMicroservices(resp, httptest.NewRequest("GET", "/microservices", nil))
			assert.Equal(http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myMicroservices)
			assert.Len(myMicroservices, 2, "expect returned MicroServices Array to have 2 entries")
			assert.Equal("HelidonApp1", myMicroservices[0].Name, "expect Helidon App Name to be HelidonApp1")
			assert.Equal("cluster1", myMicroservices[0].Cluster, "expect Helidon App Cluster Name to be cluster1")
			assert.Equal("defaultOnMC1", myMicroservices[0].Namespace, "expect Helidon App Namespace to be defaultOnMC1")
			assert.Equal("HelidonApp2", myMicroservices[1].Name, "expect Helidon App Name to be HelidonApp2")
			assert.Equal("cluster2", myMicroservices[1].Cluster, "expect Helidon App Cluster Name to be cluster2")
			assert.Equal("defaultOnMC2", myMicroservices[1].Namespace, "expect Helidon App Namespace to be defaultOnMC2")
		})
	}
}

// TestReturnSingleMicroservice tests that the Microservice with ID=HelidonApp1 is returned
// GIVEN that Microservice with ID=HelidonApp1 exists
// WHEN I call ReturnSingleMicroservice
// THEN the Microservice with ID=HelidonApp1 should be returned
// AND that the Microservice returned should have the correct ID and have valid values
func TestReturnSingleMicroservice(t *testing.T) {
	var modelBindingPairs = map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{
			name:    "verifyReturnSingleMicroservice",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{
				"id": "HelidonApp1",
			}
			var myMicroservice Microservice

			assert := assert.New(t)

			Init(tt.listers)
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/microservices", nil)
			req = mux.SetURLVars(req, vars)
			ReturnSingleMicroservice(resp, req)
			assert.Equal(http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myMicroservice)
			assert.Equal("HelidonApp1", myMicroservice.ID, "expect the Microservice with ID = HelidonApp1 to be successfully returned")
			assert.Equal("HelidonApp1", myMicroservice.Name, "expect Microservice Name to be HelidonApp1")
			assert.Equal("cluster1", myMicroservice.Cluster, "expect Helidon App Cluster Name to be cluster1")
			assert.Equal("defaultOnMC1", myMicroservice.Namespace, "expect Helidon App Namespace to be defaultOnMC1")
		})
	}
}

// TestReturnSingleMicroserviceNotFound tests that a Microservice that doesn't exist is not found
// GIVEN that Microservice with ID=HelidonApp100 doesn't exist
// WHEN I call ReturnSingleMicroservice
// THEN no Microservice should be returned
// AND that the MicroService returned should be empty
func TestReturnSingleMicroserviceNotFound(t *testing.T) {
	var modelBindingPairs = map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{
			name:    "verifyReturnSingleMicroserviceNotFound",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{
				"id": "HelidonApp100",
			}
			var myMicroservice Microservice

			assert := assert.New(t)

			Init(tt.listers)
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/microservices", nil)
			req = mux.SetURLVars(req, vars)
			ReturnSingleMicroservice(resp, req)
			assert.Equal(http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myMicroservice)
			assert.Empty(myMicroservice.ID, "expect the Microservice with ID = HelidonApp100 to be Not Found")
			assert.Empty(myMicroservice.Name, "expect Microservice Name to be empty")
			assert.Empty(myMicroservice.Cluster, "expect Helidon App Cluster Name to be empty")
			assert.Empty(myMicroservice.Namespace, "expect Helidon App Namespace to be empty")
			assert.Empty(myMicroservice.Namespace, "expect Helidon App Namespace to be empty")
		})
	}
}
