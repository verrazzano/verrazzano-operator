// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"encoding/base64"
	"math/rand"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/local"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const wlsRuntimeEncryptionSecret = "wlsRuntimeEncryptionSecret"

// CreateSecrets will go through a VerrazzanoLocation and find all of the secrets that are needed by
// components, and it will then check if those secrets exist in the correct namespaces and clusters,
// and then update or create them as needed
func CreateSecrets(vzLocation *types.VerrazzanoLocation, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec monitoring.Secrets) error {
	zap.S().Debugf("Creating/updating Secrets for VerrazzanoBinding %s", vzLocation.Location.Name)

	filteredConnections, err := GetFilteredConnections(vzLocation, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct secret for each ManagedCluster
	for clusterName, managedClusterObj := range vzLocation.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var secrets []*corev1.Secret
		if vzLocation.Location.Name == constants.VmiSystemBindingName {
			secrets = monitoring.GetSystemSecrets(sec)
		} else {
			secrets = newSecrets(vzLocation, managedClusterObj, kubeClientSet)
		}

		// Create/Update Namespace
		err := createSecrets(vzLocation.Location, managedClusterConnection, secrets, clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createSecrets(binding *types.ResourceLocation, managedClusterConnection *util.ManagedClusterConnection, newSecrets []*corev1.Secret, clusterName string) error {
	// Create or update secrets
	var secretNames = []string{}
	for _, newSecret := range newSecrets {
		secretNames = append(secretNames, newSecret.Name)
		existingSecret, err := managedClusterConnection.SecretLister.Secrets(newSecret.Namespace).Get(newSecret.Name)
		if existingSecret != nil {
			if existingSecret.Type != wlsRuntimeEncryptionSecret {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingSecret, newSecret)
				if specDiffs != "" {
					zap.S().Debugf("Secret %s : Spec differences %s", newSecret.Name, specDiffs)
					zap.S().Infof("Updating secret %s:%s in cluster %s", newSecret.Namespace, newSecret.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.CoreV1().Secrets(newSecret.Namespace).Update(context.TODO(), newSecret, metav1.UpdateOptions{})
				}
			}
		} else {
			zap.S().Infof("Creating secret %s:%s in cluster %s", newSecret.Namespace, newSecret.Name, clusterName)
			_, err = managedClusterConnection.KubeClient.CoreV1().Secrets(newSecret.Namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Constructs the necessary Secrets for the specified ManagedCluster in the given VerrazzanoBinding
// note that the actual secret data is kept in a secret in the management cluster
func newSecrets(vzLocation *types.VerrazzanoLocation, managedCluster *types.ManagedCluster, kubeClientSet kubernetes.Interface) []*corev1.Secret {
	var secrets []*corev1.Secret

	for namespace, secretNames := range managedCluster.Secrets {
		for _, secretName := range secretNames {
			found := false
			// Filter out duplicate secrets
			for _, secret := range secrets {
				if secret.Name == secretName && secret.Namespace == namespace {
					found = true
					break
				}
			}
			if !found {
				secretObj, err := newSecret(secretName, namespace, kubeClientSet, nil, nil)
				if err != nil {
					continue
				}
				secrets = append(secrets, secretObj)
			}
		}
	}

	return secrets
}

func newSecret(secretName string, namespace string, kubeClientSet kubernetes.Interface, data map[string][]byte, labels map[string]string) (*corev1.Secret, error) {

	var secretInMgmtCluster *corev1.Secret
	secretInMgmtCluster, err := local.GetSecret(secretName, constants.DefaultNamespace, kubeClientSet)
	if err != nil {
		// Check verrazzano system namespace if we don't find the secret in the default namespace since
		// VMI binding secrets are located in the verrazzano system namespace.
		secretInMgmtCluster, err = local.GetSecret(secretName, constants.VerrazzanoNamespace, kubeClientSet)
		if err != nil {
			return nil, err
		}
	}

	// Get the data from the secret in the management cluster and combine it with the given data (if any)
	newData := make(map[string][]byte)
	for k, v := range secretInMgmtCluster.Data {
		newData[k] = v
	}
	if data != nil {
		for k, v := range data {
			newData[k] = v
		}
	}

	secretObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: newData,
		Type: secretInMgmtCluster.Type,
	}
	return secretObj, nil
}

// generateRandomString returns a base64 encoded generated random string.
func generateRandomString() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
