// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jobs

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReturnAllJobs(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "verifyReturnAllJobs",
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var myJobs []Job

			Init()
			resp := httptest.NewRecorder()
			ReturnAllJobs(resp, httptest.NewRequest("GET", "/jobs", nil))
			assert.Equal(t, 200, resp.Code)
			json.NewDecoder(resp.Body).Decode(&myJobs)
			assert.Equal(t, 1, len(myJobs))
			assert.Equal(t, "1", myJobs[0].Id)
		})
	}
}

func TestReturnSingleJob(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "verifyReturnSingleJob",
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{
				"id": "1",
			}
			var myJob Job
			Init()
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/jobs", nil)
			req = mux.SetURLVars(req, vars)
			ReturnSingleJob(resp, req)
			assert.Equal(t, 200, resp.Code)
			json.NewDecoder(resp.Body).Decode(&myJob)
			assert.Equal(t, "1", myJob.Id)
		})
	}
}

func TestReturnSingleJobNotFound(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "verifyReturnSingleJobNotFound",
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{
				"id": "100",
			}
			var myJob Job
			Init()
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/jobs", nil)
			req = mux.SetURLVars(req, vars)
			ReturnSingleJob(resp, req)
			assert.Equal(t, 200, resp.Code)
			json.NewDecoder(resp.Body).Decode(&myJob)
			assert.Equal(t, "", myJob.Id)
		})
	}
}
