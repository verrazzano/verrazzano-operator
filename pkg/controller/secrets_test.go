// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	. "github.com/verrazzano/verrazzano-operator/pkg/managed"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// Test creating and getting a secret
func TestKubeSecrets_CreateGet(t *testing.T) {
	connections := GetManagedClusterConnections()
	connection := connections["cluster1"]

	origSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-secret-namespace-1",
			Name:      "test-secret-name-1"},
		Immutable:  nil,
		Data:       nil,
		StringData: nil,
		Type:       "test-secret-type",
	}

	secrets := KubeSecrets{
		namespace:     "test-secret-namespace-1",
		kubeClientSet: connection.KubeClient,
		secretLister:  connection.SecretLister,
	}

	secret, err := secrets.Create(&origSecret)
	assert.NoErrorf(t, err, "Can't create secret: %v", err)
	assert.NotNilf(t, secret, "Invalid secret created: %v", secret)
	assert.Equal(t, &origSecret, secret)

	// Get the test secret.
	secret, err = secrets.Get("test-secret-name-1")
	assert.NoErrorf(t, err, "Can't find secret: %v", err)
	assert.NotNilf(t, secret, "Invalid secret found: %v", secret)
	assert.Equal(t, "test-secret-namespace-1", secret.Namespace)
	assert.Equal(t, "test-secret-name-1", secret.Name)

	// Get the test secret.
	secret, err = secrets.Get("test-secret-name-2")
	assert.Errorf(t, err, "Should have failed to find secret: %v", err)

	// Delete the test secret.
	err = secrets.Delete("test-secret-namespace-1", "test-secret-name-1")
	assert.NoErrorf(t, err, "Can't find secret: %v", err)

	// Retry delete the test secret, should fail.
	err = secrets.Delete("test-secret-namespace-1", "test-secret-name-1")
	assert.Errorf(t, err, "Should have failed to delete: %v", err)
}

// Test updating a secret.
func TestKubeSecrets_Update(t *testing.T) {
	connections := GetManagedClusterConnections()
	connection := connections["cluster1"]

	secrets := KubeSecrets{
		namespace:     "test-secret-namespace-1",
		kubeClientSet: connection.KubeClient,
		secretLister:  connection.SecretLister,
	}

	mutable := false

	origSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-secret-namespace-1",
			Name:      "test-secret-name-1"},
		Immutable: &mutable,
		Data:      nil,
		StringData: map[string]string{
			"test-secret-data-key-1": "test-secret-data-value-1",
		},
		Type: "test-secret-type",
	}

	updatedSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-secret-namespace-1",
			Name:      "test-secret-name-1"},
		Immutable: &mutable,
		Data:      nil,
		StringData: map[string]string{
			"test-secret-data-key-1": "test-secret-data-value-2",
		},
		Type: "test-secret-type",
	}

	secret, err := secrets.Create(&origSecret)
	assert.NoErrorf(t, err, "Failed to create secret: %v", err)
	assert.Equal(t, &origSecret, secret)

	secret, err = secrets.Update(&updatedSecret)
	assert.NoError(t, err)
	assert.NotNilf(t, secret, "Invalid secret updated: %v", secret)

	secret, err = secrets.Get("test-secret-name-1")
	assert.NoErrorf(t, err, "Failed to get secret: %v", err)
	pwd := secret.StringData["test-secret-data-key-1"]
	assert.NotNilf(t, pwd, "Invalid password: %v", pwd)
	assert.Equal(t, "test-secret-data-value-2", pwd)
}

// Test listing secrets.
func TestKubeSecrets_List(t *testing.T) {
	connections := GetManagedClusterConnections()
	connection := connections["cluster1"]

	secrets := KubeSecrets{
		namespace:     "test-secret-namespace-1",
		kubeClientSet: connection.KubeClient,
		secretLister:  connection.SecretLister,
	}

	origSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-secret-namespace-1",
			Name:      "test-secret-name-1"},
	}

	origSecret2 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-secret-namespace-1",
			Name:      "test-secret-name-2"},
	}

	var err error
	var list []*corev1.Secret

	//list, err := secrets.List("test-secret-namespace-1", nil)
	assert.NoErrorf(t, err, "Error listing secrets: %v", err)
	assert.Nilf(t, list, "Should have returned a nil secret list: %v", list)

	_, err = secrets.Create(&origSecret1)
	//assert.NoErrorf(t, err, "Create failed: %v", err)
	//list, err = secrets.List("test-secret-namespace-1", nil)
	//assert.NoErrorf(t, err, "Error listing secrets: %v", err)
	//assert.NotNilf(t, list, "Invalid secret list: %v", err)
	//assert.Len(t, list, 1, "List should have one secret.")
	//assert.Equal(t, "test-secret-name-1", list[0].Name)

	_, err = secrets.Create(&origSecret2)
	assert.NoErrorf(t, err, "Create failed: %v", err)
	list, err = secrets.List("test-secret-namespace-1", nil)
	assert.NoErrorf(t, err, "Error listing secrets: %v", err)
	assert.NotNilf(t, list, "Invalid secret list: %v", err)
	assert.Len(t, list, 2, "List should have two secrets.")
	assert.NotSame(t, list[0], list[1], "Should be different instances.")
	assert.NotEqual(t, list[0].Name, list[1].Name, "Should have different names.")
}

// Test getting the VMI password secret
func TestKubeSecrets_GetVmiPassword(t *testing.T) {
	connections := GetManagedClusterConnections()
	connection := connections["cluster1"]

	secrets := KubeSecrets{
		namespace:     "test-secret-namespace-1",
		kubeClientSet: connection.KubeClient,
		secretLister:  connection.SecretLister,
	}

	vmiSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-secret-namespace-1",
			Name:      constants.VmiSecretName},
		Data: map[string][]byte{"password": []byte("test-secret-password-1")},
	}

	pwd, err := secrets.GetVmiPassword()
	assert.Equal(t, "", pwd, fmt.Sprintf("Should have retured empty vmi password: %v", pwd))
	assert.Errorf(t, err, fmt.Sprintf("Should have failed getting vmi password: %v", err))

	_, err = secrets.Create(&vmiSecret)
	assert.NoErrorf(t, err, "Error creating vmi password: %v", err)
	pwd, err = secrets.GetVmiPassword()
	assert.NoErrorf(t, err, "Error getting vmi password: %v", err)
	assert.Equal(t, "test-secret-password-1", pwd)
}
