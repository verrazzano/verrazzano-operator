// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jobs

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file

type Job struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Created  string   `json:"created"`
	Workflow Workflow `json:"workflow"`
}

type Workflow struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Steps   []Step `json:"steps"`
}

type Step struct {
	Description string `json:"description"`
	Tasks       []Task `json:"tasks"`
}

type Task struct {
	Task   string `json:"task"`
	Inputs Inputs `json:"inputs,omitempty"`
	Data   Data   `json:"data,omitempty"`
}

type Inputs struct {
	Namespace      string `json:"namespace,omitempty"`
	Name           string `json:"name,omitempty"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	SecretsName    string `json:"secretsName,omitempty"`
	Image          string `json:"image,omitempty"`
}

type Data struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

var Jobs []Job

func Init() {
	Jobs = []Job{
		{
			Id:      "1",
			Name:    "Deploy WebLogic Operator",
			Status:  "Not started",
			Created: "2019-01-01 08:00:00",
			Workflow: Workflow{
				Name:    "job-0001",
				Version: "1.0.0",
				Steps: []Step{
					{
						Description: "Deploy WebLogic Operator",
						Tasks: []Task{
							{
								Task: "createK8sNamespace",
								Inputs: Inputs{
									Namespace: "operator1",
								},
							},
							{
								Task: "deployWlsOperator",
								Inputs: Inputs{
									Name:           "operator1",
									Namespace:      "operator1",
									ServiceAccount: "operator1",
									Image:          util.GetWeblogicOperatorImage(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func ReturnAllJobs(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /jobs")

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Jobs)), 10))
	json.NewEncoder(w).Encode(Jobs)
}

func ReturnSingleJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /jobs/" + key)

	for _, jobs := range Jobs {
		if jobs.Id == key {
			json.NewEncoder(w).Encode(jobs)
		}
	}
}
