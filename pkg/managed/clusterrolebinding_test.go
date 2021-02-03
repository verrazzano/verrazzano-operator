// Copyright (c) 2020, Oracle and/or its affiliates.
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

// TestCreateClusterRoleBindingsVmiSystem tests creation of cluster role bindings when the binding name is 'system'
// GIVEN a cluster which has no cluster role bindings
//  WHEN I call CreateClusterRoleBindings with a binding named 'system'
//  THEN there should be an expected set of system cluster role bindings
//   AND each system cluster role binding should have an expected set of properties
func TestCreateClusterRoleBindingsVmiSystem(t *testing.T) {
	modelBindingPair := types.NewModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// change the binding name to the VmiSystemBindingName so that CreateClusterRoleBindings creates the system cluster role bindings
	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	err := CreateClusterRoleBindings(&modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CreateClusterRoleBindings: %v", err)
	}
	// it is expected that after the call to CreateClusterRoleBindings that all of the system cluster role bindings exist
	expectedRoleBindings := monitoring.GetSystemClusterRoleBindings("cluster1")

	// make sure that all of the expected role bindings exist
	for _, expectedBinding := range expectedRoleBindings {
		binding, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), expectedBinding.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("got an error trying to get a cluster role binding %s: %v", expectedBinding.Name, err)
		}
		// verify that the expected properties are set on each role binding
		assertClusterRoleBindingMatches(t, binding, expectedBinding)
	}
}

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
