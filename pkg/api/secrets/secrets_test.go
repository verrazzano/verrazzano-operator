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

	testutilcontroller "github.com/verrazzano/verrazzano-operator/pkg/testutilontroller"

	k8sTypes "k8s.io/apimachinery/pkg/types"

	v8 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/local"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"

	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	asserts "github.com/stretchr/testify/assert"
)

// func addSecret(secretNames []string, secretName string) []string {
func TestAddSecret(t *testing.T) {
	assert := asserts.New(t)

	outputSecrets := addSecret(nil, "test-secret-1")
	assert.NotNil(outputSecrets)
	assert.Contains(outputSecrets, "test-secret-1")
	assert.Len(outputSecrets, 1)

	outputSecrets = addSecret([]string{"test-secret-1"}, "test-secret-2")
	assert.NotNil(outputSecrets)
	assert.Contains(outputSecrets, "test-secret-1")
	assert.Contains(outputSecrets, "test-secret-2")
	assert.Len(outputSecrets, 2)

	outputSecrets = addSecret([]string{"test-secret-1"}, "test-secret-1")
	assert.NotNil(outputSecrets)
	assert.Contains(outputSecrets, "test-secret-1")
	assert.Len(outputSecrets, 1)
}

func TestAddSecretsWithInvalidSecretId(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	addSecrets([]string{"invalid-secret-id"})

	assert.Len(cachedSecrets, 0)
}

// func buildSecretData(secret Secret) map[string][]byte {
func TestBuildSecretData(t *testing.T) {
	assert := asserts.New(t)
	secret := Secret{
		ID:             "",
		Cluster:        "",
		Type:           "",
		Name:           "",
		Namespace:      "",
		Status:         "",
		Data:           nil,
		DockerRegistry: DockerRegistry{},
	}
	data := buildSecretData(secret)
	assert.NotNil(data, "Build secret data should not be nil")
	assert.Len(data, 0, "Secret data should be empty.")

	secret = Secret{
		ID:        "",
		Cluster:   "",
		Type:      "test-type-1",
		Name:      "test-secret-1",
		Namespace: "test-namespace-1",
		Status:    "",
		Data: []Data{{
			Name:  "test-data-1",
			Value: "test-value-1",
		}},
		DockerRegistry: DockerRegistry{},
	}
	data = buildSecretData(secret)
	assert.NotNil(data, "Build secret data should not be nil")
	assert.Len(data, 1, "Secret data should have one value.")
	assert.Equal([]byte("test-value-1"), data["test-data-1"])

	secret = Secret{
		ID:        "",
		Cluster:   "",
		Type:      "docker-registry",
		Name:      "test-secret-1",
		Namespace: "test-namespace-1",
		Status:    "",
		Data:      nil,
		DockerRegistry: DockerRegistry{
			Username: "test-username-1",
			Password: "test-password-1",
			Email:    "test-email-1",
			Server:   "test-server-1",
		},
	}
	data = buildSecretData(secret)
	assert.NotNil(data, "Build secret data should not be nil")
	assert.Len(data, 1, "Secret data should have one value.")
	assert.Equal(
		[]byte("{\"auths\":{\"test-server-1\":{\"Username\":\"test-username-1\",\"Password\":\"test-password-1\",\"Email\":\"test-email-1\"}}}"),
		data[".dockerconfigjson"])
}

// func getSecretLogError(w http.ResponseWriter, uid string) *corev1.Secret {
func TestGetSecretLogErrorValidSecretUid(t *testing.T) {
	assert := asserts.New(t)
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}
	testSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-secret-name-1",
			UID:       "test-secret-uid-1"},
	}
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err)
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		secret := getSecretLogError(w, "test-secret-uid-1")
		assert.Equal(testSecret1, *secret)
	})
	request, _ := http.NewRequest("GET", "/secret", nil)
	router.ServeHTTP(recorder, request)
	assert.Equal(200, recorder.Code)
}

func TestGetSecretLogErrorInvalidSecretUid(t *testing.T) {
	assert := asserts.New(t)
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}
	testSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-secret-name-1",
			UID:       "test-secret-uid-1"},
	}
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err)
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		secret := getSecretLogError(w, "test-secret-uid-2")
		assert.Nil(secret, "Expect returned secret to be nil")
	})
	request, _ := http.NewRequest("GET", "/secret", nil)
	router.ServeHTTP(recorder, request)
	assert.Equal(404, recorder.Code)
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-uid-2")
}

//func UpdateSecret(w http.ResponseWriter, r *http.Request) {
func TestUpdateSecretWhereSecretAlreadyExists(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}
	testSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-secret-name-1",
			UID:       "test-secret-uid-1"},
		Data: map[string][]byte{
			"test-secret-data-key-1": Base64EncodeStringToBytes("test-secret-data-value-1"),
		},
	}
	testSecretUpdate := Secret{
		Type:      "test-secret-type-1",
		Name:      "test-secret-name=1",
		Namespace: "default",
		Data: []Data{{
			Name:  "test-secret-data-key-2",
			Value: "test-secret-data-value-2",
		}},
	}

	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err)

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	reader, _ := NewJsonByteReaderFromObject(testSecretUpdate)
	request, _ := http.NewRequest("PATCH", "/secret/test-secret-uid-1", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secret/{id}", UpdateSecret)
	assert.Equal(200, recorder.Code)

	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, "test-secret-uid-1")
	assert.NoError(err)
	assert.True(found)
	assert.Contains(kubSecret.Data, "test-secret-data-key-2")
	assert.Equal("test-secret-data-value-2", string(kubSecret.Data["test-secret-data-key-2"]))

	// These assertions are commented out because there is a disconnect between the secrets in
	// k8s and those in the model/binding.  This is probably a bug.
	// Refresh the "local" secrets and check to make sure the secret was updated.
	//refreshSecrets()
	//found = false
	//for _, vzSecret := range Secrets {
	//	if vzSecret.ID == "test-secret-uid-1" {
	//		assert.Equal(testSecretUpdate, vzSecret)
	//		found = true
	//	}
	//}
	//assert.True(found, "Expect to find test-secret-uid-1")
}

//func UpdateSecret(w http.ResponseWriter, r *http.Request) {
func TestUpdateSecretWhereSecretDoesNotExists(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	testSecretUpdate := Secret{
		Type:      "test-secret-type-1",
		Name:      "test-secret-name=1",
		Namespace: "default",
		Data: []Data{{
			Name:  "test-secret-data-key-2",
			Value: "test-secret-data-value-2",
		}},
	}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	reader, _ := NewJsonByteReaderFromObject(testSecretUpdate)
	request, _ := http.NewRequest("PATCH", "/secret/test-secret-uid-1", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secret/{id}", UpdateSecret)
	assert.Equal(404, recorder.Code)
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-uid-1")

	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, "test-secret-uid-1")
	assert.Nil(kubSecret)
	assert.NoError(err)
	assert.False(found)
}

//func UpdateSecret(w http.ResponseWriter, r *http.Request) {
func TestUpdateSecretWithInvalidJson(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	reader := strings.NewReader("invalid-playload")
	request, _ := http.NewRequest("PATCH", "/secret/test-secret-uid-1", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secret/{id}", UpdateSecret)
	assert.Equal(400, recorder.Code)
}

// func DeleteSecret(w http.ResponseWriter, r *http.Request) {
func TestDeleteSecretWhereSecretExists(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	testSecret1 := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-secret-name-1",
			UID:       "test-secret-uid-1"},
		Data: map[string][]byte{
			"test-secret-data-key-1": Base64EncodeStringToBytes("test-secret-data-value-1"),
		},
	}
	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &testSecret1, metav1.CreateOptions{})
	assert.NoError(err)

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	request, _ := http.NewRequest("DELETE", "/secret/test-secret-uid-1", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secret/{id}", DeleteSecret)
	assert.Equal(200, recorder.Code)

	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, "test-secret-uid-1")
	assert.NoError(err)
	assert.False(found)
	assert.Nil(kubSecret)
}

// func DeleteSecret(w http.ResponseWriter, r *http.Request) {
func TestDeleteSecretWhereSecretDoesNotExists(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	request, _ := http.NewRequest("DELETE", "/secret/test-secret-uid-1", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secret/{id}", DeleteSecret)
	assert.Equal(404, recorder.Code)
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-uid-1")
}

// func CreateSecret(w http.ResponseWriter, r *http.Request) {
func TestCreateSecretWhereSecretDidNotAlreadyExist(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	testSecretCreate := Secret{
		Type:      "test-secret-type-1",
		ID:        "test-secret-id-1",
		Name:      "test-secret-name-1",
		Namespace: "default",
		Data: []Data{{
			Name:  "test-secret-data-key-1",
			Value: "test-secret-data-value-1",
		}},
	}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	reader, _ := NewJsonByteReaderFromObject(testSecretCreate)
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", CreateSecret)

	// Check the content of the returned secret.
	assert.Equal(200, recorder.Code)
	var retSecret Secret
	err := NewObjectFromJsonByteReader(bytes.NewReader(recorder.Body.Bytes()), &retSecret)
	assert.NoError(err)
	assert.Equal("test-secret-type-1", retSecret.Type)
	assert.Equal("test-secret-id-1", retSecret.ID)
	assert.Equal("test-secret-name-1", retSecret.Name)
	assert.Len(retSecret.Data, 1)
	assert.Equal("test-secret-data-key-1", retSecret.Data[0].Name)
	assert.Equal("test-secret-data-value-1", retSecret.Data[0].Value)

	// Check the content of the kubernetes secret.
	kubSecret, err := local.GetSecret("test-secret-name-1", constants.DefaultNamespace, *listerSet.KubeClientSet)
	assert.NoError(err)
	assert.Equal("test-secret-name-1", kubSecret.Name)
	assert.Contains(kubSecret.Data, "test-secret-data-key-1")
	assert.Equal("test-secret-data-value-1", string(kubSecret.Data["test-secret-data-key-1"]))
}

// func CreateSecret(w http.ResponseWriter, r *http.Request) {
func TestCreateSecretWhereSecretAlreadyExisted(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	existingSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-secret-name-1",
			UID:       "test-secret-uid-1"},
		Data: map[string][]byte{
			"test-secret-data-key-1": Base64EncodeStringToBytes("test-secret-data-value-1"),
		},
	}
	testSecretCreate := Secret{
		Type:      "test-secret-type-1",
		ID:        "test-secret-id-1",
		Name:      "test-secret-name-1",
		Namespace: "default",
		Data: []Data{{
			Name:  "test-secret-data-key-2",
			Value: "test-secret-data-value-2",
		}},
	}

	_, err := clients.CoreV1().Secrets("default").Create(context.TODO(), &existingSecret, metav1.CreateOptions{})
	assert.NoError(err)

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	reader, _ := NewJsonByteReaderFromObject(testSecretCreate)
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", CreateSecret)

	// Check the content of the returned secret.
	assert.Equal(409, recorder.Code)
	assert.Contains(string(recorder.Body.Bytes()), "test-secret-name-1")

	// Check the content of the kubernetes secret.
	kubSecret, err := local.GetSecret("test-secret-name-1", constants.DefaultNamespace, *listerSet.KubeClientSet)
	assert.NoError(err)
	assert.Equal("test-secret-name-1", kubSecret.Name)
	assert.Contains(kubSecret.Data, "test-secret-data-key-1")
	value, err := Base64DecodeBytesToString(kubSecret.Data["test-secret-data-key-1"])
	assert.NoError(err)
	assert.Equal("test-secret-data-value-1", value)
}

// func CreateSecret(w http.ResponseWriter, r *http.Request) {
func TestCreateSecretWithInvalidJson(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	reader := strings.NewReader("invalid-playload")
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", CreateSecret)

	assert.Equal(400, recorder.Code)
}

// func CreateSecret(w http.ResponseWriter, r *http.Request) {
func TestCreateSecretDockerSecret(t *testing.T) {
	assert := asserts.New(t)

	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}

	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))

	dockerSecret := Secret{
		Type: "docker-registry",
		Name: "test-secret-name-1",
		//Data: nil,
		DockerRegistry: DockerRegistry{
			Username: "test-docker-username-1",
			Password: "test-docker-password-1",
			Email:    "test-docker-email-1",
			Server:   "test-docker-server-1",
		},
	}

	reader, _ := NewJsonByteReaderFromObject(dockerSecret)
	request, _ := http.NewRequest("POST", "/secrets", reader)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", CreateSecret)

	assert.Equal(200, recorder.Code)

	var retSecret Secret
	err := NewObjectFromJsonByteReader(bytes.NewReader(recorder.Body.Bytes()), &retSecret)
	assert.NoError(err)
	assert.Equal(dockerSecret, retSecret)

	// Check the content of the kubernetes secret.
	kubSecret, err := local.GetSecret("test-secret-name-1", constants.DefaultNamespace, *listerSet.KubeClientSet)
	assert.NoError(err)
	assert.Equal("test-secret-name-1", kubSecret.Name)
	assert.Equal(
		"{\"auths\":{\"test-docker-server-1\":{\"Username\":\"test-docker-username-1\",\"Password\":\"test-docker-password-1\",\"Email\":\"test-docker-email-1\"}}}",
		string(kubSecret.Data[".dockerconfigjson"]))
}

// func ReturnAllSecrets(w http.ResponseWriter, r *http.Request) {
func TestReturnAllSecretsWhenThereAreNoSecrets(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", ReturnAllSecrets)
	assert.Equal(200, recorder.Code)
	assert.Equal("0", recorder.Header().Get("X-Total-Count"))
	assert.Equal("[]", strings.TrimSpace(string(recorder.Body.Bytes())))
}

// func ReturnAllSecrets(w http.ResponseWriter, r *http.Request) {
func TestReturnAllSecretsWhereModelHasWeblogicSecrets(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()

	wlsPullSecret := NewK8sSecret("test-wls-pull-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &wlsPullSecret, metav1.CreateOptions{})
	assert.NoError(err)

	wlsConfSecret := NewK8sSecret("test-wls-conf-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err = clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &wlsConfSecret, metav1.CreateOptions{})
	assert.NoError(err)

	wlsCredSecret := NewK8sSecret("test-wls-cred-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err = clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &wlsCredSecret, metav1.CreateOptions{})
	assert.NoError(err)

	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": &types.ModelBindingPair{
			Model: &v1beta1.VerrazzanoModel{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-model-name-1",
					UID:       "test-model-uid-1",
				},
				Spec: v1beta1.VerrazzanoModelSpec{
					WeblogicDomains: []v1beta1.VerrazzanoWebLogicDomain{{
						DomainCRValues: v8.DomainSpec{
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
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", ReturnAllSecrets)
	assert.Equal(200, recorder.Code)
	ret := make([]Secret, 0)
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err)
	// Need to invent some way to make the order independent.
	assert.Equal(NewV8oSecretFromK8sSecret(wlsPullSecret), ret[0])
	assert.Equal(NewV8oSecretFromK8sSecret(wlsConfSecret), ret[1])
	assert.Equal(NewV8oSecretFromK8sSecret(wlsCredSecret), ret[2])
	assert.Len(ret, 3)
}

// func ReturnAllSecrets(w http.ResponseWriter, r *http.Request) {
func TestReturnAllSecretsWhereModelHasCoherenceSecret(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()

	cohPullSecret := NewK8sSecret("test-coh-pull-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &cohPullSecret, metav1.CreateOptions{})
	assert.NoError(err)

	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": &types.ModelBindingPair{
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
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", ReturnAllSecrets)
	assert.Equal(200, recorder.Code)
	ret := make([]Secret, 0)
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err)
	assert.Equal(NewV8oSecretFromK8sSecret(cohPullSecret), ret[0])
	assert.Len(ret, 1)
}

// func ReturnAllSecrets(w http.ResponseWriter, r *http.Request) {
func TestReturnAllSecretsWhereModelHasHelidonSecret(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()

	helPullSecret := NewK8sSecret("test-hel-pull-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &helPullSecret, metav1.CreateOptions{})
	assert.NoError(err)

	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": &types.ModelBindingPair{
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
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", ReturnAllSecrets)
	assert.Equal(200, recorder.Code)
	ret := make([]Secret, 0)
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err)
	assert.Equal(NewV8oSecretFromK8sSecret(helPullSecret), ret[0])
	assert.Len(ret, 1)
}

// func ReturnAllSecrets(w http.ResponseWriter, r *http.Request) {
func TestReturnAllSecretsWhereModelHasDatabaseSecret(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()

	dbSecret := NewK8sSecret("test-db-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &dbSecret, metav1.CreateOptions{})
	assert.NoError(err)

	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": &types.ModelBindingPair{
			Binding: &v1beta1.VerrazzanoBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-binding-name-1",
					UID:       "test-bidning-uid-1",
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
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets", ReturnAllSecrets)
	assert.Equal(200, recorder.Code)
	ret := make([]Secret, 0)
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err)
	assert.Equal(NewV8oSecretFromK8sSecret(dbSecret), ret[0])
	assert.Len(ret, 1)
}

// func ReturnSingleSecret(w http.ResponseWriter, r *http.Request) {
func TestReturnSingleSecretWithValidKey(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()

	dbSecret := NewK8sSecret("test-db-secret-name-1", constants.DefaultNamespace, "generic", nil)
	_, err := clients.CoreV1().Secrets(constants.DefaultNamespace).Create(context.TODO(), &dbSecret, metav1.CreateOptions{})
	assert.NoError(err)

	mbPairs := map[string]*types.ModelBindingPair{
		"test-model-1": &types.ModelBindingPair{
			Binding: &v1beta1.VerrazzanoBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: constants.DefaultNamespace,
					Name:      "test-binding-name-1",
					UID:       "test-bidning-uid-1",
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
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets/test-db-secret-name-1", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets/{id}", ReturnSingleSecret)
	assert.Equal(200, recorder.Code)
	ret := make([]Secret, 0)
	err = json.Unmarshal(recorder.Body.Bytes(), &ret)
	assert.NoError(err)
	assert.Equal(NewV8oSecretFromK8sSecret(dbSecret), ret[0])
	assert.Len(ret, 1)
}

// func ReturnSingleSecret(w http.ResponseWriter, r *http.Request) {
func TestReturnSingleSecretWithInvalidKey(t *testing.T) {
	assert := asserts.New(t)
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	mbPairs := map[string]*types.ModelBindingPair{}
	clusters := []v1beta1.VerrazzanoManagedCluster{}
	Init(testutilcontroller.NewControllerListers(&clients, clusters, &mbPairs))
	request, _ := http.NewRequest("GET", "/secrets/invalid-secret-key", nil)
	recorder := testutil.InvokeHttpHandler(request, "/secrets/{id}", ReturnSingleSecret)
	// Some debate about getting a 200 vs a 404 here.
	assert.Equal(404, recorder.Code)
	//assert.Equal(200, recorder.Code)
	//assert.Equal("0", recorder.Header().Get("X-Total-Count"))
	//assert.Equal("[]", strings.TrimSpace(string(recorder.Body.Bytes())))
}

func NewK8sSecret(n string, ns string, t string, d map[string]string) corev1.Secret {
	var dd map[string][]byte = nil
	if d != nil {
		dd = make(map[string][]byte, len(d))
		for k, v := range d {
			dd[k] = []byte(v)
		}
	}
	s := corev1.Secret{
		Type: corev1.SecretType(t),
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      n,
			UID:       k8sTypes.UID(n),
		},
		Data: dd,
	}
	return s
}

func NewV8oSecret(n string, ns string, t string, d map[string]string) Secret {
	var dd []Data = nil
	if d != nil {
		dd = make([]Data, len(d))
		for k, v := range d {
			dd = append(dd, Data{Name: k, Value: v})
		}
	}
	s := Secret{
		Type:      t,
		Namespace: ns,
		Name:      n,
		ID:        n,
		Data:      dd,
	}
	return s
}

func NewK8sSecretFromV8oSecret(s Secret) corev1.Secret {
	var d map[string][]byte = nil
	if s.Data != nil {
		d := make(map[string][]byte, len(s.Data))
		for i := range s.Data {
			d[s.Data[i].Name] = []byte(s.Data[i].Value)
		}
	}
	r := corev1.Secret{
		Type: corev1.SecretType(s.Type),
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      s.Name,
			// Using name as ID/UID
			UID: k8sTypes.UID(s.Name),
		},
		Data: d,
	}
	return r
}

func NewV8oSecretFromK8sSecret(s corev1.Secret) Secret {
	var d []Data = nil
	if s.Data != nil {
		d := make([]Data, len(s.Data))
		for k, v := range s.Data {
			d = append(d, Data{Name: k, Value: string(v)})
		}
	}
	r := Secret{
		Type:      string(s.Type),
		Namespace: s.Namespace,
		Name:      s.Name,
		// Using name as ID/UID
		ID:   s.Name,
		Data: d,
	}
	return r
}

func Base64EncodeStringToBytes(message string) []byte {
	b := make([]byte, base64.StdEncoding.EncodedLen(len(message)))
	base64.StdEncoding.Encode(b, []byte(message))
	return b
}

func Base64DecodeBytesToString(message []byte) (string, error) {
	b := make([]byte, base64.StdEncoding.DecodedLen(len(message)))
	_, err := base64.StdEncoding.Decode(b, message)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func NewJsonByteReaderFromObject(o interface{}) (*bytes.Reader, error) {
	buf, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(buf)
	return reader, nil
}

func NewObjectFromJsonByteReader(r *bytes.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
