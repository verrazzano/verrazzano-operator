// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	"reflect"
	"testing"
)

// assertGivenRoleHasExpectedRules asserts that the given cluster role has the exact set of expected policy rules
func assertGivenRoleHasExpectedRules(t *testing.T, cr *v1.ClusterRole, expectedRules []rbacv1.PolicyRule) {
	assert := assert.New(t)

	// assert that the given role has the exact set of expected rules
	assert.Len(cr.Rules, len(expectedRules), "the given role has does not match the expected policy rules")
	for _, expected := range expectedRules {
		matched := false
		for _, rule := range cr.Rules {
			matched = reflect.DeepEqual(expected.APIGroups, rule.APIGroups) &&
				reflect.DeepEqual(expected.Resources, rule.Resources) &&
				reflect.DeepEqual(expected.Verbs, rule.Verbs)
			if matched {
				break
			}
		}
		if !matched {
			t.Fatalf("can't match the expected policy rule %v in the given cluster role %v", expected, cr)
		}
	}
}
