// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package images

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

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

var Images []Image

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

func ReturnAllImages(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /images")

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Images)), 10))
	json.NewEncoder(w).Encode(Images)
}

func ReturnSingleImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /images/" + key)

	for _, images := range Images {
		if images.ID == key {
			json.NewEncoder(w).Encode(images)
		}
	}
}
