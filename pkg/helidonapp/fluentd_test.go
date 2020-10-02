// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
)

func TestCreateFluentdConfigMap(t *testing.T) {
	app := v1beta1v8o.VerrazzanoHelidon{}
	labels := map[string]string{"myLabel": "hello"}
	cm := CreateFluentdConfigMap(&app, "", labels)
	assert.Equal(t, 1, len(cm.Data), "Expected Data size to be 1")
	assert.NotNil(t, cm.Data[fluentdConf], "Expected fluentd.conf")
}
