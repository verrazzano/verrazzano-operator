// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"net/http"

	"github.com/verrazzano/verrazzano-operator/pkg/api/instance"
)

// EnableCors adds headers necessary for the browser running the UI to be able to call the API server
func EnableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", instance.GetConsoleURL())
	// The web UI expects to be able to see X-Total-Count, which is uses for paging result sets
	(*w).Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
}

// SetupOptionsResponse creates the headers that are needed for a HTTP OPTIONS request
// The web UI makes an OPTIONS request before each GET/POST/etc.
func SetupOptionsResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", instance.GetConsoleURL())
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
