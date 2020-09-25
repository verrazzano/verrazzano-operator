// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package applications

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"
	yaml "gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"

	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
)

var modelBindingPairs = map[string]*types.ModelBindingPair{
	"test-pair-1": testutil.GetModelBindingPair(),
}

func TestInit(t *testing.T) {
	clusters := testutil.GetTestClusters()
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "usingFakeListers", listers: testutilcontroller.NewControllerListers(clusters, &modelBindingPairs)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.listers)
			assert.Equal(t, tt.listers, listerSet)
			assert.Equal(t, 1, len(Applications))
			assert.Equal(t, "testBinding", Applications[0].Name)
			assertApplicationOk(t, Applications[0], modelBindingPairs["test-pair-1"])
		})
	}
}

func TestReturnAllApplications(t *testing.T) {
	Init(testutilcontroller.NewControllerListers(testutil.GetTestClusters(), &modelBindingPairs))
	request, _ := http.NewRequest("GET", "/applications", nil)
	responseRecorder := testutil.InvokeHTTPHandler(request, "/applications", ReturnAllApplications)
	assert.Equal(t, 200, responseRecorder.Code)
	responseApplications := []Application{}
	err := yaml.Unmarshal(responseRecorder.Body.Bytes(), &responseApplications)
	assert.Nil(t, err, "Could not unmarshall response")
	assert.Equal(t, 1, len(responseApplications))
	assertApplicationOk(t, responseApplications[0], modelBindingPairs["test-pair-1"])
}

func TestReturnSingleApplication(t *testing.T) {
	Init(testutilcontroller.NewControllerListers(testutil.GetTestClusters(), &modelBindingPairs))
	request, _ := http.NewRequest("GET", "/applications/0", nil)
	responseRecorder := testutil.InvokeHTTPHandler(request, "/applications/{id}", ReturnSingleApplication)
	assert.Equal(t, 200, responseRecorder.Code)
	responseApplication := Application{}
	err := yaml.Unmarshal(responseRecorder.Body.Bytes(), &responseApplication)
	assert.Nil(t, err, "Could not unmarshall response")
	assertApplicationOk(t, responseApplication, modelBindingPairs["test-pair-1"])
}

func TestReturnSingleApplicationNonExistent(t *testing.T) {
	Init(testutilcontroller.NewControllerListers(testutil.GetTestClusters(), &modelBindingPairs))
	request, _ := http.NewRequest("GET", "/applications/5", nil)
	responseRecorder := testutil.InvokeHTTPHandler(request, "/applications/{id}", ReturnSingleApplication)
	assert.Equal(t, 404, responseRecorder.Code)
	assert.Contains(t, string(responseRecorder.Body.Bytes()), "Application with ID 5 not found")
}

func TestCreateNewApplication(t *testing.T) {
	Init(testutilcontroller.NewControllerListers(testutil.GetTestClusters(), &modelBindingPairs))
	noModelApp := Application{ID: "91", Name: "NoModelApp", Status: "Broken",
		Description: "I have no model", Binding: Applications[0].Binding}
	noBindingApp := Application{ID: "92", Name: "NoBindingApp", Status: "AlsoBroken",
		Description: "I have no binding", Model: Applications[0].Model}
	notAnApplication := "this is some random text that can't be unmarshaled"

	validAppBytes, err := json.Marshal(Applications[0])
	assert.Nil(t, err, "Unexpected error - cannot marshal valid test application to JSON")
	noModelAppBytes, err := json.Marshal(noModelApp)
	assert.Nil(t, err, "Unexpected error - cannot marshal noModel test application to JSON")
	noBindingAppBytes, err := json.Marshal(noBindingApp)
	assert.Nil(t, err, "Unexpected error - cannot marshal noBinding test application to JSON")

	// Create http requests
	validAppRequest, err := http.NewRequest("POST", "/applications", bytes.NewBuffer(validAppBytes))
	assert.Nil(t, err, "Unexpected error - cannot create http request for valid test application")
	noModelAppRequest, err := http.NewRequest("POST", "/applications", bytes.NewBuffer(noModelAppBytes))
	assert.Nil(t, err, "Unexpected error - cannot create http request for noModel test application")
	noBindingAppRequest, err := http.NewRequest("POST", "/applications", bytes.NewBuffer(noBindingAppBytes))
	assert.Nil(t, err, "Unexpected error - cannot create http request for noBinding test application")
	notAnAppRequest, err := http.NewRequest("POST", "/applications", bytes.NewBuffer([]byte(notAnApplication)))
	assert.Nil(t, err, "Unexpected error - cannot create http request for invalid application")
	tests := []struct {
		name           string
		req            *http.Request
		expectedStatus int
	}{
		{"validApplication", validAppRequest, 200}, // no failures if model or binding are missing
		{"noModel", noModelAppRequest, 200},
		{"noBinding", noBindingAppRequest, 200},
		{"notAnApplication", notAnAppRequest, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseRecorder := testutil.InvokeHTTPHandler(tt.req, "/applications", CreateNewApplication)
			assert.Equal(t, tt.expectedStatus, responseRecorder.Code)
		})
	}
}

func TestDeleteApplication(t *testing.T) {
	Init(testutilcontroller.NewControllerListers(testutil.GetTestClusters(), &modelBindingPairs))
	nonExistingAppReq, err := http.NewRequest("DELETE", "/applications/22", nil)
	assert.Nil(t, err, "Unexpected error - failed creating delete request")
	existingAppReq, err := http.NewRequest("DELETE", "/applications/0", nil)
	assert.Nil(t, err, "Unexpected error - failed creating delete request")
	tests := []struct {
		name           string
		req            *http.Request
		expectedStatus int
	}{
		{"deleteNonexistentApplication", nonExistingAppReq, 404},
		{"deleteExistingApplication", existingAppReq, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseRecorder := testutil.InvokeHTTPHandler(tt.req, "/applications/{id}", DeleteApplication)
			assert.Equal(t, tt.expectedStatus, responseRecorder.Code)
		})
	}
}

func TestUpdateApplication(t *testing.T) {
	updateMbPairs := map[string]*types.ModelBindingPair{
		"mbPairGood0": testutil.GetModelBindingPairWithNames("goodModel", "goodBinding", "default"),
		//"mbPairGood1": testutil.GetModelBindingPairWithNames("tooGoodModel", "tooGoodBinding", "default"),
	}
	Init(testutilcontroller.NewControllerListers(testutil.GetTestClusters(), &updateMbPairs))
	goodApp := Application{ID: "0", Name: "StillAGoodApp", Status: "Good",
		Description: "What a good app am I!",
		Model:       Applications[0].Model,
		Binding:     Applications[0].Binding}
	notAnApp := "I'm not an app at all"

	goodAppBytes, err := json.Marshal(goodApp)
	assert.Nil(t, err, "Unexpected error - cannot marshal valid test application to JSON")

	// Create http requests
	goodAppRequest, err := http.NewRequest("PUT", "/applications/0", bytes.NewBuffer(goodAppBytes))
	assert.Nil(t, err, "Unexpected error - cannot create http request for valid test application")
	nonExistentAppRequest, err := http.NewRequest("PUT", "/applications/88", bytes.NewBuffer(goodAppBytes))
	assert.Nil(t, err, "Unexpected error - cannot create http request for nonexistent test application")
	notAnAppRequest, err := http.NewRequest("PUT", "/applications/0", bytes.NewBuffer([]byte(notAnApp)))
	assert.Nil(t, err, "Unexpected error - cannot create http request for invalid test application")

	tests := []struct {
		name           string
		req            *http.Request
		expectedStatus int
	}{
		{"goodAppUpdate", goodAppRequest, 200},
		{"nonExistentApp", nonExistentAppRequest, 404},
		{"notAnApplication", notAnAppRequest, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseRecorder := testutil.InvokeHTTPHandler(tt.req, "/applications/{id}", UpdateApplication)
			assert.Equal(t, tt.expectedStatus, responseRecorder.Code)
		})
	}
}

func assertApplicationOk(t *testing.T, application Application, mbPair *types.ModelBindingPair) {
	bindingInApplication := v1beta1.VerrazzanoBinding{}
	modelInApplication := v1beta1.VerrazzanoModel{}
	err := yaml.Unmarshal([]byte(application.Binding), &bindingInApplication)
	assert.Nil(t, err, "Application does not contain valid binding!")

	err = yaml.Unmarshal([]byte(application.Model), &modelInApplication)
	assert.Nil(t, err, "Application does not contain valid Model!")

	// Marshal and unmarshal the in-memory expected model and binding
	// Otherwise there are differences between the actual/expected in terms of nil vs empty {}
	marshaledModel, err := yaml.Marshal(mbPair.Model)
	assert.Nil(t, err, "Unexpected error - cannot marshal in-memory model")
	marshaledBinding, err := yaml.Marshal(mbPair.Binding)
	assert.Nil(t, err, "Unexpected error - cannot marshal in-memory binding")
	expectedModel := v1beta1.VerrazzanoModel{}
	expectedBinding := v1beta1.VerrazzanoBinding{}
	err = yaml.Unmarshal(marshaledModel, &expectedModel)
	assert.Nil(t, err, "Unexpected error, cannot unmarshal in-memory marshaled model!")
	err = yaml.Unmarshal(marshaledBinding, &expectedBinding)
	assert.Nil(t, err, "Unexpected error, cannot unmarshal in-memory marshaled model!")

	assert.Equal(t, expectedBinding, bindingInApplication)
	assert.Equal(t, expectedModel, modelInApplication)
}
