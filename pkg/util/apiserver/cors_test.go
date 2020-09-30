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
		expectedAllowOrigin string
		args                args
	}{
		{"With Verrazzano URI set", "https://console.somesuffix.xip.io", args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},
		{"Without Verrazzano URI set", "", args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN the Verrazzano URI suffix is set
			instance.SetVerrazzanoURI(tt.args.verrazzanoURI)

			//WHEN EnableCors is called
			EnableCors(&tt.args.writer)

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
		expectedAllowOrigin  string
		expectedAllowMethods []string
		expectedAllowHeaders []string
		args                 args
	}{
		{"With Verrazzano URI set", "https://console.somesuffix.xip.io", allowedHTTPMethods, allowedHeaders, args{verrazzanoURI: "somesuffix.xip.io", writer: http.ResponseWriter(httptest.NewRecorder())}},
		{"Without Verrazzano URI set", "", allowedHTTPMethods, allowedHeaders, args{verrazzanoURI: "", writer: http.ResponseWriter(httptest.NewRecorder())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//GIVEN the Verrazzano URI suffix is set
			instance.SetVerrazzanoURI(tt.args.verrazzanoURI)

			//WHEN SetupOptionsResponse is called
			SetupOptionsResponse(&tt.args.writer, nil)

			//THEN it returns the expected CORS header values
			assert.Equal(t, tt.expectedAllowOrigin, tt.args.writer.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, strings.Join(tt.expectedAllowMethods, ", "), tt.args.writer.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, strings.Join(tt.expectedAllowHeaders, ", "), tt.args.writer.Header().Get("Access-Control-Allow-Headers"))
		})
	}
}

func Test_getAllowedOrigins(t *testing.T) {
	vzURI := "something.v8o.oracledx.com"
	defaultExpectedValue := fmt.Sprintf("https://console.%s", vzURI)
	tests := []struct {
		name              string
		additionalOrigins []string
	}{
		{"no additional origins", []string{}},
		{"one additional origin", []string{"http://localhost:8000"}},
		{"multiple additional origins", []string{"http://localhost:8000", "http://localhost:8888"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance.SetVerrazzanoURI(vzURI)
			expectedOrigins := []string{defaultExpectedValue}
			if len(tt.additionalOrigins) != 0 {
				expectedOrigins = append(expectedOrigins, tt.additionalOrigins...)
			}
			os.Setenv("ACCESS_CONTROL_ALLOW_ORIGINS", strings.Join(tt.additionalOrigins, ", "))
			strActualOrigins := getAllowedOrigins()
			actualOrigins := strings.Split(strActualOrigins, ", ")
			for _, expected := range expectedOrigins {
				assert.Contains(t, actualOrigins, expected)
			}
			os.Unsetenv("ACCESS_CONTROL_ALLOW_ORIGINS")
		})
	}
}
