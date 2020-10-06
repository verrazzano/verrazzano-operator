package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

var allHttpMethods = []string{http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete,
	http.MethodConnect, http.MethodOptions, http.MethodPost, http.MethodTrace}

var readMethods []string = []string{http.MethodGet}
var readCreateMethods = []string{http.MethodGet, http.MethodPost}
var readUpdateDeleteMethods = []string{http.MethodGet, http.MethodPut, http.MethodDelete}
var readPatchDeleteMethods = []string{http.MethodGet, http.MethodPatch, http.MethodDelete}

var expectedRequestPaths = map[string][]string{
	addApiPrefix("/"):                       allHttpMethods,
	addApiPrefix("/instance"):               readMethods,
	addApiPrefix("/applications"):           readCreateMethods,
	addApiPrefix("/applications/1"):         readUpdateDeleteMethods,
	addApiPrefix("/clusters"):               readMethods,
	addApiPrefix("/clusters/3"):             readMethods,
	addApiPrefix("/domains"):                readMethods,
	addApiPrefix("/domains/25"):             readMethods,
	addApiPrefix("/grids/"):                 readMethods,
	addApiPrefix("/grids/someid"):           readMethods,
	addApiPrefix("/images/"):                readMethods,
	addApiPrefix("/images/someImgid"):       readMethods,
	addApiPrefix("/jobs/"):                  readMethods,
	addApiPrefix("/jobs/someJobid"):         readMethods,
	addApiPrefix("/microservices/"):         readMethods,
	addApiPrefix("/microservices/someMsId"): readMethods,
	addApiPrefix("/operators/"):             readMethods,
	addApiPrefix("/operators/someOpId"):     readMethods,
	addApiPrefix("/secrets"):                readCreateMethods,
	addApiPrefix("/secrets/someSecretId"):   readPatchDeleteMethods,
}

func addApiPrefix(path string) string {
	return fmt.Sprintf("%s%s", apiVersionPrefix, path)
}

// Only routes under the apiVersionPrefix should be registered - sample a few without the prefix to
// make sure they are not registered
var unexpectedRequestPaths = map[string][]string{
	"/":               allHttpMethods,
	"/instance":       allHttpMethods,
	"/applications/":  allHttpMethods,
	"/jobs/someJobId": allHttpMethods,
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

	// GIVEN functions to "register" string and bool flags
	stringVarFunc := func(p *string, name string, value string, usage string) {
		foundStringFlags[name] = value
	}
	boolVarFunc := func(p *bool, name string, value bool, usage string) {
		foundBoolFlags[name] = value
	}

	// WHEN its initFlags is called
	initFlags(stringVarFunc, boolVarFunc)

	// THEN assert that the expected command line flags are initialized
	assert.Equal(t, len(expectedStringFlags), len(foundStringFlags))
	assert.Equal(t, len(expectedBoolFlags), len(foundBoolFlags))
	for expectedName, expectedValue := range expectedStringFlags {
		assert.Equal(t, expectedValue, foundStringFlags[expectedName])
	}
	for expectedName, expectedValue := range expectedBoolFlags {
		assert.Equal(t, expectedValue, foundBoolFlags[expectedName])
	}
}

// assertNotExpected asserts that the rootRouter does NOT contain a match for the given
// http method and path
func assertNotExpected(t *testing.T, rootRouter *mux.Router, method string, path string) {
	req, err := http.NewRequest(method, path, nil)
	assert.Nil(t, err, "Unexpected error creating test http request for path "+path)
	routeMatch := &mux.RouteMatch{}
	assert.False(t, rootRouter.Match(req, routeMatch),
		fmt.Sprintf("Unexpected Match found for '%s %s'", method, path))
}

// assertExpected asserts that the rootRouter contains a match for the given http method and path
func assertExpected(t *testing.T, rootRouter *mux.Router, method string, path string) {
	req, err := http.NewRequest(method, path, nil)
	assert.Nil(t, err, "Unexpected error creating test http request for path "+path)
	routeMatch := &mux.RouteMatch{}

	// assert that the match for given request path and request method was found
	assert.True(t, rootRouter.Match(req, routeMatch),
		fmt.Sprintf("Matcher for '%s %s' not found", method, path))

	assert.Nil(t, routeMatch.MatchErr,
		fmt.Sprintf("Error matching path %s: %s", path, routeMatch.MatchErr))

	fmt.Printf("Matched %s %s\n", method, path)
}

// getComplementaryMethods returns the list of all http methods that are NOT in the given list
// e.g. if given {"GET", "PUT"}, then it returns a list of all http methods other than GET and PUT
func getComplementaryMethods(methods []string) []string {
	complementaryMethods := []string{}
	for _, eachMethod := range allHttpMethods {
		if !sliceContains(methods, eachMethod) {
			complementaryMethods = append(complementaryMethods, eachMethod)
		}
	}
	return complementaryMethods
}

func sliceContains(theSlice []string, itemToFind string) bool {
	for _, item := range theSlice {
		if item == itemToFind {
			return true
		}
	}
	return false
}
