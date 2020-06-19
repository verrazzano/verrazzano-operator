// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ

import (
	"os"
	"testing"

	"github.com/golang/glog"
	"github.com/verrazzano/verrazzano-operator/test/integ/framework"
)

func TestMain(m *testing.M) {
	// Global setup
	if err := framework.Setup(); err != nil {
		glog.Errorf("Failed to setup framework: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	// Global tear-down
	if err := framework.Teardown(); err != nil {
		glog.Errorf("Failed to teardown framework: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}
