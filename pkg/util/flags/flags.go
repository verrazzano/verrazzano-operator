// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package flags

import (
	goflag "flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

// InitFlags parses, then logs the command line flags.
func InitFlags() {
	// Create log instance for initializing flags
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Flags").Str("name", "Init").Logger()

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	pflag.VisitAll(func(flag *pflag.Flag) {
		logger.Info().Msgf("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
