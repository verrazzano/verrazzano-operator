// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package secrets

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"

	k8sTypes "k8s.io/apimachinery/pkg/types"

	wlscrd "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/local"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"

	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	asserts "github.com/stretchr/testify/assert"
)

// TestCreateSecretWhereSecretDidNotAlreadyExist tests creating a new secret where the ID was
// not already used.
func TestCreateSecretWhereSecretDidNotAlreadyExist(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN the information used to create a new secret.
	testSecretCreate := newV8oSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-id-1", "test-secret-type-1", map[string]string{"test-secret-data-key-1": "test-secret-data-value-1"})

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to create a new secret.
	reader, _ := newJSONByteReaderFromObject(testSecretCreate)
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", CreateSecret)

	// THEN check the HTTP response code.
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP success code.")
	// THEN check the content of the returned secret.
	var retSecret Secret
	err := json.Unmarshal(recorder.Body.Bytes(), &retSecret)
	assert.NoError(err, "Should have been able to parse the returned secret.")
	assert.Equal("test-secret-type-1", retSecret.Type, "Type of created secret should match the create value.")
	assert.Equal("test-secret-id-1", retSecret.ID, "ID of created secret should match the create value.")
	assert.Equal("test-secret-name-1", retSecret.Name, "Name of created secret should match the create value.")
	assert.Len(retSecret.Data, 1, "Created secret should have one data element.")
	assert.Equal("test-secret-data-key-1", retSecret.Data[0].Name, "Created secret data key should match create value.")
	assert.Equal("test-secret-data-value-1", retSecret.Data[0].Value, "Created secret data value should match create value.")

	// THEN check the content of the kubernetes secret.
	kubSecret, err := local.GetSecret("test-secret-name-1", constants.DefaultNamespace, *listerSet.KubeClientSet)
	assert.NoError(err, "Created secret should have been found in kubernetes.")
	assert.Equal("test-secret-name-1", kubSecret.Name, "Created secret name in kubernetes should match create value.")
	assert.Contains(kubSecret.Data, "test-secret-data-key-1", "Created secret data in kubernetes should contain create data key.")
	assert.Equal("test-secret-data-value-1", string(kubSecret.Data["test-secret-data-key-1"]), "Created secret data in kubernetes should contain create data value.")
}

// TestCreateSecretWhereSecretAlreadyExisted tests attempting to create a secrete with the same
// ID as a secret that already exists.
func TestCreateSecretWhereSecretAlreadyExisted(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	existingSecret := newK8sSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-uid-1", "generic", nil)
	existingSecret.Data = map[string][]byte{
		"test-secret-data-key-1": encodeStringToBase64Bytes("test-secret-data-value-1"),
	}
	testSecretCreate := newV8oSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-id-1", "generic", map[string]string{"test-secret-data-key-2": "test-secret-data-value-2"})

	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &existingSecret, metav1.CreateOptions{})
	assert.NoError(err)

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to create a secret with the same ID as an existing secret
	reader, _ := newJSONByteReaderFromObject(testSecretCreate)
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", CreateSecret)

	// THEN check the HTTP response code and body.
	assert.Equal(http.StatusConflict, recorder.Code, "Should have return a HTTP conflict response code.")
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-name-1", "Should have returned a response containing the secret ID.")

	// THEN check the initial secret in kubernetes is unmodified.
	kubSecret, err := local.GetSecret("test-secret-name-1", constants.DefaultNamespace, *listerSet.KubeClientSet)
	assert.NoError(err, "Should have been able to get the original secret.")
	assert.Equal("test-secret-name-1", kubSecret.Name, "Original secret name should not have changed.")
	assert.Contains(kubSecret.Data, "test-secret-data-key-1", "Original secret data key should not have changed.")
	value, err := decodeBase64BytesToString(kubSecret.Data["test-secret-data-key-1"])
	assert.NoError(err, "Original secret data value should still be parsable.")
	assert.Equal("test-secret-data-value-1", value, "Original secret data value should not have changed.")
}

// TestCreateSecretWithInvalidJson test attempting to create a secret using an invalid JSON payload.
func TestCreateSecretWithInvalidJson(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to create a secret using an invalid JSON payload
	reader := strings.NewReader("invalid-playload")
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", CreateSecret)

	// THEN confirm that the correct HTTP status code was returned.
	assert.Equal(http.StatusBadRequest, recorder.Code, "Should have returned a HTTP bad request status code.")
}

// TestCreateSecretWithValidDockerSecret tests creating a valid docker secret.
func TestCreateSecretWithValidDockerSecret(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// GIVEN creation details for a valid docker registry secret.
	dockerSecret := newV8oSecret("test-secret-name-1", "test-namespace-1", "test-secret-id-1", "docker-registry", nil)
	dockerSecret.DockerRegistry = DockerRegistry{
		Username: "test-docker-username-1",
		Password: "test-docker-password-1",
		Email:    "test-docker-email-1",
		Server:   "test-docker-server-1",
	}

	// WHEN a request is made to create a valid docker registry secret
	reader, _ := newJSONByteReaderFromObject(dockerSecret)
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", CreateSecret)

	// THEN verify the HTTP response
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP OK code.")
	var retSecret Secret
	err := json.Unmarshal(recorder.Body.Bytes(), &retSecret)
	assert.NoError(err, "Should have returned a parsable JSON body.")
	assert.Equal(dockerSecret, retSecret, "Should have returned a JSON body matching to request body.")

	// THEN check the content of the kubernetes secret.
	kubSecret, err := local.GetSecret("test-secret-name-1", constants.DefaultNamespace, *listerSet.KubeClientSet)
	assert.NoError(err, "Should not have failed getting the new secret.")
	assert.Equal("test-secret-name-1", kubSecret.Name, "New secret name should match the create value.")
	assert.Equal(
		`{"auths":{"test-docker-server-1":{"Username":"test-docker-username-1","Password":"test-docker-password-1","Email":"test-docker-email-1"}}}`,
		string(kubSecret.Data[".dockerconfigjson"]),
		"New secret data should match the create values.")
}

// TestUpdateSecretWhereSecretAlreadyExists tests updating a secret that already exists.
func TestUpdateSecretWhereSecretAlreadyExists(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN a valid secret with some existing data.
	testSecret1 := newK8sSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-uid-1", "generic", nil)
	testSecret1.Data = map[string][]byte{
		"test-secret-data-key-1": encodeStringToBase64Bytes("test-secret-data-value-1"),
	}

	// GIVEN updates to a secret with different data.
	testSecretUpdate := newV8oSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-id-1", "generic", map[string]string{"test-secret-data-key-2": "test-secret-data-value-2"})
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err, "Failed to create test secret in kubernetes.")

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a valid request is made to update a valid secret.
	reader, _ := newJSONByteReaderFromObject(testSecretUpdate)
	request, _ := http.NewRequest("PATCH", "/secret/test-secret-uid-1", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secret/{id}", UpdateSecret)

	// THEN test that the http response code is success
	assert.Equal(http.StatusOK, recorder.Code)

	// THEN test the secret was updated in kubernetes correctly.
	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, "test-secret-uid-1")
	assert.NoError(err, "Failed to get updated secret.")
	assert.True(found, "Updated secret not found.")
	assert.Contains(kubSecret.Data, "test-secret-data-key-2", "Found secret data should have an updated key.")
	assert.Equal("test-secret-data-value-2", string(kubSecret.Data["test-secret-data-key-2"]), "Found secret data should have an updated value.")
}

// TestUpdateSecretWhereSecretDoesNotExists tests attempting to update a secret that does not exist.
func TestUpdateSecretWhereSecretDoesNotExists(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an update to a non-existent secret.
	testSecretUpdate := newV8oSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-id-1", "generic", map[string]string{"test-secret-data-key-2": "test-secret-data-value-2"})

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to update a non-existent secret
	reader, _ := newJSONByteReaderFromObject(testSecretUpdate)
	request, _ := http.NewRequest("PATCH", "/secret/test-secret-uid-1", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secret/{id}", UpdateSecret)

	// THEN test the http response code and body is correct.
	assert.Equal(http.StatusNotFound, recorder.Code, "Should return a HTTP not foudn status.")
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-uid-1", "Should return a body containing the attempted secret id.")

	// THEN test that the update was not applied to the kubernetes secrets.
	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, "test-secret-uid-1")
	assert.Nil(kubSecret, "Should not find the invalid secret.")
	assert.NoError(err, "Should not fail while getting the invalid secret.")
	assert.False(found, "Should not have found the invalid secret.")
}

// TestUpdateSecretWithInvalidJson tests attempting to update a secret using invalid json.
func TestUpdateSecretWithInvalidJson(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN an update request is made with an invalid payload
	reader := strings.NewReader("invalid-payload")
	request, _ := http.NewRequest("PATCH", "/secret/test-secret-uid-1", reader)
	recorder := testutil.InvokeHTTPHandler(request, "/secret/{id}", UpdateSecret)

	// THEN test that and HTTP bad request response is returned.
	assert.Equal(http.StatusBadRequest, recorder.Code, "Should have returned a HTTP bad request.")
}

// TestDeleteSecretWhereSecretExists tests validly deleting an existing secret.
func TestDeleteSecretWhereSecretExists(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an existing kubernetes secret.
	testSecret1 := newK8sSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-uid-1", "generic", nil)
	testSecret1.Data = map[string][]byte{
		"test-secret-data-key-1": encodeStringToBase64Bytes("test-secret-data-value-1"),
	}
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err, "Failed to create given test secret.")

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request to delete an existing secret is made
	request, _ := http.NewRequest("DELETE", "/secret/test-secret-uid-1", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secret/{id}", DeleteSecret)

	// THEN confirm that HTTP response code is correct.
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP OK response code.")

	// THEN confirm that the secret was deleted from kubernetes.
	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, "test-secret-uid-1")
	assert.NoError(err, "Should not return an looking for a non-existent secret.")
	assert.False(found, "Should not find an non-existing secret.")
	assert.Nil(kubSecret, "Should return nil for a non-existing secret.")
}

// TestDeleteSecretWhereSecretDoesNotExists tests attempting to delete a secret that does not exist.
func TestDeleteSecretWhereSecretDoesNotExists(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request do delete a non-existing secret is made
	request, _ := http.NewRequest("DELETE", "/secret/test-secret-uid-1", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secret/{id}", DeleteSecret)

	// THEN verify the response status code.
	assert.Equal(http.StatusNotFound, recorder.Code, "Should have returned a HTTP not found status code.")
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-uid-1", "Response should contain the secret ID used.")
}

// TestReturnAllSecretsWhenThereAreNoSecrets tests getting all secrets when there are none.
func TestReturnAllSecretsWhenThereAreNoSecrets(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", ReturnAllSecrets)

	// THEN validate the HTTP response
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP OK status code.")
	assert.Equal("0", recorder.Header().Get("X-Total-Count"), "Should have returned a header indicating zero count.")
	assert.Equal("[]", strings.TrimSpace(string(recorder.Body.Bytes())), "Should have returned an empty JSON array body.")
}

// TestReturnAllSecretsWhereModelHasWebLogicSecrets tests listing all secrets when the model
// contains WebLogic secrets.
func TestReturnAllSecretsWhereModelHasWebLogicSecrets(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()

	// GIVEN a secret that will be used as a WLS image pull secret.
	wlsPullSecret := newK8sSecret("test-wls-pull-secret-name-1", constants.DefaultNamespace, "test-wls-pull-secret-id-1", "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &wlsPullSecret, metav1.CreateOptions{})
	assert.NoError(err, "Failed creating the test WLS pull secret.")

	// GIVEN a secret that will be used as a config override secret.
	wlsConfSecret := newK8sSecret("test-wls-conf-secret-name-1", constants.DefaultNamespace, "test-wls-conf-secret-id-1", "generic", nil)
	_, err = clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &wlsConfSecret, metav1.CreateOptions{})
	assert.NoError(err, "Failed creating the WLS config override secret.")

	// GIVEN a secret that will be used as a credentials secret.
	wlsCredSecret := newK8sSecret("test-wls-cred-secret-name-1", constants.DefaultNamespace, "test-wls-cred-secret-id-1", "generic", nil)
	_, err = clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &wlsCredSecret, metav1.CreateOptions{})
	assert.NoError(err, "Failed creating the WLS credentials secret.")

	// GIVEN a model binding pair containing a WLS image pull secret, config override secret and credentials secret.
	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": {
			Model: &v1beta1.VerrazzanoModel{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-model-name-1",
					UID:       "test-model-uid-1",
				},
				Spec: v1beta1.VerrazzanoModelSpec{
					WeblogicDomains: []v1beta1.VerrazzanoWebLogicDomain{{
						DomainCRValues: wlscrd.DomainSpec{
							ConfigOverrideSecrets: []string{wlsConfSecret.Name},
							ImagePullSecrets: []corev1.LocalObjectReference{{
								Name: wlsPullSecret.Name,
							}},
							WebLogicCredentialsSecret: corev1.SecretReference{
								Namespace: wlsCredSecret.Namespace,
								Name:      wlsCredSecret.Name,
							},
						},
					}},
				},
			},
		},
	}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", ReturnAllSecrets)

	// THEN validate the HTTP response.
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned HTTP OK status code.")
	secrets := []Secret{}
	err = json.Unmarshal(recorder.Body.Bytes(), &secrets)
	assert.NoError(err, "Should have been able to parse HTTP response body.")
	find := map[string]int{wlsPullSecret.Name: -1, wlsConfSecret.Name: -1, wlsCredSecret.Name: -1}
	for i, s := range secrets {
		find[s.Name] = i
	}
	assert.Equal(newV8oSecretFromK8sSecret(wlsPullSecret), secrets[find[wlsPullSecret.Name]], "Returned WLS image pull secret does not match original.")
	assert.Equal(newV8oSecretFromK8sSecret(wlsConfSecret), secrets[find[wlsConfSecret.Name]], "Returned WLS config override secret does not match original.")
	assert.Equal(newV8oSecretFromK8sSecret(wlsCredSecret), secrets[find[wlsCredSecret.Name]], "Returned WLS credentials secret does not match original.")
	assert.Len(secrets, 3, "Returned secrets array list is incorrect.")
}

// TestReturnAllSecretsWhereModelHasCoherenceSecret tests listing all secrets when the model
// contains Coherence secrets.
func TestReturnAllSecretsWhereModelHasCoherenceSecret(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()

	// GIVEN a secret that will be used as a Coherence image pull secret.
	cohPullSecret := newK8sSecret("test-coh-pull-secret-name-1", constants.DefaultNamespace, "test-coh-pull-secret-id-1", "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &cohPullSecret, metav1.CreateOptions{})
	assert.NoError(err, "Should have been able to create coherence image pull secret.")

	// GIVEN a model binding pair containing a Coherence image pull secret
	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": {
			Model: &v1beta1.VerrazzanoModel{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-model-name-1",
					UID:       "test-model-uid-1",
				},
				Spec: v1beta1.VerrazzanoModelSpec{
					CoherenceClusters: []v1beta1.VerrazzanoCoherenceCluster{{
						ImagePullSecrets: []corev1.LocalObjectReference{{
							Name: cohPullSecret.Name,
						}},
					}},
				},
			},
		},
	}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", ReturnAllSecrets)

	// THEN validate the HTTP response.
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP OK status.")
	ret := []Secret{}
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err, "Returned JSON should have been parsable.")
	assert.Equal(newV8oSecretFromK8sSecret(cohPullSecret), ret[0], "Returned JSON should have matched original secret.")
	assert.Len(ret, 1, "Returned JSON array should contain a single secret.")
}

// TestReturnAllSecretsWhereModelHasHelidonSecret tests listing all secrets when the model
// contains Helidon secrets.
func TestReturnAllSecretsWhereModelHasHelidonSecret(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()

	// GIVEN a secret that will be used as a Helidon image pull secret.
	helPullSecret := newK8sSecret("test-hel-pull-secret-name-1", constants.DefaultNamespace, "test-hel-pull-secret-id-1", "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &helPullSecret, metav1.CreateOptions{})
	assert.NoError(err)

	// GIVEN a model binding pair containing a Helidon image pull secret.
	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": {
			Model: &v1beta1.VerrazzanoModel{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-model-name-1",
					UID:       "test-model-uid-1",
				},
				Spec: v1beta1.VerrazzanoModelSpec{
					HelidonApplications: []v1beta1.VerrazzanoHelidon{{
						ImagePullSecrets: []corev1.LocalObjectReference{{
							Name: helPullSecret.Name,
						}},
					}},
				},
			},
		},
	}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets.
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", ReturnAllSecrets)

	// THEN validate the content of the HTTP response.
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP OK status.")
	ret := []Secret{}
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err, "Should have been able to parse response JSON body.")
	assert.Equal(newV8oSecretFromK8sSecret(helPullSecret), ret[0], "Secret returned should have matched original.")
	assert.Len(ret, 1, "Should have returned a single secret.")
}

// TestReturnAllSecretsWhereBindingHasDatabaseSecret tests listing all secrets when the binding
// contains a database secret.
func TestReturnAllSecretsWhereBindingHasDatabaseSecret(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()

	// GIVEN a secret that will be used as a database secret.
	dbSecret := newK8sSecret("test-db-secret-name-1", constants.DefaultNamespace, "test-db-secret-id-1", "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &dbSecret, metav1.CreateOptions{})
	assert.NoError(err)

	// GIVEN a model binding pair containing a database secret.
	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": {
			Binding: &v1beta1.VerrazzanoBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-binding-name-1",
					UID:       "test-binding-uid-1",
				},
				Spec: v1beta1.VerrazzanoBindingSpec{
					DatabaseBindings: []v1beta1.VerrazzanoDatabaseBinding{{
						Credentials: dbSecret.Name,
					}},
				},
			},
		},
	}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets", ReturnAllSecrets)

	// THEN verify the HTTP response
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned an HTTP OK status code.")
	ret := []Secret{}
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err, "Should have returned a parsable JSON body.")
	assert.Equal(newV8oSecretFromK8sSecret(dbSecret), ret[0], "Should have returned a secret matching original.")
	assert.Len(ret, 1, "Should have returned a single secret.")
}

// TestReturnSingleSecretWithValidKey tests listing secrets using a valid secret ID as a filter.
func TestReturnSingleSecretWithValidKey(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()

	// GIVEN a secret that will be used as a database secret.
	dbSecret := newK8sSecret("test-db-secret-name-1", constants.DefaultNamespace, "test-db-secret-id-1", "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &dbSecret, metav1.CreateOptions{})
	assert.NoError(err)

	// GIVEN a model binding pair containing a database secret.
	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": {
			Binding: &v1beta1.VerrazzanoBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-binding-name-1",
					UID:       "test-binding-uid-1",
				},
				Spec: v1beta1.VerrazzanoBindingSpec{
					DatabaseBindings: []v1beta1.VerrazzanoDatabaseBinding{{
						Credentials: dbSecret.Name,
					}},
				},
			},
		},
	}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets
	request, _ := http.NewRequest("GET", "/secrets/test-db-secret-id-1", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets/{id}", ReturnSingleSecret)

	// THEN verify the HTTP response
	assert.Equal(http.StatusOK, recorder.Code, "Should have returned a HTTP OK status code.")
	ret := []Secret{}
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err, "Should have returned a parsable JSON body.")
	assert.Equal(newV8oSecretFromK8sSecret(dbSecret), ret[0], "Should have returned a secret matching original.")
	assert.Len(ret, 1, "Should have returned a single secret.")
}

// TestReturnSingleSecretWithInvalidKey tests listing secrets using an invalid secret ID as a filter.
func TestReturnSingleSecretWithInvalidKey(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to list all secrets when an invalid secret ID.
	request, _ := http.NewRequest("GET", "/secrets/invalid-secret-id", nil)
	recorder := testutil.InvokeHTTPHandler(request, "/secrets/{id}", ReturnSingleSecret)

	// THEN validate the HTTP response.
	assert.Equal(http.StatusNotFound, recorder.Code, "Should have returned a HTTP note found status code.")
}

// TestAddSecret tests adding secret names to a secret name array.
func TestAddSecret(t *testing.T) {
	assert := asserts.New(t)

	// Test a nil input array.
	outputSecrets := addSecret(nil, "test-secret-1")
	assert.NotNil(outputSecrets, "Expect the returned name array not be nil.")
	assert.Contains(outputSecrets, "test-secret-1", "Expect the returned name array contain the added secret.")
	assert.Len(outputSecrets, 1, "Expect that the returned name array contain a single element.")

	// Test adding a new secret name to an existing secret name array.
	outputSecrets = addSecret([]string{"test-secret-1"}, "test-secret-2")
	assert.NotNil(outputSecrets, "Expect the returned name array not be nil.")
	assert.Contains(outputSecrets, "test-secret-1", "Expect the returned name array contains the initial secret name.")
	assert.Contains(outputSecrets, "test-secret-2", "Expect the returned name array contains the new secret name.")
	assert.Len(outputSecrets, 2, "Expect the returned name array contains two elements.")

	// Test trying to add an existing secret name to an existing secret name array.
	outputSecrets = addSecret([]string{"test-secret-1"}, "test-secret-1")
	assert.NotNil(outputSecrets, "The returned name array should not be nil.")
	assert.Contains(outputSecrets, "test-secret-1", "The returned name array should contain the initial secret name.")
	assert.Len(outputSecrets, 1, "The returned name array should contain a single element.")
}

// TestAddSecretsWithInvalidSecretId tests adding an invalid secret name to the cached secrets.
// Specifically test that addSecrets does not add a secret to the cached secret if that
// secret does not exist in Kubernetes.
func TestAddSecretsWithInvalidSecretId(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN an empty initial state
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN attempting to add a secret name that is invalid because it is not found among the
	// kubernetes secrets
	addSecrets([]string{"invalid-secret-id"})

	// THEN verify a secret is not added to the cached secrets
	assert.Len(cachedSecrets, 0, "The cached secrets should still be empty.")
}

// TestBuildSecretData tests the utility function that builds the string to byte array map used to
// hold secret data within a Kubernetes secret.
func TestBuildSecretData(t *testing.T) {
	assert := asserts.New(t)

	// Test buildSecretData when nil secret contains nil for data.
	secret := newV8oSecret("", "", "", "generic", nil)
	data := buildSecretData(secret)
	assert.NotNil(data, "Build secret data should not be nil")
	assert.Len(data, 0, "Secret data should be empty.")

	// Test buildSecretData when secret  contains basic string name value pair.
	secret = newV8oSecret("test-secret-name-1", "test-namespace-1", "test-secret-id-1", "generic", map[string]string{"test-data-1": "test-value-1"})
	data = buildSecretData(secret)
	assert.NotNil(data, "Build secret data should not be nil.")
	assert.Len(data, 1, "Secret data should have one value.")
	assert.Equal([]byte("test-value-1"), data["test-data-1"], "Resulting secret data should match input data.")

	// Test buildSecretData when secret contains basic docker registry information.
	secret = newV8oSecret("test-secret-name-1", "test-namespace-1", "test-secret-id-1", "docker-registry", nil)
	secret.DockerRegistry = DockerRegistry{
		Username: "test-username-1",
		Password: "test-password-1",
		Email:    "test-email-1",
		Server:   "test-server-1",
	}
	data = buildSecretData(secret)
	assert.NotNil(data, "Build secret data should not be nil")
	assert.Len(data, 1, "Secret data should have one value.")
	assert.Equal(
		[]byte(`{"auths":{"test-server-1":{"Username":"test-username-1","Password":"test-password-1","Email":"test-email-1"}}}`),
		data[".dockerconfigjson"],
		"Resulting docker config json should match given docker information.")
}

// TestGetSecretLogErrorValidSecretUid ensures the utility function getSecretLogError correctly
// retrieves an existing secret with a valid uid.
func TestGetSecretLogErrorValidSecretUid(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs and managed clusters.
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN a valid secret exists in Kubernetes.
	testSecret1 := newK8sSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-id-1", "generic", nil)
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err, "Failed to create valid test secret.")

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made for a valid secret.
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		actualSecret := getSecretLogError(w, "test-secret-id-1")
		// THEN verify the retrieved secret is correct.
		assert.Equal(testSecret1, *actualSecret, "Retrieved secret does not match expected secret.")
	})
	request, _ := http.NewRequest("GET", "/secret", nil)
	router.ServeHTTP(recorder, request)
	// THEN verify the retrieved secret is correct.
	assert.Equal(http.StatusOK, recorder.Code, "Should have resulted in HTTP success status.")
}

// TestGetSecretLogErrorInvalidSecretUid ensures the utility function getSecretLogError correctly
// handles attempting to retrieve a secret with an invalid/non-existent id.
func TestGetSecretLogErrorInvalidSecretUid(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN empty model binding pairs and managed clusters.
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	// GIVEN a valid secret exists in Kubernetes.
	testSecret1 := newK8sSecret("test-secret-name-1", constants.DefaultNamespace, "test-secret-uid-1", "generic", nil)
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err)

	// GIVEN an initialized set of listers.
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	// WHEN a request is made to get an invalid/missing secret
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		secret := getSecretLogError(w, "test-secret-uid-2")
		// THEN a nil secret should be returned.
		assert.Nil(secret, "Expect returned secret to be nil")
	})
	request, _ := http.NewRequest("GET", "/secret", nil)
	router.ServeHTTP(recorder, request)

	// THEN verify the http response code and body.
	assert.Equal(http.StatusNotFound, recorder.Code, "Should return an HTTP not found status.")
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-uid-2", "Error message should contain the the invalid id.")
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Test utility functions.
///////////////////////////////////////////////////////////////////////////////////////////////////

// newK8sSecret creates a kubernetes secret from the supplied parameters.
// name provides the name for the secret.
// namespace provides the namespace for the secret. Typically constants.DefaultNamespace. Should not be an empty string.
// id provides the id/uid for the secret. Should not be an empty string.
// kind provides the type/king of the secret. Typically "generic" or "docker-registry"
// data provides the data for the secret. May be nil
func newK8sSecret(name string, namespace string, id string, kind string, data map[string]string) corev1.Secret {
	var dd map[string][]byte = nil
	if data != nil {
		dd = make(map[string][]byte, len(data))
		for k, v := range data {
			dd[k] = []byte(v)
		}
	}
	s := corev1.Secret{
		Type: corev1.SecretType(kind),
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       k8sTypes.UID(id),
		},
		Data: dd,
	}
	return s
}

// newK8sSecret creates a Verrazzano API secret from the supplied parameters.
// name provides the name for the secret.
// namespace provides the namespace for the secret. Typically constants.DefaultNamespace. Should not be an empty string.
// id provides the id/uid for the secret. Should not be an empty string.
// kind provides the type/king of the secret. Typically "generic" or "docker-registry"
// data provides the data for the secret. May be nil
func newV8oSecret(name string, namespace string, id string, kind string, data map[string]string) Secret {
	var dd []Data = nil
	if data != nil {
		dd = []Data{}
		for k, v := range data {
			dd = append(dd, Data{Name: k, Value: v})
		}
	}
	s := Secret{
		Type:      kind,
		Namespace: namespace,
		Name:      name,
		ID:        id,
		Data:      dd,
	}
	return s
}

// newK8sSecretFromV8oSecret converts a Verrazzano API secret into a Kubernetes secret.
// This is typically used to prevent the need to create "duplicate" secrets in each test.
// s the Verrazzano API secret to convert
// Return the Kubernetes secret
func newK8sSecretFromV8oSecret(s Secret) corev1.Secret {
	var d map[string][]byte = nil
	if s.Data != nil {
		d := map[string][]byte{}
		for i := range s.Data {
			d[s.Data[i].Name] = []byte(s.Data[i].Value)
		}
	}
	r := corev1.Secret{
		Type: corev1.SecretType(s.Type),
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      s.Name,
			UID:       k8sTypes.UID(s.ID),
		},
		Data: d,
	}
	return r
}

// newV8oSecretFromK8sSecret converts a Kubernetes secret into a Verrazzano API secret.
// This is typically used to prevent the need to create "duplicate" secrets in each test.
// s the Kubernetes secret.
// Return the Verrazzano API secret.
func newV8oSecretFromK8sSecret(s corev1.Secret) Secret {
	var d []Data = nil
	if s.Data != nil {
		d := []Data{}
		for k, v := range s.Data {
			d = append(d, Data{Name: k, Value: string(v)})
		}
	}
	r := Secret{
		Type:      string(s.Type),
		Namespace: s.Namespace,
		Name:      s.Name,
		ID:        string(s.UID),
		Data:      d,
	}
	return r
}

// encodeStringToBase64Bytes encodes a string into an array of base64 encoded characters.
// input the string to be converted to a base64 encoded byte array
// Returns the encoded byte array.
func encodeStringToBase64Bytes(input string) []byte {
	b := make([]byte, base64.StdEncoding.EncodedLen(len(input)))
	base64.StdEncoding.Encode(b, []byte(input))
	return b
}

// decodeBase64BytesToString decodes a byte array containing base64 encoded characters.
// input the byte array containing base64 included characters.
// Returns the decoded string and an error.
func decodeBase64BytesToString(input []byte) (string, error) {
	b := make([]byte, base64.StdEncoding.DecodedLen(len(input)))
	_, err := base64.StdEncoding.Decode(b, input)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// newJSONByteReaderFromObject creates a reader of the bytes created by marshalling the input object to JSON.
// obj the input object to marshal to a JSON byte reader.
// Returns a pointer to a byte reader and an error.
func newJSONByteReaderFromObject(obj interface{}) (*bytes.Reader, error) {
	buf, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(buf)
	return reader, nil
}
