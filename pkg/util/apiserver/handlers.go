// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/rs/zerolog"
)

// CORSHandler is an HTTP handler that will handle CORS preflight requests
// or delegate to the actual handler on real requests
func CORSHandler(h http.Handler) http.Handler {
	// Create log instance for initializing flags
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "CORSHandler").Str("name", "Handle").Logger()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		EnableCors(r, &w)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			// CORS preflight request
			SetupOptionsResponse(&w, r)
			logger.Info().Msg("OPTIONS /" + r.URL.Path)
			return
		}
		// actual request
		h.ServeHTTP(w, r)
	})

}

// Error represents an error structure for HTTP errors.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPError replies to the request with the specified error message and HTTP code.
func HTTPError(w http.ResponseWriter, statusCode int, code, message string) {
	error := Error{Code: code, Message: message}
	errMsg := message
	bytes, err := json.Marshal(&error)
	if err == nil || (bytes != nil && len(bytes) > 0) {
		errMsg = string(bytes)
	}
	http.Error(w, errMsg, statusCode)
}

// InternalServerError replies to request with InternalServerError.
func InternalServerError(w http.ResponseWriter, message string) {
	HTTPError(w, http.StatusInternalServerError, "InternalServerError", message)
}
