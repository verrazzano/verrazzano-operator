// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/verrazzano/verrazzano-operator/test/integ/framework"
)

func TestMain(m *testing.M) {
	// Create log instance for building model pair
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Main").Str("name", "Test").Logger()

	// Global setup
	if err := framework.Setup(); err != nil {
		logger.Error().Msgf("Failed to setup framework: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	// Global tear-down
	if err := framework.Teardown(); err != nil {
		logger.Error().Msgf("Failed to teardown framework: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}
