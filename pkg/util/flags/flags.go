// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package flags

import (
	goflag "flag"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

// InitFlags parses, then logs the command line flags.
func InitFlags() {
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	pflag.VisitAll(func(flag *pflag.Flag) {
		glog.V(4).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
