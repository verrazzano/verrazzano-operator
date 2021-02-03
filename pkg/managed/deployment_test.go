// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var origGetEnvFunc = util.GetEnvFunc

// TestCreateDeploymentsVmiSystem tests the creation of deployments when the binding name is 'system'.
// GIVEN a cluster which does not have any deployments
//  WHEN I call CreateDeployments with a binding named 'system'
//  THEN there should be an expected set of system deployments created
func TestCreateDeploymentsVmiSystem(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	vmiSecret := monitoring.NewVmiSecret(modelBindingPair.Binding)
	secrets := &testutil.FakeSecrets{Secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}

	// temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName
	err := CreateDeployments(modelBindingPair, clusterConnections,"testURI", secrets)
	assert.Nil(err, "got an error from CreateDeployments: %v", err)

	// validate that the created deployments match the expected deployments
	expectedDeployments, err := monitoring.GetSystemDeployments("cluster1", verrazzanoURI, util.GetManagedLabelsNoBinding("cluster1"), secrets)
	assert.Nil(err, "got error trying to get deployments: %v", err)
	assertExpectedDeployments(t, clusterConnection, expectedDeployments)
}

// getenv returns a mocked response for keys used by these tests
func getenv(key string) string {
	if key == "WLS_MICRO_REQUEST_MEMORY" || key == "COH_MICRO_REQUEST_MEMORY" || key == "HELIDON_MICRO_REQUEST_MEMORY" {
		return "2.5Gi"
	}
	return origGetEnvFunc(key)
}

// assertDeployments validates that the expected deployments have been created
func assertDeployments(err error, clusterConnection *util.ManagedClusterConnection, assert *assert.Assertions, generic bool) {

	// validate that the monitoring deployments have been created
	list, err := clusterConnection.KubeClient.AppsV1().Deployments("monitoring").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got error trying to get deployments: %v", err)
	assert.Len(list.Items, 1, "expected exactly 1 deployment in the monitoring namespace")
	assert.Equal("prom-pusher-testBinding", list.Items[0].Name, "expected deployment not found")
	assert.Equal("cluster1", list.Items[0].Labels["verrazzano.cluster"], "label not equal to expected value")
	assert.Equal("verrazzano.io", list.Items[0].Labels["k8s-app"], "label not equal to expected value")
	// validate that the generic component deployments have been created
	list, err = clusterConnection.KubeClient.AppsV1().Deployments("test").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got error trying to get deployments: %v", err)
	if generic {
		assert.Len(list.Items, 1, "expected exactly 1 deployment in the test namespace")
		assert.Equal("test-generic", list.Items[0].Name, "expected deployment not found")
		assert.Equal("cluster1", list.Items[0].Labels["verrazzano.cluster"], "label not equal to expected value")
		assert.Equal(int32(2), *list.Items[0].Spec.Replicas, "replicas not equal to expected value")
		assert.Equal("generic-image:1.0", list.Items[0].Spec.Template.Spec.Containers[0].Image, "container image not equal to expected value")
		assert.Equal("test-generic-image", list.Items[0].Spec.Template.Spec.Containers[0].Name, "container name not equal to expected value")
	} else {
		assert.Len(list.Items, 0, "expected exactly 0 deployment in the test namespace")
	}
}

// assertExpectedDeployments validates that the current deployments match the given expected deployments
func assertExpectedDeployments(t *testing.T, clusterConnection *util.ManagedClusterConnection, expectedDeployments []*appsv1.Deployment) {
	assert := assert.New(t)

	selector := labels.Everything()
	deployments, err := clusterConnection.DeploymentLister.List(selector)
	assert.Nil(err, "got error trying to get deployments: %v", err)
	assert.Len(deployments, len(expectedDeployments), "deployments do not match expected")

	for _, expectedDeployment := range expectedDeployments {
		match := false
		for _, deployment := range deployments {
			if deployment.Name == expectedDeployment.Name && reflect.DeepEqual(expectedDeployment.Labels, deployment.Labels) {
				match = true
				break
			}
		}
		assert.True(match, "expected deployment not found")
	}
}
