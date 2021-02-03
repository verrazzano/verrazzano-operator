// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

// TestCreateClusterRolesVmiSystem tests creation of cluster roles when the binding name is 'system'
// GIVEN a cluster which has no cluster roles
//  WHEN I call CreateClusterRoles with a binding named 'system'
//  THEN there should be an expected set of system cluster roles
//   AND each system cluster role should have an expected set of policy rules
func TestCreateClusterRolesVmiSystem(t *testing.T) {
	modelBindingPair := types.NewModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// change the binding name to the VmiSystemBindingName so that CreateClusterRoles creates the system cluster roles
	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	err := CreateClusterRoles(&modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CreateClusterRoles: %v", err)
	}

	// it is expected that after the call to CreateClusterRoles that all of the system cluster roles exist
	expectedRoles := monitoring.GetSystemClusterRoles("cluster1")

	// make sure that all of the expected roles exist
	for _, expected := range expectedRoles {
		cr, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), expected.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("got an error trying to get a cluster role %s: %v", expected.Name, err)
		}
		// verify that the expected policy rules are set on each role
		assertGivenRoleHasExpectedRules(t, cr, expected.Rules)
	}
}

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
