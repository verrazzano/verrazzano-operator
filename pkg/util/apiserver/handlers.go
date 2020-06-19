// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
)

// CORSHandler is an HTTP handler that will handle CORS preflight requests
// or delegate to the actual handler on real requests
func CORSHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		EnableCors(&w)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			// CORS preflight request
			SetupOptionsResponse(&w, r)
			glog.Info("OPTIONS /" + r.URL.Path)
			return
		}
		// actual request
		h.ServeHTTP(w, r)
	})

}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func HttpError(w http.ResponseWriter, statusCode int, code, message string) {
	error := Error{Code: code, Message: message}
	errMsg := message
	bytes, err := json.Marshal(&error)
	if err == nil || (bytes != nil && len(bytes) > 0) {
		errMsg = string(bytes)
	}
	http.Error(w, errMsg, statusCode)
}

func InternalServerError(w http.ResponseWriter, message string) {
	HttpError(w, http.StatusInternalServerError, "InternalServerError", message)
}
