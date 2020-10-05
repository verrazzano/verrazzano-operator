// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"reflect"
	"testing"
)

const verrazzanoURI = "test"
const clusterName1 = "cluster1"

// TestCreateDaemonSets tests that the creation of daemon sets is skipped when the binding name
// does not equal 'system'... CreateDaemonSets skips creating Daemon sets if binding is not System binding
// GIVEN a cluster which does not have any daemon sets
//  WHEN I call CreateDaemonSets with a binding named 'testBinding'
//  THEN there should be no daemon sets created
func TestCreateDaemonSets(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections[clusterName1]

	err := CreateDaemonSets(modelBindingPair, clusterConnections, "test")
	assert.Nil(err, "got an error from CreateDaemonSets: %v", err)
	lister := clusterConnection.DaemonSetLister

	selector := labels.Everything()
	sets, err := lister.List(selector)
	assert.Nil(err, "can't get daemon sets: %v", err)

	// assert creating Daemon sets was skipped
	assert.Empty(sets, "expected no deamon sets")
}

// TestCreateDaemonSetsVmiSystem tests the creation of daemon sets when the binding name is 'system'
// GIVEN a cluster which does not have any daemon sets
//  WHEN I call CreateDaemonSets with a binding named 'system'
//  THEN there should be a set of system daemon sets created for the specified ManagedCluster
func TestCreateDaemonSetsVmiSystem(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections[clusterName1]

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	err := CreateDaemonSets(modelBindingPair, clusterConnections, verrazzanoURI)
	assert.Nil(err, "got an error from CreateDaemonSets: %v", err)

	// assert that the daemon sets in the given cluster match the expected daemon sets
	assertExpectedDaemonSets(t, clusterConnection, monitoring.SystemDaemonSets(clusterName1, verrazzanoURI))
}

// TestCreateDaemonSetsUpdateExistingSet tests that an existing daemon set will be properly updated
// on a call to CreateDaemonSets.
// GIVEN a cluster which has a single daemon set named 'filebeat'
//  WHEN I call CreateDaemonSets with a binding named 'system'
//  THEN there should be a set of system daemon sets;  2 created & 1 updated(filebeat) for the specified ManagedCluster
func TestCreateDaemonSetsUpdateExistingSet(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections[clusterName1]

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName

	ds := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "filebeat",
			Namespace: "logging",
		},
	}
	_, err := clusterConnection.KubeClient.AppsV1().DaemonSets("logging").Create(context.TODO(), &ds, metav1.CreateOptions{})
	assert.Nil(err, "can't create daemon sets: %v", err)

	err = CreateDaemonSets(modelBindingPair, clusterConnections, verrazzanoURI)
	assert.Nil(err, "got an error from CreateDaemonSets: %v", err)

	// assert that the daemon sets in the given cluster match the expected daemon sets
	assertExpectedDaemonSets(t, clusterConnection, monitoring.SystemDaemonSets(clusterName1, verrazzanoURI))
}

// assertExpectedDaemonSets asserts that the daemon sets in the given cluster match the expected daemon sets
func assertExpectedDaemonSets(t *testing.T, clusterConnection *util.ManagedClusterConnection, expectedDaemonSets []*appsv1.DaemonSet) {
	assert := assert.New(t)

	// get the daemon sets from the given cluster connection
	lister := clusterConnection.DaemonSetLister
	selector := labels.Everything()
	daemonSets, err := lister.List(selector)
	assert.Nil(err, "can't get daemon sets: %v", err)

	// assert that the daemon sets from the cluster match the expected
	assert.Len(daemonSets, len(expectedDaemonSets), "can't match the expected daemon sets")
	for _, expected := range expectedDaemonSets {
		matched := false
		for _, daemonSet := range daemonSets {
			matched = reflect.DeepEqual(expected.Kind, daemonSet.Kind) &&
				reflect.DeepEqual(expected.Spec, daemonSet.Spec)
			if matched {
				break
			}
		}
		assert.True(matched, "can't match the expected daemon set %v", expected)
	}
}
