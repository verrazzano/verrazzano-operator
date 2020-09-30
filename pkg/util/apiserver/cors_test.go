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
		referer             string
		expectedAllowOrigin string
		args                args
	}{
		{"With Verrazzano URI set", "https://console.somesuffix.xip.io",
			"https://console.somesuffix.xip.io",
			args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},
		{"Without Verrazzano URI set", "http://localhost",
			"", args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},
		{"Referer does not match", "http://localhost",
			"", args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN the Verrazzano URI suffix is set
			instance.SetVerrazzanoURI(tt.args.verrazzanoURI)
			req, err := http.NewRequest("GET", "http://someUrl", nil)
			assert.Nil(t, err, "Error creating test request")
			req.Header.Add("Referer", tt.referer)
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
		referer              string
		expectedAllowOrigin  string
		expectedAllowMethods []string
		expectedAllowHeaders []string
		args                 args
	}{
		{"With Verrazzano URI set", "https://console.somesuffix.xip.io",
			"https://console.somesuffix.xip.io",
			allowedHTTPMethods, allowedHeaders,
			args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{"Without Verrazzano URI set", "", "",
			allowedHTTPMethods, allowedHeaders,
			args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},

		{"Referer does not match", "http://localhost", "",
			allowedHTTPMethods, allowedHeaders,
			args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN the Verrazzano URI suffix is set
			instance.SetVerrazzanoURI(tt.args.verrazzanoURI)
			req, err := http.NewRequest("GET", "http://someUrl", nil)
			assert.Nil(t, err, "Error creating test request")
			req.Header.Add("Referer", tt.referer)

			//WHEN SetupOptionsResponse is called
			SetupOptionsResponse(&tt.args.writer, req)

			//THEN it returns the expected CORS header values
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
		name              string
		referer           string
		additionalOrigins []string
		refererAllowed    bool
	}{
		{"origin matches default", defaultExpectedValue, []string{}, true},
		{"origin matches default with additional origin", defaultExpectedValue, []string{"http://localhost:8000"}, true},
		{"origin matches single additional origin", "http://localhost:8000", []string{"http://localhost:8000"}, true},
		{"origin matches one of additional origins", "http://localhost:8888", []string{"http://localhost:8000", "http://localhost:8888"}, true},
		{"origin matches default with multiple additional origins", defaultExpectedValue, []string{"http://localhost:8000", "http://localhost:8888"}, true},
		{"negative test origin matches none", "http://somethingelse", []string{"http://localhost:8000", "http://localhost:8888"}, false},
		{"negative test origin does not match default", "http://somethingelse", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN the verrazzano URI suffix and an ACCESS_CONTROL_ALLOW_ORIGINS override env var
			instance.SetVerrazzanoURI(vzURI)
			os.Setenv("ACCESS_CONTROL_ALLOW_ORIGINS", strings.Join(tt.additionalOrigins, ", "))

			// WHEN getAllowedOrigin is called for a referer value
			actualOrigin := getAllowedOrigin(tt.referer)

			// THEN, if the referer is allowed, it should be returned in the allowed origin,
			// otherwise empty string should be returned
			if tt.refererAllowed {
				assert.Equal(t, tt.referer, actualOrigin)
			} else {
				assert.Empty(t, actualOrigin)
			}
			os.Unsetenv("ACCESS_CONTROL_ALLOW_ORIGINS")
		})
	}
}
