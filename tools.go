// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// +build tools

package tools

import (
        // Other tools
        _ "github.com/gordonklaus/ineffassign"
        _ "golang.org/x/lint"
        _ "github.com/go-bindata/go-bindata/go-bindata/v3"
        - "go.uber.org/zap"
)
