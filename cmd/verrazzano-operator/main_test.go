// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_initFlags(t *testing.T) {
	// GIVEN a set of expected string and boolean flags
	var expectedStringFlags = map[string]string{
		"master":                  "",
		"kubeconfig":              "",
		"watchNamespace":          "",
		"enableMonitoringStorage": "true",
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

// Test_prepare tests the prepare func refactored from main to have better coverage
// GIVEN a command line argument --zap-log-level=debug
//  WHEN prepare is called (from main)
//  THEN the zap-log-level flag is bound and the argument is parsed
func Test_prepare(t *testing.T) {
	os.Args[1] = "--zap-log-level=debug"
	err := prepare()
	assert.NoError(t, err)

	zapLevelFlag := flag.Lookup("zap-log-level")
	assert.NotNil(t, zapLevelFlag)
	assert.Equal(t, "debug", zapLevelFlag.Value.String())
}
