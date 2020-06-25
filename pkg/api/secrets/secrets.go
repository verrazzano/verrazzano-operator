// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/local"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// This file is very similar to applications.go - please see comments there
// which equally apply to this file
type Secret struct {
	Id             string         `json:"id"`
	Cluster        string         `json:"cluster"`
	Type           string         `json:"type"`
	Name           string         `json:"name"`
	Namespace      string         `json:"namespace"`
	Status         string         `json:"status"`
	Data           []Data         `json:"data,omitempty"`
	DockerRegistry DockerRegistry `json:"dockerRegistry,omitempty"`
}

type Data struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DockerRegistry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Server   string `json:"server"`
}

var (
	Secrets   []Secret
	listerSet controller.Listers
)

func Init(listers controller.Listers) {
	listerSet = listers
	refreshSecrets()
}

func refreshSecrets() {
	// initialize domains as an empty list to avoid json encoding "nil"
	Secrets = []Secret{}

	modelSelector := labels.SelectorFromSet(map[string]string{})
	models, err := (*listerSet.ModelLister).VerrazzanoModels("default").List(modelSelector)
	if err != nil {
		glog.Errorf("Error getting application models: %s", err.Error())
		return
	}

	for _, model := range models {
		var secretNames []string
		for _, domain := range model.Spec.WeblogicDomains {
			for _, pullSecret := range domain.DomainCRValues.ImagePullSecrets {
				secretNames = append(secretNames, pullSecret.Name)
			}
			for _, pullSecret := range domain.DomainCRValues.ConfigOverrideSecrets {
				secretNames = append(secretNames, pullSecret)
			}
			secretNames = append(secretNames, domain.DomainCRValues.WebLogicCredentialsSecret.Name)
		}
		for _, helidon := range model.Spec.HelidonApplications {
			for _, pullSecret := range helidon.ImagePullSecrets {
				secretNames = append(secretNames, pullSecret.Name)
			}
		}
		for _, coherence := range model.Spec.CoherenceClusters {
			for _, pullSecret := range coherence.ImagePullSecrets {
				secretNames = append(secretNames, pullSecret.Name)
			}
		}

		i := 0
		for _, secretName := range secretNames {
			// get the actual secret from the management cluster
			theSecret, err := local.GetSecret(secretName, constants.DefaultNamespace, *listerSet.KubeClientSet)
			if err != nil {
				glog.Warningf("Error getting secret %s in management cluster: %s", secretName, err.Error())
				continue
			}
			if theSecret == nil {
				glog.Warningf("Secret %s not found in management cluster", secretName)
				continue
			}

			Secrets = append(Secrets, Secret{
				Id:        string(theSecret.UID),
				Name:      secretName,
				Namespace: "",
				Cluster:   "",
				Type:      string(theSecret.Type),
				Status:    "NYI",
				Data: func() []Data {

					theData := []Data{}
					for k, v := range theSecret.Data {
						theData = append(theData, Data{
							Name:  k,
							Value: string(v),
						})
					}
					return theData
				}(),
			})
			i++
		}
	}
}

func ReturnAllSecrets(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /secrets")
	refreshSecrets()

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Secrets)), 10))
	json.NewEncoder(w).Encode(Secrets)
}

func ReturnSingleSecret(w http.ResponseWriter, r *http.Request) {
	refreshSecrets()
	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /secrets/" + key)

	for _, secrets := range Secrets {
		if secrets.Id == key {
			json.NewEncoder(w).Encode(secrets)
		}
	}
}

func CreateSecret(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("POST /secrets")

	// unmarshall the secret from the payload
	reqBody, _ := ioutil.ReadAll(r.Body)
	var secret Secret
	err := json.Unmarshal(reqBody, &secret)
	if err != nil {
		msg := fmt.Sprintf("Error: failed to unmarshal json: %s", err.Error())
		glog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	// Make sure the secret doesn't exist
	kSecret, _ := local.GetSecret(secret.Name, constants.DefaultNamespace, *listerSet.KubeClientSet)
	if kSecret != nil {
		msg := fmt.Sprintf("Error: secret %s already exists in default namespace", secret.Name)
		glog.Error(msg)
		http.Error(w, msg, http.StatusConflict)
		return
	}
	newSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: constants.DefaultNamespace,
		},
		// we need to handle both generic and docker-registry secrets here
		// docker-registry secrets require us to create the docker config.json format manually
		Data: func() map[string][]byte {
			var result map[string][]byte = map[string][]byte{}
			if secret.Type == "docker-registry" {
				theData := fmt.Sprintf(
					`{"auths":{"%s":{"Username":"%s","Password":"%s","Email":"%s"}}}`,
					secret.DockerRegistry.Server,
					secret.DockerRegistry.Username,
					secret.DockerRegistry.Password,
					secret.DockerRegistry.Email)
				result[".dockerconfigjson"] = []byte(theData)
			} else {
				for _, data := range secret.Data {
					result[data.Name] = []byte(data.Value)
				}
			}
			return result
		}(),
		Type: func() corev1.SecretType {
			if secret.Type == "docker-registry" {
				return corev1.SecretTypeDockerConfigJson
			} else {
				return corev1.SecretTypeOpaque
			}
		}(),
	}

	// create the secret in the management cluster
	err = local.CreateGenericSecret(newSecret, *listerSet.KubeClientSet)
	if err != nil {
		msg := fmt.Sprintf("Error creating secret %s:%s failed: %s", secret.Namespace, secret.Name, err.Error())
		glog.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(secret)
}

// DeleteSecret will delete a secret identified by the secret Kubernetes UID.
func DeleteSecret(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["id"]
	glog.V(4).Info("DELETE /secrets/" + uid)

	kubSecret := getSecretLogError(w, uid)
	if kubSecret == nil {
		return
	}
	err := local.DeleteSecret(*listerSet.KubeClientSet, kubSecret)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: delete secret with UID %s failed: %s", uid, err.Error()),
			http.StatusInternalServerError)
		return
	}
}

// UpdateSecret will update a secret identified by the secret Kubernetes UID.
func UpdateSecret(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["id"]
	glog.V(4).Info("PATCH /secrets/" + uid)

	// unmarshall the secret from the payload
	reqBody, _ := ioutil.ReadAll(r.Body)
	var secret Secret
	err := json.Unmarshal(reqBody, &secret)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: failed to unmarshal json payload: %s", err.Error()),
			http.StatusBadRequest)
		return
	}
	kubSecret := getSecretLogError(w, uid)
	if kubSecret == nil {
		return
	}
	kubSecret.Data = buildSecretData(secret)
	err = local.UpdateSecret(*listerSet.KubeClientSet, kubSecret)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: update secret with UID %s failed: %s", uid, err.Error()),
			http.StatusInternalServerError)
	}
}

// build the data section of the secret
func buildSecretData(secret Secret) map[string][]byte {
	var result map[string][]byte = map[string][]byte{}
	if secret.Type == "docker-registry" {
		theData := fmt.Sprintf(
			`{"auths":{"%s":{"Username":"%s","Password":"%s","Email":"%s"}}}`,
			secret.DockerRegistry.Server,
			secret.DockerRegistry.Username,
			secret.DockerRegistry.Password,
			secret.DockerRegistry.Email)
		result[".dockerconfigjson"] = []byte(theData)
	} else {
		for _, data := range secret.Data {
			result[data.Name] = []byte(data.Value)
		}
	}
	return result
}

// Get the secret by UID.  Write http response and log error on failure
func getSecretLogError(w http.ResponseWriter, uid string) *corev1.Secret {
	kubSecret, found, err := local.GetSecretByUID(*listerSet.KubeClientSet, constants.DefaultNamespace, uid)
	if err != nil {
		msg := fmt.Sprintf("Error: get secret with UID %s failed: %s", uid, err.Error())
		glog.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return nil
	}
	if !found {
		msg := fmt.Sprintf("Error: secret with UID %s not found: ", uid)
		glog.Error(msg)
		http.Error(w, msg, http.StatusNotFound)
		return nil
	}
	return kubSecret
}
