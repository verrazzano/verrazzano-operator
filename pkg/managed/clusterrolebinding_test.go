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
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

// TestCleanupOrphanedClusterRoleBindings tests the cleanup of orphaned cluster role bindings
// GIVEN a cluster which has no existing cluster role bindings
//  WHEN I create an orphaned cluster binding(not expected on the cluster) and call CleanupOrphanedClusterRoleBindings
//  THEN the orphaned cluster role binding should be cleaned up (deleted)
func TestCleanupOrphanedClusterRoleBindings(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster3"]

	// create a cluster role binding that is not expected on the cluster
	binding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orphaned",
			Labels: map[string]string{
				"verrazzano.binding": "testBinding",
				"verrazzano.cluster": "cluster3",
			},
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), &binding, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to create a cluster role binding: %v", err)
	}

	// the orphaned cluster role binding should be cleaned up by the call to CleanupOrphanedClusterRoleBindings
	err = CleanupOrphanedClusterRoleBindings(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CleanupOrphanedClusterRoleBindings: %v", err)
	}

	// verify that the orphaned cluster role binding no longer exists
	_, err = clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), "orphaned", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected cluster role binding to be cleaned up")
	}
}

// TestCreateClusterRoleBindings tests creation of cluster role bindings
// GIVEN a cluster which does not have a cluster role binding with the service account name
//  WHEN I call CreateClusterRoleBindings
//  THEN there should be a cluster role binding with the service account created
//   AND that cluster role binding should ...
func TestCreateClusterRoleBindings(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// the call to CreateClusterRoleBindings should create a cluster role binding with the service account name
	// the expectation is that it doesn't exist prior to the call
	// verify that it doesn't exist
	serviceAccountName := util.GetServiceAccountNameForSystem()
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("did not expect cluster role binding %s to exist", serviceAccountName)
	}

	err = CreateClusterRoleBindings(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CreateClusterRoleBindings: %v", err)
	}

	// verify that a cluster role binding with the service account name has been created by the call to CreateClusterRoleBindings
	binding, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error trying to get a cluster role binding: %v", err)
	}
	// verify the expected properties on the new cluster role binding
	assert.Equal(serviceAccountName, binding.Name, "the name of the cluster role binding should be %s", serviceAccountName)
	assert.Len(binding.Subjects, 1, "expected exactly one subject")
	assert.Equal(serviceAccountName, binding.Subjects[0].Name, "expected the name of the subject to be %s", serviceAccountName)
	assert.Equal(serviceAccountName, binding.RoleRef.Name, "expected the name of the role ref to be %s", serviceAccountName)
	assert.Equal("ClusterRole", binding.RoleRef.Kind, "expected the kind of the role ref to be ClusterRole")
	assert.Equal("rbac.authorization.k8s.io", binding.RoleRef.APIGroup, "expected the API group of the role ref to be rbac.authorization.k8s.io")
}

// TestCreateClusterRoleBindingsVmiSystem tests creation of cluster role bindings when the binding name is 'system'
// GIVEN a cluster which has no cluster role bindings
//  WHEN I call CreateClusterRoleBindings with a binding named 'system'
//  THEN there should be an expected set of system cluster role bindings
//   AND each system cluster role binding should have an expected set of properties
func TestCreateClusterRoleBindingsVmiSystem(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// change the binding name to the VmiSystemBindingName so that CreateClusterRoleBindings creates the system cluster role bindings
	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	err := CreateClusterRoleBindings(modelBindingPair, clusterConnections)
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

// TestCreateClusterRoleBindingsCreateAnAlreadyExistingBinding tests that an existing cluster role binding will be
// properly updated on a call to CreateClusterRoleBindings.
// GIVEN a cluster which has an expected cluster role binding with no subject or role ref
//  WHEN I call CreateClusterRoleBindings
//  THEN the expected cluster role binding should still exist
//   AND the cluster role binding should have been updated with the expected set of properties
func TestCreateClusterRoleBindingsCreateAnAlreadyExistingBinding(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	serviceAccountName := util.GetServiceAccountNameForSystem()

	// create a cluster role binding with the service account name but no subject or role ref
	// the expectation is that the call to CreateClusterRoleBindings should update the binding with the missing properties
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceAccountName,
			Labels: util.GetManagedLabelsNoBinding("cluster1"),
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to create a cluster role: %v", err)
	}

	err = CreateClusterRoleBindings(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CreateClusterRoleBindings: %v", err)
	}

	// verify that a cluster role binding with the service account name exists
	binding, err = clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error trying to  get a cluster role binding: %v", err)
	}
	// verify that the expected properties are set on the new cluster role binding
	expectedBinding := getManagedClusterRoleBinding("cluster1")
	assertClusterRoleBindingMatches(t, binding, expectedBinding)
}

// TestDeleteClusterRoleBindings tests that cluster role bindings are deleted when DeleteClusterRoleBindings is called.
// CASE 1: bindingLabel = false
//  GIVEN a cluster which has a cluster role binding that should be cleaned up by a call to DeleteClusterRoleBindings
//   WHEN I call DeleteClusterRoleBindings with the arg bindingLabel = false
//   THEN the cluster role binding matched by "k8s-app" and "verrazzano.cluster" labels should be deleted
// CASE 2: bindingLabel = true
//  GIVEN a cluster which has a cluster role binding that should be cleaned up by a call to DeleteClusterRoleBindings
//   WHEN I call DeleteClusterRoleBindings with the arg bindingLabel = true
//   THEN the cluster role binding matched by "verrazzano.binding" label should be deleted
func TestDeleteClusterRoleBindings(t *testing.T) {
	type args struct {
		mbPair                             *types.ModelBindingPair
		availableManagedClusterConnections map[string]*util.ManagedClusterConnection
		bindingLabel                       bool
		labels                             map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "BindingLabelFalse",
			args: args{
				mbPair:                             testutil.GetModelBindingPair(),
				availableManagedClusterConnections: testutil.GetManagedClusterConnections(),
				bindingLabel:                       false,
				labels: map[string]string{
					"k8s-app":            "verrazzano.io",
					"verrazzano.cluster": "cluster1",
				},
			},
			wantErr: false,
		},
		{
			name: "BindingLabelTrue",
			args: args{
				mbPair:                             testutil.GetModelBindingPair(),
				availableManagedClusterConnections: testutil.GetManagedClusterConnections(),
				bindingLabel:                       true,
				labels: map[string]string{
					"verrazzano.binding": "testBinding",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterConnection := tt.args.availableManagedClusterConnections["cluster1"]

			// create a cluster role binding that should be cleaned up by a call to DeleteClusterRoleBindings (based on labels)
			b := rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "delete-me",
					Labels: tt.args.labels,
				},
			}
			_, err := clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), &b, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("got an error trying to create a cluster role binding: %v", err)
			}

			// create a cluster role binding that should NOT be cleaned up by a call to DeleteClusterRoleBindings (based on labels)
			b = rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dont-delete-me",
				},
			}
			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), &b, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("got an error trying to create a cluster role binding: %v", err)
			}

			if err := DeleteClusterRoleBindings(tt.args.mbPair, tt.args.availableManagedClusterConnections, tt.args.bindingLabel); (err != nil) != tt.wantErr {
				t.Errorf("DeleteClusterRoleBindings() error = %v, wantErr %v", err, tt.wantErr)
			}

			// the expectation is that this cluster role binding should be deleted after the call to DeleteClusterRoleBindings
			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), "delete-me", metav1.GetOptions{})
			if err == nil {
				t.Fatal("expected cluster role binding 'delete-me' to be deleted")
			}

			// the expectation is that this cluster role binding should NOT be deleted after the call to DeleteClusterRoleBindings
			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), "dont-delete-me", metav1.GetOptions{})
			if err != nil {
				t.Fatal("expected cluster role binding 'dont-delete-me' to be still exist")
			}
		})
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
