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
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

// TestCleanupOrphanedClusterRoles tests the cleanup of orphaned cluster roles
// GIVEN a cluster which has no existing cluster roles
//  WHEN I create an orphaned cluster (not expected on the cluster) and call CleanupOrphanedClusterRoles
//  THEN the orphaned cluster role should be cleaned up (deleted)
func TestCleanupOrphanedClusterRoles(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster3"]

	// create a cluster role that is not expected on the cluster
	cr := v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orphaned",
			Labels: map[string]string{
				"verrazzano.binding": "testBinding",
				"verrazzano.cluster": "cluster3",
			},
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to create a cluster role: %v", err)
	}

	// the orphaned cluster role should be cleaned up by the call to CleanupOrphanedClusterRoles
	err = CleanupOrphanedClusterRoles(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CleanupOrphanedClusterRoles: %v", err)
	}

	// verify that the orphaned cluster role no longer exists
	_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), "orphaned", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected cluster role to be cleaned up")
	}
}

// TestCreateClusterRoles tests creation of cluster roles
// GIVEN a cluster which does not have a cluster role with the service account name
//  WHEN I call CreateClusterRoles
//  THEN there should be a cluster role with the service account created
//   AND that cluster role should have an expected set of policy rules
func TestCreateClusterRoles(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// the call to CreateClusterRoles should create a cluster role with the service account name
	// the expectation is that it doesn't exist prior to the call
	// verify that it doesn't exist
	serviceAccountName := util.GetServiceAccountNameForSystem()
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("did not expect cluster role %s to exist", serviceAccountName)
	}

	err = CreateClusterRoles(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CreateClusterRoles: %v", err)
	}

	// verify that a cluster role with the service account name has been created by the call to CreateClusterRoles
	cr, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error trying to get a cluster role: %v", err)
	}
	// verify that the expected properties and policy rules are set on the new cluster role
	assert.Equal(serviceAccountName, cr.Name, "the name of the cluster role should be %s", serviceAccountName)
	assertGivenRoleHasExpectedRules(t, cr, policyRules)
}

// TestCreateClusterRolesVmiSystem tests creation of cluster roles when the binding name is 'system'
// GIVEN a cluster which has no cluster roles
//  WHEN I call CreateClusterRoles with a binding named 'system'
//  THEN there should be an expected set of system cluster roles
//   AND each system cluster role should have an expected set of policy rules
func TestCreateClusterRolesVmiSystem(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	// change the binding name to the VmiSystemBindingName so that CreateClusterRoles creates the system cluster roles
	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	err := CreateClusterRoles(modelBindingPair, clusterConnections)
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

// TestCreateClusterRolesCreateAnAlreadyExistingRole tests that an existing cluster role will be properly updated
// on a call to CreateClusterRoles.
// GIVEN a cluster which has an expected cluster role with no policy rules
//  WHEN I call CreateClusterRoles
//  THEN the expected cluster role should still exist
//   AND the cluster role should have been updated with the expected set of policy rules
func TestCreateClusterRolesCreateAnAlreadyExistingRole(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	serviceAccountName := util.GetServiceAccountNameForSystem()

	// create a cluster role with the service account name but NO rules
	// the expectation is that the call to CreateClusterRoles should update the cluster role with the missing rules
	cr := &v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceAccountName,
			Labels: util.GetManagedLabelsNoBinding("cluster1"),
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error trying to create a cluster role: %v", err)
	}

	err = CreateClusterRoles(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("got an error from CreateClusterRoles: %v", err)
	}

	// verify that a cluster role with the service account name exists
	cr, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error trying to  get a cluster role: %v", err)
	}
	// verify that the expected properties and policy rules are set on the new cluster role
	assert.Equal(serviceAccountName, cr.Name, "the name of the cluster role should be %s", serviceAccountName)
	assertGivenRoleHasExpectedRules(t, cr, policyRules)
}

// TestDeleteClusterRoles tests that cluster roles are deleted when DeleteClusterRoles is called.
// CASE 1: bindingLabel = false
//  GIVEN a cluster which has a cluster role that should be cleaned up by a call to DeleteClusterRoles (based on labels)
//   WHEN I call DeleteClusterRoles with the arg bindingLabel = false
//   THEN the cluster role matched by "k8s-app" and "verrazzano.cluster" labels should be deleted
// CASE 2: bindingLabel = true
//  GIVEN a cluster which has a cluster role that should be cleaned up by a call to DeleteClusterRoles (based on labels)
//   WHEN I call DeleteClusterRoles with the arg bindingLabel = true
//   THEN the cluster role matched by "verrazzano.binding" label should be deleted
func TestDeleteClusterRoles(t *testing.T) {
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
			// test with bindingLabel arg set to false and matching cluster role labels
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
			// test with bindingLabel arg set to true and matching cluster role labels
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

			// create a cluster role that should be cleaned up by a call to DeleteClusterRoles (based on labels)
			cr := v1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "delete-me",
					Labels: tt.args.labels,
				},
			}
			_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("got an error trying to create a cluster role: %v", err)
			}

			// create a cluster role that should NOT be cleaned up by a call to DeleteClusterRoles (based on labels)
			cr = v1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dont-delete-me",
				},
			}
			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("got an error trying to create a cluster role: %v", err)
			}

			if err := DeleteClusterRoles(tt.args.mbPair, tt.args.availableManagedClusterConnections, tt.args.bindingLabel); (err != nil) != tt.wantErr {
				t.Errorf("DeleteClusterRoles() error = %v, wantErr %v", err, tt.wantErr)
			}

			// the expectation is that this cluster role should be deleted after the call to DeleteClusterRoles
			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), "delete-me", metav1.GetOptions{})
			if err == nil {
				t.Fatal("expected cluster role 'delete-me' to be deleted")
			}

			// the expectation is that this cluster role should NOT be deleted after the call to DeleteClusterRoles
			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), "dont-delete-me", metav1.GetOptions{})
			if err != nil {
				t.Fatal("expected cluster role 'dont-delete-me' to be still exist")
			}
		})
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
