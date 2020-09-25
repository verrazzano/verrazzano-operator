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

func TestCleanupOrphanedClusterRoles(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster3"]

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
		t.Fatalf("can't create cluster role: %v", err)
	}

	CleanupOrphanedClusterRoles(modelBindingPair, clusterConnections)

	_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), "orphaned", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected cluster role to be cleaned up")
	}
}

func TestCreateClusterRoles(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	CreateClusterRoles(modelBindingPair, clusterConnections)

	list, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("can't list cluster roles: %v", err)
	}
	assert.Equal(1, len(list.Items))
	cr := list.Items[0]
	assert.Equal("verrazzano-system", cr.Name)
	assertRules(t, &cr, getRules())
}

func TestCreateClusterRolesVmiSystem(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	CreateClusterRoles(modelBindingPair, clusterConnections)

	list, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("can't list cluster roles: %v", err)
	}
	expectedRoles := monitoring.GetSystemClusterRoles("cluster1")
	assert.Equal(len(expectedRoles), len(list.Items))

	for _, expected := range expectedRoles {
		cr, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), expected.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("can't get cluster role %s: %v", expected.Name, err)
		}
		assertRules(t, cr, expected.Rules)
	}
}

func TestCreateClusterRolesUpdate(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	cr := v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   util.GetServiceAccountNameForSystem(),
			Labels: util.GetManagedLabelsNoBinding("cluster1"),
		},
	}
	_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("can't create cluster role: %v", err)
	}

	CreateClusterRoles(modelBindingPair, clusterConnections)

	list, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("can't list cluster roles: %v", err)
	}
	assert.Equal(1, len(list.Items))
	cr = list.Items[0]
	assert.Equal(util.GetServiceAccountNameForSystem(), cr.Name)
	assertRules(t, &cr, getRules())
}

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
			cr := v1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-clusterrole",
					Labels: tt.args.labels,
				},
			}
			_, err := clusterConnection.KubeClient.RbacV1().ClusterRoles().Create(context.TODO(), &cr, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("can't create cluster role: %v", err)
			}

			if err := DeleteClusterRoles(tt.args.mbPair, tt.args.availableManagedClusterConnections, tt.args.bindingLabel); (err != nil) != tt.wantErr {
				t.Errorf("DeleteClusterRoles() error = %v, wantErr %v", err, tt.wantErr)
			}

			_, err = clusterConnection.KubeClient.RbacV1().ClusterRoles().Get(context.TODO(), "test-clusterrole", metav1.GetOptions{})
			if err == nil {
				t.Fatal("expected cluster role to be deleted")
			}
		})
	}
}

func assertRules(t *testing.T, cr *v1.ClusterRole, expectedRules []rbacv1.PolicyRule) {
	assert := assert.New(t)
	assert.Equal(len(expectedRules), len(cr.Rules))
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
			t.Fatalf("can't match rule %v", expected)
		}
	}
}
