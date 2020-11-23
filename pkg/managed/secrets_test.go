// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// SecretProperties contains the set of properties used to identify a secret during testing
type SecretProperties struct {
	name      string
	namespace string
}

// TestCreateSecretes tests creation of secrets
// GIVEN a model binding pair and cluster connections configured with required secrets
//  WHEN the CreateSecrets is called
//  THEN the required secrets are generated and associated to the specified namespaces
func TestCreateSecrets(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	vmiSecret := monitoring.NewVmiSecret(modelBindingPair.Binding)
	secrets := &testutil.FakeSecrets{Secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}

	CreateSecrets(modelBindingPair, clusterConnections, clusterConnection.KubeClient, secrets)
	assert := asserts.New(t)
	list, err := clusterConnection.KubeClient.CoreV1().Secrets("test").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("got an error listing secrets: %v", err)
	}
	assert.Len(list.Items, 2, "expected exactly 2 Secrets")
	secretProperties := []SecretProperties{
		{
			name:      "testSecret1",
			namespace: "test",
		},
		{
			name:      "testSecret2",
			namespace: "test",
		},
	}
	assertSecretProperties(secretProperties, list, assert)

	list, err = clusterConnection.KubeClient.CoreV1().Secrets("test2").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("got an error listing secrets: %v", err)
	}
	assert.Len(list.Items, 2, "expected exactly 2 Secrets")
	secretProperties = []SecretProperties{
		{
			name:      "test2Secret1",
			namespace: "test2",
		},
		{
			name:      "test2Secret2",
			namespace: "test2",
		},
	}
	assertSecretProperties(secretProperties, list, assert)

	list, err = clusterConnection.KubeClient.CoreV1().Secrets("test3").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("got an error listing secrets: %v", err)
	}
	assert.Len(list.Items, 4, "expected exactly 4 Secrets")
	secretProperties = []SecretProperties{
		{
			name:      "test-wls-secret",
			namespace: "test3",
		},
		{
			name:      "testMysqlSecret",
			namespace: "test3",
		},
		{
			name:      "arbitrary-secret-1",
			namespace: "test3",
		},
		{
			name:      "arbitrary-secret-2",
			namespace: "test3",
		},
	}
	assertSecretProperties(secretProperties, list, assert)

}

// assertSecretProperties will test whether the given list of secrets contains the expected secrets
// based on the provided secret attributes/properties (e.g. name and namespace)
func assertSecretProperties(expectedSecretValues []SecretProperties, list *corev1.SecretList, assert *asserts.Assertions) {
	for _, expectedSecretValue := range expectedSecretValues {
		match := false
		for _, secret := range list.Items {
			if secret.Name == expectedSecretValue.name && secret.Namespace == expectedSecretValue.namespace {
				match = true
				break
			}
		}
		assert.True(match, "expected secret not found")
	}
}
