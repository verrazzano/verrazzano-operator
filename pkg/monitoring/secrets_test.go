// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package monitoring

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"golang.org/x/crypto/pbkdf2"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/types"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type MockSecrets struct {
	secrets map[string]*corev1.Secret
}

func (ms *MockSecrets) Get(name string) (*corev1.Secret, error) {
	return ms.secrets[name], nil
}

func (ms *MockSecrets) Create(newSecret *corev1.Secret) (*corev1.Secret, error) {
	ms.secrets[newSecret.Name] = newSecret
	return newSecret, nil
}

func (ms *MockSecrets) Update(newSecret *corev1.Secret) (*corev1.Secret, error) {
	ms.secrets[newSecret.Name] = newSecret
	return newSecret, nil
}

func (ms *MockSecrets) List(ns string, selector labels.Selector) (ret []*corev1.Secret, err error) {
	return []*corev1.Secret{}, nil
}

func (ms *MockSecrets) Delete(ns, name string) error {
	delete(ms.secrets, name)
	return nil
}

func (ms *MockSecrets) GetVmiPassword() (string, error) {
	return GetVmiPassword(ms)
}

func TestExistingVmiSecrets(t *testing.T) {
	binding := types.SyntheticBinding{}
	binding.Name = "TestExistingVmiSecrets"
	existing := NewVmiSecret(&binding)
	secrets := MockSecrets{secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: existing,
	}}
	CreateVmiSecrets(&binding, &secrets)
	sec, _ := GetVmiPassword(&secrets)
	assert.Equal(t, string(existing.Data["password"]), sec, "existing vmi secret")
	assertSaltedHash(t, &secrets)
}

func assertSaltedHash(t *testing.T, secrets Secrets) {
	sec, _ := secrets.Get(constants.VmiSecretName)
	saltString := base64.StdEncoding.EncodeToString(sec.Data["salt"])
	salt, _ := base64.StdEncoding.DecodeString(saltString)
	hash := pbkdf2.Key(sec.Data["password"], salt, 27500, 64, sha256.New)
	hashString := base64.StdEncoding.EncodeToString(sec.Data["hash"])
	assert.Equal(t, hashString, base64.StdEncoding.EncodeToString(hash), "expected SaltedHash")
}

func TestNewVmiRandomPassword(t *testing.T) {
	binding := types.SyntheticBinding{}
	binding.Name = "TestNewVmiRandomPassword"
	secrets := &MockSecrets{secrets: map[string]*corev1.Secret{}}
	CreateVmiSecrets(&binding, secrets)
	sec1, _ := GetVmiPassword(secrets)
	assertSaltedHash(t, secrets)
	secrets.Delete(constants.VerrazzanoNamespace, constants.VmiSecretName)

	CreateVmiSecrets(&binding, secrets)
	sec2, _ := GetVmiPassword(secrets)
	assertSaltedHash(t, secrets)
	secrets.Delete(constants.VerrazzanoNamespace, constants.VmiSecretName)

	CreateVmiSecrets(&binding, secrets)
	sec3, _ := GetVmiPassword(secrets)
	assertSaltedHash(t, secrets)
	secrets.Delete(constants.VerrazzanoNamespace, constants.VmiSecretName)

	assert.NotEqual(t, sec1, sec2, "new random password")
	assert.NotEqual(t, sec3, sec2, "new random password")
	assert.NotEqual(t, sec3, sec1, "new random password")
}

func TestGetSystemSecretsNotManaged(t *testing.T) {
	testPassword := "testPassword"

	mockSecrets := &MockSecrets{secrets: map[string]*corev1.Secret{}}
	mockSecrets.Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VmiSecretName,
			Namespace: constants.VerrazzanoNamespace,
		},
		Data: map[string][]byte{
			"username": []byte(constants.VmiUsername),
			"password": []byte(testPassword),
		},
	})

	clusterInfo := ClusterInfo{}
	systemSecrets := GetSystemSecrets(mockSecrets, clusterInfo)
	assert.Equal(t, 2, len(systemSecrets), "expected number of system secrets")
	assert.Equal(t, "filebeat-secret", systemSecrets[0].Name, "expected secret name")
	assert.Equal(t, constants.VmiUsername, string(systemSecrets[0].Data["username"]), "expected username")
	assert.Equal(t, testPassword, string(systemSecrets[0].Data["password"]), "expected password")
	assert.Equal(t, "journalbeat-secret", systemSecrets[1].Name, "expected secret name")
	assert.Equal(t, constants.VmiUsername, string(systemSecrets[1].Data["username"]), "expected username")
	assert.Equal(t, testPassword, string(systemSecrets[1].Data["password"]), "expected password")
}

func TestGetSystemSecretsManaged(t *testing.T) {
	testUsername := "testUsername"
	testPassword := "testPassword"
	testURL := "testURL"
	testCABundle := "testCABundle"

	mockSecrets := &MockSecrets{secrets: map[string]*corev1.Secret{}}

	clusterInfo := ClusterInfo{ManagedClusterName: "cluster1",
		ElasticsearchURL:      testURL,
		ElasticsearchUsername: testUsername,
		ElasticsearchPassword: testPassword,
		ElasticsearchCABundle: []byte(testCABundle),
	}
	systemSecrets := GetSystemSecrets(mockSecrets, clusterInfo)
	assert.Equal(t, 2, len(systemSecrets), "expected number of system secrets")
	assert.Equal(t, "filebeat-secret", systemSecrets[0].Name, "expected secret name")
	assert.Equal(t, testUsername, string(systemSecrets[0].Data["username"]), "expected username")
	assert.Equal(t, testPassword, string(systemSecrets[0].Data["password"]), "expected password")
	assert.Equal(t, testURL, string(systemSecrets[0].Data["es-url"]), "expected url")
	assert.Equal(t, testCABundle, string(systemSecrets[0].Data["ca-bundle"]), "expected ca bundle")
	assert.Equal(t, "journalbeat-secret", systemSecrets[1].Name, "expected secret name")
	assert.Equal(t, testUsername, string(systemSecrets[1].Data["username"]), "expected username")
	assert.Equal(t, testPassword, string(systemSecrets[1].Data["password"]), "expected password")
	assert.Equal(t, testURL, string(systemSecrets[1].Data["es-url"]), "expected url")
	assert.Equal(t, testCABundle, string(systemSecrets[1].Data["ca-bundle"]), "expected ca bundle")
}
