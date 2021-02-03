// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

type acmeDNS struct {
	AllowFrom  []string `json:"allowfrom"`
	Fulldomain string   `json:"fulldomain"`
	Password   string   `json:"password"`
	Subdomain  string   `json:"subdomain"`
	Username   string   `json:"username"`
}

// DeleteSecrets deletes secrets for a given binding in the management cluster.
func DeleteSecrets(binding *types.ClusterBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error {
	zap.S().Debugf("Deleting Management Cluster secrets for VerrazzanoBinding %s", binding.Name)

	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})
	existingSecretsList, err := secretLister.Secrets("").List(selector)
	if err != nil {
		return err
	}
	for _, existingSecret := range existingSecretsList {
		zap.S().Infof("Deleting secret %s", existingSecret.Name)
		err := kubeClientSet.CoreV1().Secrets(existingSecret.Namespace).Delete(context.TODO(), existingSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

//
//  ----------------------------------------------------------------------------------------------------
//

//
// Functions for generic secrets that we need to create from the UI/CLI and store in the management
// cluster, these are for things like ImagePullSecrets, where we do not want to put the secret
// data into the model/binding files
//

// CreateGenericSecret will create a secret in the specified cluster - this is intended to
// be used to create secrets in the management cluster
func CreateGenericSecret(newSecret corev1.Secret, kubeClientSet kubernetes.Interface) error {
	zap.S().Infof("Creating Secret %s", newSecret.Name)
	_, err := kubeClientSet.CoreV1().Secrets(newSecret.Namespace).Create(context.TODO(), &newSecret, metav1.CreateOptions{})

	if err != nil {
		zap.S().Errorf("Error creating secret %s:%s - error %s", newSecret.Namespace, newSecret.Name, err.Error())
	}
	return err
}

// GetSecret will retrieve a secret from the specified cluster and namespace.
// This is intended to be used in the management cluster.
func GetSecret(name string, namespace string, kubeClientSet kubernetes.Interface) (*corev1.Secret, error) {
	secret, err := kubeClientSet.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetSecretByUID gets the secret by the Kubernetes UID
func GetSecretByUID(kubeClientSet kubernetes.Interface, ns string, uid string) (*corev1.Secret, bool, error) {
	secretsList, err := kubeClientSet.CoreV1().Secrets(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		zap.S().Errorf("Error %s getting the list of secrets", err.Error())
		return nil, false, err
	}
	for _, secret := range secretsList.Items {
		if string(secret.UID) == uid {
			return &secret, true, nil
		}
	}
	return nil, false, nil
}

// UpdateAcmeDNSSecret updates the AcmeDnsSecret, which is used for "bring your own dns" installs, to contain the DNS
// entries for the model/binding.  This is one of the required steps for the monitoring endpoints to work.
func UpdateAcmeDNSSecret(binding *types.ClusterBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error {
	zap.S().Debugf("Updating Management Cluster secret %s for VerrazzanoBinding %s", name, binding.Name)
	namespace := constants.VerrazzanoNamespace
	acmeDNSKey := constants.AcmeDNSSecretKey

	// Get the cert-manager secret for acms-dns
	secret, err := secretLister.Secrets(namespace).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		// Secret will not exist when not using "bring your own dns"
		return nil
	} else if err != nil {
		return fmt.Errorf("Request to get secret %s in namespace %s failed with error %v", name, namespace, err)
	}

	// Unmarshal the acme dns credentials
	var dnsCredentials map[string]acmeDNS
	acmeDNSData, found := secret.Data[acmeDNSKey]
	if !found {
		return fmt.Errorf("The data in secret %s in namespace %s does not contain the key %s", name, namespace, acmeDNSKey)
	}
	err = json.Unmarshal(acmeDNSData, &dnsCredentials)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal key %s from secret %s in namespace %s, error: %v", acmeDNSKey, name, namespace, err)
	}

	// Make sure the map is not empty
	if len(dnsCredentials) == 0 {
		return fmt.Errorf("The key %s in secret %s in namespace %s does not contain any data", acmeDNSKey, name, namespace)
	}

	// All the records contain the same credentials, obtain the entry keyed by the uri
	dnsCred := dnsCredentials[verrazzanoURI]

	// Check to see if credential records already exist for the binding
	updated := false
	bindingDNSName := fmt.Sprintf("vmi.%s.%s", binding.Name, verrazzanoURI)
	_, found = dnsCredentials[bindingDNSName]
	if !found {
		dnsCredentials[bindingDNSName] = dnsCred
		updated = true
	}

	bindingDNSName = fmt.Sprintf("*.vmi.%s.%s", binding.Name, verrazzanoURI)
	_, found = dnsCredentials[bindingDNSName]
	if !found {
		dnsCredentials[bindingDNSName] = dnsCred
		updated = true
	}

	// Update the secret?
	if updated {
		secretDataEnc, err := json.Marshal(dnsCredentials)
		if err != nil {
			return err
		}

		secret.Data[acmeDNSKey] = secretDataEnc

		_, err = kubeClientSet.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteSecret deletes an existing secret
func DeleteSecret(kubeClientSet kubernetes.Interface, secret *corev1.Secret) error {
	return kubeClientSet.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})
}

// UpdateSecret updates an existing secret
func UpdateSecret(kubeClientSet kubernetes.Interface, secret *corev1.Secret) error {
	_, err := kubeClientSet.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	return err
}
