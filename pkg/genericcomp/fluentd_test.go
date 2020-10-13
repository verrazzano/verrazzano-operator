// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package genericcomp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateFluentdConfigMap(t *testing.T) {
	assertion := assert.New(t)

	labels := map[string]string{
		"myLabel": "hello",
	}
	cm := CreateFluentdConfigMap("test-generic", "test-namespace", labels)
	assertion.Equal("test-generic-fluentd", cm.Name, "config map name not equal to expected value")
	assertion.Equal("test-namespace", cm.Namespace, "config map namespace not equal to expected value")
	assertion.Equal(labels, cm.Labels, "config map labels not equal to expected value")
	assertion.Equal(1, len(cm.Data), "config map data size not equal to expected value")
	assertion.NotNil(cm.Data[fluentdConf], "Expected fluentd.conf")
	assertion.Equal(cm.Data[fluentdConf], FluentdConfiguration, "config map data not equal to expected value")
}
