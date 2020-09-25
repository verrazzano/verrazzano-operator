// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package images

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

// a set of test images for testing purposes
var testImages []Image = []Image{
	{
		ID:              "1",
		Format:          "docker",
		WebLogicVersion: "12.2.1.3.0",
		JDKVersion:      "8u161",
		WebLogicPSU:     "OCT PSU",
		Patches:         "28186730,28298734,28076014",
		Name:            "container-registry.oracle.com/weblogicdev/my-great-domain",
		Tag:             "1.0",
		Description:     "WebLogic image with my domain in it",
		Status:          "PROVISIONING",
	},
	{
		ID:              "2",
		Format:          "docker",
		WebLogicVersion: "12.2.1.3.1",
		JDKVersion:      "8u161",
		WebLogicPSU:     "OCT PSU",
		Patches:         "28186730,28298734,28076014",
		Name:            "container-registry.oracle.com/weblogicdev/my-great-domain-2",
		Tag:             "1.0",
		Description:     "WebLogic image with my domain in it",
		Status:          "PROVISIONING",
	}}

// simple test to exercise the one line in the actual Init method
func TestInit(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "initTest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init()
			assert.Equal(t, []Image{}, Images)
		})
	}
}

func TestReturnNoImages(t *testing.T) {
	assert := assert.New(t)
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		ReturnAllImages(w, r)
	})
	request, _ := http.NewRequest("GET", "/images", nil)
	router.ServeHTTP(recorder, request)
	assert.Equal(200, recorder.Code)
	var imgs []Image
	err := json.Unmarshal(recorder.Body.Bytes(), &imgs)
	if err != nil {
		t.Errorf("Error while reading response JSON: %s", err)
	}
	assert.Empty(imgs)
}

func TestReturnAllImages(t *testing.T) {
	Images = append(Images, testImages...)
	assert := assert.New(t)
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		ReturnAllImages(w, r)
	})
	request, _ := http.NewRequest("GET", "/images", nil)
	router.ServeHTTP(recorder, request)
	assert.Equal(200, recorder.Code)
	var imgs []Image
	err := json.Unmarshal(recorder.Body.Bytes(), &imgs)
	if err != nil {
		t.Errorf("Error while reading response JSON: %s", err)
	}
	img := imgs[0]
	assert.Equal("1", img.ID)
	assert.Equal("12.2.1.3.0", img.WebLogicVersion)

	img = imgs[1]
	assert.Equal("2", img.ID)
	assert.Equal("12.2.1.3.1", img.WebLogicVersion)
}

func TestReturnSingleImage(t *testing.T) {
	Images = nil
	Images = append(Images, testImages...)
	assert := assert.New(t)
	recorder, router := createRequestHandlers()
	request, _ := http.NewRequest("GET", "/images/1", nil)
	router.ServeHTTP(recorder, request)
	testSingleImageResponse(t, assert, recorder, "12.2.1.3.0")

	recorder.Flush()
	recorder, router = createRequestHandlers()
	request, _ = http.NewRequest("GET", "/images/2", nil)
	router.ServeHTTP(recorder, request)
	testSingleImageResponse(t, assert, recorder, "12.2.1.3.1")

	recorder.Flush()
	recorder, router = createRequestHandlers()
	request, _ = http.NewRequest("GET", "/images/3", nil)
	router.ServeHTTP(recorder, request)
	assert.Equal(http.StatusNotFound, recorder.Code)
}

// Create the structs required for generating and posting a request
func createRequestHandlers() (*httptest.ResponseRecorder, *mux.Router) {
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}", func(w http.ResponseWriter, r *http.Request) {
		ReturnSingleImage(w, r)
	})
	return recorder, router
}

// validate a single image response by ensuring the version value is correct
func testSingleImageResponse(t *testing.T, assert *assert.Assertions, recorder *httptest.ResponseRecorder, version string) {
	assert.Equal(http.StatusOK, recorder.Code)
	var img Image
	err := json.Unmarshal(recorder.Body.Bytes(), &img)
	if err != nil {
		t.Errorf("Error while reading response JSON: %s", err)
	}
	assert.Equal(version, img.WebLogicVersion)
}
