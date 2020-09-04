// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

func doTestCreateStorageOption(t *testing.T, option string, expected string) {
	storage := createStorageOption(option)
	prometheus := vmov1.Prometheus{
		Enabled: true,
		Storage: storage,
	}
	assert.Equal(t, expected, storage.Size, "right storage size")
	assert.Equal(t, expected, prometheus.Storage.Size, "right storage size")
}
func TestCreateStorageOption(t *testing.T) {
	doTestCreateStorageOption(t, "false", "")
	doTestCreateStorageOption(t, "False", "")
	doTestCreateStorageOption(t, "", "50Gi")
	doTestCreateStorageOption(t, "true", "50Gi")
	doTestCreateStorageOption(t, "True", "50Gi")
	doTestCreateStorageOption(t, "random", "50Gi")
}

func TestNewVmiSecrets(t *testing.T) {
	binding := v1beta1v8o.VerrazzanoBinding{}
	binding.Name = "TestNewVmiSecrets"
	vmi := newVMIs(&binding, "v8o.uri", "")
	assert.Equal(t, constants.VerrazzanoNamespace, vmi[0].Namespace, "right Namespace")
	assert.Equal(t, constants.VmiSecretName, vmi[0].Spec.SecretsName, "right SecretsName")
}
