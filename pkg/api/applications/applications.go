// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package applications

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	v1beta1 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/labels"
)

// Application represents an application in Verrazzano
type Application struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model"`
	Binding     string `json:"binding"`
	Status      string `json:"status"`
}

// ModelFinder keeps track of matching models.
type ModelFinder struct {
	Model        *v1beta1.VerrazzanoModel
	BindingMatch bool
}

// Applications is the collection of applications
var (
	Applications []Application
	listerSet    controller.Listers
)

// Init sets up sample data for testing purposes
func Init(listers controller.Listers) {
	listerSet = listers
	refreshApplications()
}

func refreshApplications() error {

	bindingSelector := labels.SelectorFromSet(map[string]string{})
	bindings, err := (*listerSet.BindingLister).VerrazzanoBindings("default").List(bindingSelector)
	if err != nil {
		glog.Errorf("Error getting application bindings: %s", err.Error())
		return err
	}

	modelSelector := labels.SelectorFromSet(map[string]string{})
	models, err := (*listerSet.ModelLister).VerrazzanoModels("default").List(modelSelector)
	if err != nil {
		glog.Errorf("Error getting application models: %s", err.Error())
		return err
	}

	modelMap := make(map[string]*ModelFinder)
	for _, model := range models {
		modelMap[model.Name] = &ModelFinder{
			Model:        model,
			BindingMatch: false,
		}
	}

	Applications = []Application{}
	i := 0
	for _, binding := range bindings {
		b, _ := yaml.Marshal(binding)
		modelYaml := []byte{}
		if model, ok := modelMap[binding.Spec.ModelName]; ok {
			modelYaml, _ = yaml.Marshal(model.Model)
			model.BindingMatch = true
		}
		Applications = append(Applications, Application{
			ID:          strconv.Itoa(i),
			Name:        binding.Name,
			Description: binding.Spec.Description,
			Model:       string(modelYaml),
			Binding:     string(b),
			Status:      "NYI",
		})
		i++
	}

	// Add any models that do not have a matching binding.  Any details about the binding will
	// be empty.
	for _, model := range modelMap {
		if model.BindingMatch == false {
			modelYaml, _ := yaml.Marshal(model.Model)
			Applications = append(Applications, Application{
				ID:          strconv.Itoa(i),
				Name:        "",
				Description: "",
				Model:       string(modelYaml),
				Binding:     "",
				Status:      "NYI",
			})
			i++
		}
	}

	return nil
}

// ReturnAllApplications returns a list of all of the applications
func ReturnAllApplications(w http.ResponseWriter, r *http.Request) {
	err := refreshApplications()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting applications: %s", err.Error()),
			http.StatusInternalServerError)
		return
	}
	glog.V(4).Info("GET /applications")

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Applications)), 10))
	json.NewEncoder(w).Encode(Applications)
}

// ReturnSingleApplication returns a single application
func ReturnSingleApplication(w http.ResponseWriter, r *http.Request) {
	err := refreshApplications()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting applications: %s", err.Error()),
			http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	key := vars["id"]
	glog.V(4).Info("GET /applications/" + key)

	// Loop over all of our Applications
	// if the article.Id equals the key we pass in
	// return the article encoded as JSON
	foundApplication := false
	for _, application := range Applications {
		if application.ID == key {
			foundApplication = true
			json.NewEncoder(w).Encode(application)
		}
	}
	if !foundApplication {
		msg := fmt.Sprintf("Application with ID %v not found", key)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
}

// CreateNewApplication creates a new application
func CreateNewApplication(w http.ResponseWriter, r *http.Request) {
	err := refreshApplications()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting applications: %s", err.Error()),
			http.StatusInternalServerError)
		return
	}
	glog.V(4).Info("POST /applications")

	// get the body of our POST request
	// unmarshal this into a new Application struct
	// append this to our Applications array.
	reqBody, _ := ioutil.ReadAll(r.Body)
	var application Application
	err = json.Unmarshal(reqBody, &application)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing application JSON: %s", err.Error()),
			http.StatusBadRequest)
		return
	}
	// update our global Applications array to include
	// our new Application
	Applications = append(Applications, application)

	json.NewEncoder(w).Encode(application)
}

// DeleteApplication deletes an application
func DeleteApplication(w http.ResponseWriter, r *http.Request) {
	err := refreshApplications()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting applications: %s", err.Error()),
			http.StatusInternalServerError)
		return
	}

	// once again, we will need to parse the path parameters
	vars := mux.Vars(r)
	// we will need to extract the `id` of the application we
	// wish to delete
	ID := vars["id"]
	glog.V(4).Info("DELETE /applications/" + ID)

	foundApplication := false
	// we then need to loop through all our applications
	for index, application := range Applications {
		// if our id path parameter matches one of our
		// applications
		if application.ID == ID {
			foundApplication = true
			// updates our Applications array to remove the
			// application
			Applications = append(Applications[:index], Applications[index+1:]...)
		}
	}
	if !foundApplication {
		msg := fmt.Sprintf("Application with ID %v not found", ID)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
}

// UpdateApplication updates an existing application
func UpdateApplication(w http.ResponseWriter, r *http.Request) {
	err := refreshApplications()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting applications: %s", err.Error()),
			http.StatusInternalServerError)
		return
	}

	// find the id of the application to update
	vars := mux.Vars(r)
	ID := vars["id"]

	glog.V(4).Info("PUT /applications/" + ID)

	// get the updated application
	reqBody, _ := ioutil.ReadAll(r.Body)
	var updatedApplication Application
	err = json.Unmarshal(reqBody, &updatedApplication)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing application JSON: %s", err.Error()),
			http.StatusBadRequest)
		return
	}
	// find the right application and update it
	foundApplication := false
	for index, application := range Applications {
		if application.ID == ID {
			foundApplication = true
			Applications = append(Applications[:index], updatedApplication)
			Applications = append(Applications, Applications[index+1:]...)
		}
	}
	if !foundApplication {
		msg := fmt.Sprintf("Application with ID %v not found", ID)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
}
