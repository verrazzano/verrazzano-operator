// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jobs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, http.StatusOK, resp.Code)
			json.NewDecoder(resp.Body).Decode(&myJobs)
			assert.Len(t, myJobs, 1)
			assert.Equal(t, "1", myJobs[0].ID)
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
			assert.Equal(t, http.StatusOK, resp.Code)
			json.NewDecoder(resp.Body).Decode(&myJob)
			assert.Equal(t, "1", myJob.ID)
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
			assert.Equal(t, http.StatusOK, resp.Code)
			json.NewDecoder(resp.Body).Decode(&myJob)
			assert.Equal(t, "", myJob.ID)
		})
	}
}
