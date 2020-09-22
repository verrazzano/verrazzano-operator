// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	s := []string{"foo", "bar", "baz"}
	assert.True(t, StringArrayContains(s, "foo"))
	assert.True(t, StringArrayContains(s, "bar"))
	assert.True(t, StringArrayContains(s, "baz"))
	assert.False(t, StringArrayContains(s, "biz"))
}
