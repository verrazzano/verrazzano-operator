// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/verrazzano/verrazzano-operator/pkg/api/instance"
)

var allowedHTTPMethods = []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"}
var allowedHeaders = []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"}

//TestEnableCors tests that the EnableCors method adds the expected Access-Control-Allow-Origin header to the response
func TestEnableCors(t *testing.T) {
	type args struct {
		verrazzanoURI string
		writer        http.ResponseWriter
	}
	tests := []struct {
		name                string
		origin              string
		expectedAllowOrigin string
		args                args
	}{
		{name: "Verrazzano URI is set",
			origin:              "https://console.somesuffix.xip.io",
			expectedAllowOrigin: "https://console.somesuffix.xip.io",
			args:                args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{name: "Without Verrazzano URI set",
			origin:              "http://localhost",
			expectedAllowOrigin: "",
			args:                args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{name: "Without Verrazzano URI set and different port Origin",
			origin:              "http://localhost:8000",
			expectedAllowOrigin: "",
			args:                args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{name: "Origin does not match",
			origin:              "http://localhost",
			expectedAllowOrigin: "",
			args:                args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN the Verrazzano URI suffix is set
			instance.SetVerrazzanoURI(tt.args.verrazzanoURI)
			req, err := http.NewRequest("GET", "http://someUrl", nil)
			assert.Nil(t, err, "Error creating test request")
			req.Header.Add("origin", tt.origin)

			//WHEN EnableCors is called
			EnableCors(req, &tt.args.writer)

			//THEN it returns the expected Access-Control-Allow-Origin header value
			assert.Equal(t, tt.expectedAllowOrigin, tt.args.writer.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

//TestSetupOptionsResponse tests that the SetupOptionsResponse method adds the expected CORS related headers to the response
func TestSetupOptionsResponse(t *testing.T) {
	type args struct {
		verrazzanoURI string
		writer        http.ResponseWriter
	}
	tests := []struct {
		name                 string
		origin               string
		expectedAllowOrigin  string
		expectedAllowMethods []string
		expectedAllowHeaders []string
		args                 args
	}{
		{name: "With Verrazzano URI set",
			origin:               "https://console.somesuffix.xip.io",
			expectedAllowOrigin:  "https://console.somesuffix.xip.io",
			expectedAllowMethods: allowedHTTPMethods,
			expectedAllowHeaders: allowedHeaders,
			args:                 args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{name: "Without Verrazzano URI set",
			origin:               "",
			expectedAllowOrigin:  "",
			expectedAllowMethods: allowedHTTPMethods,
			expectedAllowHeaders: allowedHeaders,
			args:                 args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{name: "Origin does not match",
			origin:               "http://localhost",
			expectedAllowOrigin:  "",
			expectedAllowMethods: allowedHTTPMethods,
			expectedAllowHeaders: allowedHeaders,
			args:                 args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN the Verrazzano URI suffix is set
			instance.SetVerrazzanoURI(tt.args.verrazzanoURI)
			req, err := http.NewRequest("GET", "http://someUrl", nil)
			assert.Nil(t, err, "Error creating test request")
			req.Header.Add("Origin", tt.origin)

			//WHEN SetupOptionsResponse is called
			SetupOptionsResponse(&tt.args.writer, req)

			//THEN it writes the expected CORS header values
			assert.Equal(t, tt.expectedAllowOrigin, tt.args.writer.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, strings.Join(tt.expectedAllowMethods, ", "), tt.args.writer.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, strings.Join(tt.expectedAllowHeaders, ", "), tt.args.writer.Header().Get("Access-Control-Allow-Headers"))
		})
	}
}

func Test_getAllowedOrigin(t *testing.T) {
	vzURI := "something.v8o.oracledx.com"
	defaultExpectedValue := fmt.Sprintf("https://console.%s", vzURI)
	tests := []struct {
		name             string
		origin           string
		additionalOrigin string
		originAllowed    bool
	}{
		{name: "origin matches default",
			origin:           defaultExpectedValue,
			additionalOrigin: "",
			originAllowed:    true},

		{name: "origin matches default with additional origin",
			origin:           defaultExpectedValue,
			additionalOrigin: "http://localhost:8000",
			originAllowed:    true},

		{name: "origin matches additional origin",
			origin:           "http://localhost:8000",
			additionalOrigin: "http://localhost:8000",
			originAllowed:    true},

		{name: "negative test origin matches none",
			origin:           "http://somethingelse",
			additionalOrigin: "http://localhost:8000",
			originAllowed:    false},

		{name: "negative test origin does not match default",
			origin:           "http://somethingelse",
			additionalOrigin: "",
			originAllowed:    false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN the verrazzano URI suffix and an ACCESS_CONTROL_ALLOW_ORIGINS override env var
			instance.SetVerrazzanoURI(vzURI)
			os.Setenv("ACCESS_CONTROL_ALLOW_ORIGIN", tt.additionalOrigin)

			// WHEN getAllowedOrigin is called for a origin value
			actualOrigin := getAllowedOrigin(tt.origin)

			// THEN, if the origin is allowed, it should be returned in the allowed origin,
			// otherwise empty string should be returned
			if tt.originAllowed {
				assert.Equal(t, tt.origin, actualOrigin)
			} else {
				assert.Empty(t, actualOrigin)
			}
			os.Unsetenv("ACCESS_CONTROL_ALLOW_ORIGINS")
		})
	}
}
