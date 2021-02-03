// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	asserts "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

// SecretProperties contains the set of properties used to identify a secret during testing
type SecretProperties struct {
	name      string
	namespace string
}

// assertSecretProperties will test whether the given list of secrets contains the expected secrets
// based on the provided secret attributes/properties (e.g. name and namespace)
func assertSecretProperties(expectedSecretValues []SecretProperties, list *corev1.SecretList, assert *asserts.Assertions) {
	for _, expectedSecretValue := range expectedSecretValues {
		match := false
		for _, secret := range list.Items {
			if secret.Name == expectedSecretValue.name && secret.Namespace == expectedSecretValue.namespace {
				match = true
				break
			}
		}
		assert.True(match, "expected secret not found")
	}
}
