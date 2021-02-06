// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	"reflect"
	"testing"
)

// assertClusterRoleBindingMatches asserts that the given cluster role binding matches the expected cluster role binding
func assertClusterRoleBindingMatches(t *testing.T, binding *rbacv1.ClusterRoleBinding, expectedBinding *rbacv1.ClusterRoleBinding) {
	assert := assert.New(t)

	assert.Equal(expectedBinding.Name, binding.Name, "expected binding names to match")
	assert.Equal(expectedBinding.Kind, binding.Kind, "expected binding kind to match")
	assert.True(reflect.DeepEqual(binding.RoleRef, expectedBinding.RoleRef), "expected binding role refs to match")
	assert.Equal(len(expectedBinding.Subjects), len(binding.Subjects), "expected binding subjects to match")

	for _, expectedSubject := range expectedBinding.Subjects {
		matched := false
		for _, subject := range binding.Subjects {
			matched = reflect.DeepEqual(expectedSubject, subject)
			if matched {
				break
			}
		}
		if !matched {
			t.Fatalf("can't match the expected subject %v in the given cluster role binding %v", expectedSubject, binding)
		}
	}
}
