// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

var allHTTPMethods = []string{http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete,
	http.MethodConnect, http.MethodOptions, http.MethodPost, http.MethodTrace}

var readMethods []string = []string{http.MethodGet}
var readCreateMethods = []string{http.MethodGet, http.MethodPost}
var readUpdateDeleteMethods = []string{http.MethodGet, http.MethodPut, http.MethodDelete}
var readPatchDeleteMethods = []string{http.MethodGet, http.MethodPatch, http.MethodDelete}

var expectedRequestPaths = map[string][]string{
	addAPIPrefix("/"):                       allHTTPMethods,
	addAPIPrefix("/instance"):               readMethods,
	addAPIPrefix("/applications"):           readCreateMethods,
	addAPIPrefix("/applications/1"):         readUpdateDeleteMethods,
	addAPIPrefix("/clusters"):               readMethods,
	addAPIPrefix("/clusters/3"):             readMethods,
	addAPIPrefix("/domains"):                readMethods,
	addAPIPrefix("/domains/25"):             readMethods,
	addAPIPrefix("/grids/"):                 readMethods,
	addAPIPrefix("/grids/someid"):           readMethods,
	addAPIPrefix("/images/"):                readMethods,
	addAPIPrefix("/images/someImgid"):       readMethods,
	addAPIPrefix("/jobs/"):                  readMethods,
	addAPIPrefix("/jobs/someJobid"):         readMethods,
	addAPIPrefix("/microservices/"):         readMethods,
	addAPIPrefix("/microservices/someMsId"): readMethods,
	addAPIPrefix("/operators/"):             readMethods,
	addAPIPrefix("/operators/someOpId"):     readMethods,
	addAPIPrefix("/secrets"):                readCreateMethods,
	addAPIPrefix("/secrets/someSecretId"):   readPatchDeleteMethods,
}

func addAPIPrefix(path string) string {
	return fmt.Sprintf("%s%s", apiVersionPrefix, path)
}

// Only routes under the apiVersionPrefix should be registered - sample a few without the prefix to
// make sure they are not registered
var unexpectedRequestPaths = map[string][]string{
	"/":               allHTTPMethods,
	"/instance":       allHTTPMethods,
	"/applications/":  allHTTPMethods,
	"/jobs/someJobId": allHTTPMethods,
}

// Test_registerPathHandlers tests that registerPathHandlers registers the expected routes
// and none of the routes that we don't expect
func Test_registerPathHandlers(t *testing.T) {

	// GIVEN a path prefix of /somePrefix
	rootRouter := mux.NewRouter().StrictSlash(true)

	// WHEN path handlers are registered
	apiRouter := registerPathHandlers(rootRouter)

	// THEN assert that the expected methods for each path were registered
	for path, expectedMethods := range expectedRequestPaths {
		for _, expectedMethod := range expectedMethods {
			assertExpected(t, apiRouter, expectedMethod, path)
		}
		// AND assert that no unexpected methods were registered
		unexpectedMethods := getComplementaryMethods(expectedMethods)
		for _, unexpectedMethod := range unexpectedMethods {
			assertNotExpected(t, apiRouter, unexpectedMethod, path)
		}
	}

	// AND assert that no unexpected paths were registered
	for unexpectedPath, methods := range unexpectedRequestPaths {
		for _, method := range methods {
			assertNotExpected(t, apiRouter, method, unexpectedPath)
		}
	}
}

func Test_initFlags(t *testing.T) {
	// GIVEN a set of expected string and boolean flags
	var expectedStringFlags = map[string]string{
		"master":                       "",
		"kubeconfig":                   "",
		"watchNamespace":               "",
		"verrazzanoUri":                "",
		"helidonAppOperatorDeployment": "",
		"enableMonitoringStorage":      "",
		"apiServerRealm":               "",
	}
	var expectedBoolFlags = map[string]bool{
		"startController": true,
	}

	foundStringFlags := map[string]string{}
	foundBoolFlags := map[string]bool{}
	var foundFlags *flag.FlagSet
	// GIVEN functions to "register" string and bool flags
	stringVarFunc := func(p *string, name string, value string, usage string) {
		foundStringFlags[name] = value
	}
	boolVarFunc := func(p *bool, name string, value bool, usage string) {
		foundBoolFlags[name] = value
	}
	flagSetFunc := func(fs *flag.FlagSet) {
		foundFlags = fs
	}

	// WHEN its initFlags is called
	initFlags(stringVarFunc, boolVarFunc, flagSetFunc)

	// THEN assert that the expected command line flags are initialized
	assert.Len(t, foundStringFlags, len(expectedStringFlags))
	assert.Len(t, foundBoolFlags, len(expectedBoolFlags))
	for expectedName, expectedValue := range expectedStringFlags {
		assert.Equal(t, expectedValue, foundStringFlags[expectedName])
	}
	for expectedName, expectedValue := range expectedBoolFlags {
		assert.Equal(t, expectedValue, foundBoolFlags[expectedName])
	}
	assert.NotNil(t, foundFlags)
}

// assertNotExpected asserts that the rootRouter does NOT contain a match for the given
// http method and path
func assertNotExpected(t *testing.T, rootRouter *mux.Router, method string, path string) {
	req, err := http.NewRequest(method, path, nil)
	assert.NoError(t, err, "creating test http request for path "+path)
	routeMatch := &mux.RouteMatch{}
	assert.False(t, rootRouter.Match(req, routeMatch),
		fmt.Sprintf("Unexpected Match found for '%s %s'", method, path))
}

// assertExpected asserts that the rootRouter contains a match for the given http method and path
func assertExpected(t *testing.T, rootRouter *mux.Router, method string, path string) {
	req, err := http.NewRequest(method, path, nil)
	assert.NoError(t, err, "creating test http request for path "+path)
	routeMatch := &mux.RouteMatch{}

	// assert that the match for given request path and request method was found
	assert.True(t, rootRouter.Match(req, routeMatch),
		fmt.Sprintf("Matcher for '%s %s' not found", method, path))

	assert.NoError(t, routeMatch.MatchErr,
		fmt.Sprintf("when matching path %s: %s", path, routeMatch.MatchErr))

}

// getComplementaryMethods returns the list of all http methods that are NOT in the given list
// e.g. if given {"GET", "PUT"}, then it returns a list of all http methods other than GET and PUT
func getComplementaryMethods(methods []string) []string {
	complementaryMethods := []string{}
	for _, eachMethod := range allHTTPMethods {
		if !util.Contains(methods, eachMethod) {
			complementaryMethods = append(complementaryMethods, eachMethod)
		}
	}
	return complementaryMethods
}

// Test_prepare tests the prepare func refactored from main to have better coverage
// GIVEN a command line argument --zap-log-level=debug
//  WHEN prepare is called (from main)
//  THEN the zap-log-level flag is bound and the argument is parsed
func Test_prepare(t *testing.T) {
	os.Args[1] = "--zap-log-level=debug"
	manifest, err := prepare()
	assert.NotNil(t, manifest)
	assert.NotNil(t, err)
	zapLevelFlag := flag.Lookup("zap-log-level")
	assert.NotNil(t, zapLevelFlag)
	assert.Equal(t, "debug", zapLevelFlag.Value.String())
}
