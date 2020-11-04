// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package images

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

// Image is a structure providing the image metadata
type Image struct {
	ID              string `json:"id"`
	Format          string `json:"format"`
	WebLogicVersion string `json:"weblogic_version"`
	JDKVersion      string `json:"jdk_version"`
	WebLogicPSU     string `json:"weblogic_psu"`
	Patches         string `json:"patches"`
	Name            string `json:"name"`
	Tag             string `json:"tag"`
	Description     string `json:"description"`
	Status          string `json:"status"`
}

// Images is the available set of images
var Images []Image

// Init creates an empty slice of images
func Init() {
	Images = []Image{}
	//Images = []Image{
	//	Image{
	//		Id: "1",
	//		Format: "docker",
	//		WebLogicVersion: "12.2.1.3.0",
	//		JDKVersion: "8u161",
	//		WebLogicPSU: "OCT PSU",
	//		Patches: "28186730,28298734,28076014",
	//		Name: "container-registry.oracle.com/weblogicdev/my-great-domain",
	//		Tag: "1.0",
	//		Description: "WebLogic image with my domain in it",
	//		Status: "PROVISIONING",
	//	},
	//}
}

// ReturnAllImages returns all images available
func ReturnAllImages(w http.ResponseWriter, r *http.Request) {
	zap.S().Infow("GET /images")

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Images)), 10))
	json.NewEncoder(w).Encode(Images)
}

// ReturnSingleImage returns the image identified by the supplied key. If no image
// is found, sets the 404/NotFound HTTP status header
func ReturnSingleImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	zap.S().Infow("GET /images/" + key)

	found := false
	for _, images := range Images {
		if images.ID == key {
			found = true
			json.NewEncoder(w).Encode(images)
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
	}
}
