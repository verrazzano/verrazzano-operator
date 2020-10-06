// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package signals

import (
	"github.com/stretchr/testify/assert"
	"os"
	"syscall"
	"testing"
)

// TestSetupSignalHandler tests that the Signal Handler returned is valid
// GIVEN that a valid Signal Handler Channel is returned
// WHEN I send a SIGTERM to the Process
// THEN a value should be received from the SignalHandler Channel
func TestSetupSignalHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "verifySetupSignalHandler",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			gotStopCh := SetupSignalHandler()
			assert.NotNil(gotStopCh, "expect the returned signal handler to be not nil")
			assert.IsType(make(<-chan struct{}), gotStopCh, "expect the returned signal handler to be of type <-chan struct{}")
			proc, _ := os.FindProcess(os.Getpid())
			proc.Signal(syscall.SIGTERM)
			// NOTE: If the SIGTERM is not sent, the pulling a value off the channel will block
			msg := <-gotStopCh
			assert.NotNil(msg, "expect SigTerm to cause close on SignalHandler Channel and msg returned from channel to be not nil")

		})
	}
}
