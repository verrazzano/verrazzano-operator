// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package flags

import (
	goflag "flag"

	"go.uber.org/zap"

	"github.com/spf13/pflag"
)

// InitFlags parses, then logs the command line flags.
func InitFlags() {
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	pflag.VisitAll(func(flag *pflag.Flag) {
		zap.S().Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
