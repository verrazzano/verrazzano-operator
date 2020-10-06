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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"reflect"
	"testing"
)

var origGetEnvFunc = util.GetEnvFunc

// TestCreateDeployments tests the creation of deployments.
// GIVEN a cluster which does not have any deployments
//  WHEN I call CreateDeployments
//  THEN there should be an expected set of deployments created
func TestCreateDeployments(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	manifest := testutil.GetManifest()
	vmiSecret := monitoring.NewVmiSecret(modelBindingPair.Binding)
	secrets := &testutil.FakeSecrets{Secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}

	// temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	err := CreateDeployments(modelBindingPair, clusterConnections, &manifest, "testURI", secrets)
	assert.Nil(err, "got an error from CreateDeployments: %v", err)

	assertDeployments(err, clusterConnection, assert)
}

// TestCreateDeployments tests that an existing deployment is properly updated.
// GIVEN a cluster which has an existing expected deployment (verrazzano-wko-operator)
//  WHEN I call CreateDeployments
//  THEN there should be an expected set of deployments created
//    AND the existing deployment (verrazzano-wko-operator) should be updated as expected
func TestCreateDeploymentsUpdateExisting(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	manifest := testutil.GetManifest()
	vmiSecret := monitoring.NewVmiSecret(modelBindingPair.Binding)
	secrets := &testutil.FakeSecrets{Secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}

	// temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano-wko-operator",
			Namespace: "verrazzano-system",
		},
	}
	_, err := clusterConnection.KubeClient.AppsV1().Deployments("verrazzano-system").Create(context.TODO(), &deployment, metav1.CreateOptions{})
	assert.Nil(err, "got an error creating deployment: %v", err)

	err = CreateDeployments(modelBindingPair, clusterConnections, &manifest, "testURI", secrets)
	assert.Nil(err, "got an error from CreateDeployments: %v", err)

	assertDeployments(err, clusterConnection, assert)
}

// TestCreateDeploymentsVmiSystem tests the creation of deployments when the binding name is 'system'.
// GIVEN a cluster which does not have any deployments
//  WHEN I call CreateDeployments with a binding named 'system'
//  THEN there should be an expected set of system deployments created
func TestCreateDeploymentsVmiSystem(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	manifest := testutil.GetManifest()
	vmiSecret := monitoring.NewVmiSecret(modelBindingPair.Binding)
	secrets := &testutil.FakeSecrets{Secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}

	// temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName
	err := CreateDeployments(modelBindingPair, clusterConnections, &manifest, "testURI", secrets)
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
func assertDeployments(err error, clusterConnection *util.ManagedClusterConnection, assert *assert.Assertions) {
	// validate that the verrazzano-system deployments have been created
	list, err := clusterConnection.KubeClient.AppsV1().Deployments("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got error trying to get deployments: %v", err)
	assert.Len(list.Items, 3, "expected exactly 3 deployments in the verrazzano-system namespace")
	expectedDeployments := map[string]struct{}{"verrazzano-wko-operator": {}, "verrazzano-coh-cluster-operator": {}, "verrazzano-helidon-app-operator": {}}
	for _, deployment := range list.Items {
		assert.Contains(expectedDeployments, deployment.Name, "expected deployment not found")
		assert.Equal("cluster1", list.Items[0].Labels["verrazzano.cluster"], "label not equal to expected value")
		assert.Equal("verrazzano.io", list.Items[0].Labels["k8s-app"], "label not equal to expected value")
	}
	// validate that the monitoring deployments have been created
	list, err = clusterConnection.KubeClient.AppsV1().Deployments("monitoring").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err, "got error trying to get deployments: %v", err)
	assert.Len(list.Items, 1, "expected exactly 1 deployment in the monitoring namespace")
	assert.Equal("prom-pusher-testBinding", list.Items[0].Name, "expected deployment not found")
	assert.Equal("cluster1", list.Items[0].Labels["verrazzano.cluster"], "label not equal to expected value")
	assert.Equal("verrazzano.io", list.Items[0].Labels["k8s-app"], "label not equal to expected value")
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
