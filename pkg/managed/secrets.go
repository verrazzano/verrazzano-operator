// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"encoding/base64"
	"math/rand"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/local"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const wlsRuntimeEncryptionSecret = "wlsRuntimeEncryptionSecret"

// CreateSecrets will go through a ModelBindingPair and find all of the secrets that are needed by
// components, and it will then check if those secrets exist in the correct namespaces and clusters,
// and then update or create them as needed
func CreateSecrets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec monitoring.Secrets) error {

	glog.V(6).Infof("Creating/updating Secrets for VerrazzanoBinding %s", mbPair.Binding.Name)

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct secret for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var secrets []*corev1.Secret
		if mbPair.Binding.Name == constants.VmiSystemBindingName {
			secrets = monitoring.GetSystemSecrets(sec)
		} else {
			secrets = newSecrets(mbPair, managedClusterObj, kubeClientSet)
		}

		// Create/Update Namespace
		err := createSecrets(mbPair.Binding, managedClusterConnection, secrets, clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createSecrets(binding *v1beta1v8o.VerrazzanoBinding, managedClusterConnection *util.ManagedClusterConnection, newSecrets []*corev1.Secret, clusterName string) error {
	// Create or update secrets
	var secretNames = []string{}
	for _, newSecret := range newSecrets {
		secretNames = append(secretNames, newSecret.Name)
		existingSecret, err := managedClusterConnection.SecretLister.Secrets(newSecret.Namespace).Get(newSecret.Name)
		if existingSecret != nil {
			if existingSecret.Type != wlsRuntimeEncryptionSecret {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingSecret, newSecret)
				if specDiffs != "" {
					glog.V(6).Infof("Secret %s : Spec differences %s", newSecret.Name, specDiffs)
					glog.V(4).Infof("Updating secret %s:%s in cluster %s", newSecret.Namespace, newSecret.Name, clusterName)
					_, err = managedClusterConnection.KubeClient.CoreV1().Secrets(newSecret.Namespace).Update(context.TODO(), newSecret, metav1.UpdateOptions{})
				}
			}
		} else {
			glog.V(4).Infof("Creating secret %s:%s in cluster %s", newSecret.Namespace, newSecret.Name, clusterName)
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
func newSecrets(mbPair *types.ModelBindingPair, managedCluster *types.ManagedCluster, kubeClientSet kubernetes.Interface) []*corev1.Secret {

	var secrets []*corev1.Secret

	// For each database binding check to see if there are any corresponding WebLogic domain connections
	binding := mbPair.Binding
	for _, databaseBinding := range binding.Spec.DatabaseBindings {
		secretName := databaseBinding.Credentials

		// Get the url from the binding to add to the data for the new secret
		data := make(map[string][]byte)
		data["url"] = []byte(databaseBinding.Url)

		for _, domain := range mbPair.Model.Spec.WeblogicDomains {

			hasConnection := false
			for _, connection := range domain.Connections {
				for _, databaseConnection := range connection.Database {
					if databaseConnection.Target == databaseBinding.Name {
						hasConnection = true
						break
					}
				}
			}
			// If this domain has a database connection that targets this database binding...
			if hasConnection {
				// Find the namespace for this domain in the binding placements
				namespace, err := util.GetComponentNamespace(domain.Name, binding)
				if err != nil {
					glog.V(6).Infof("Getting namespace for domain %s is giving error %s", domain.Name, err)
					continue
				}

				// Create the new secret in the domain's namespace from the secret named in this database binding
				labels := make(map[string]string)
				labels["weblogic.domainUID"] = domain.Name
				secretObj, err := newSecret(secretName, namespace, kubeClientSet, data, labels)
				if err != nil {
					glog.V(6).Infof("Copying secret %s to namespace %s for database binding %s is giving error %s", secretName, namespace, databaseBinding.Name, err)
					continue
				}
				secrets = append(secrets, secretObj)
			}
		}
	}

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

	// Each WebLogic domain requires a runtime encryption secret that contains a randomly generated password.
	// Note: we are assuming that each domain has DomainHomeSourceType is set to FromModel.
	for _, domain := range managedCluster.WlsDomainCRs {
		// Find the namespace for this domain in the binding placements
		namespace, err := util.GetComponentNamespace(domain.Name, binding)
		if err != nil {
			glog.Errorf("Getting namespace for domain %s is giving error %s", domain.Name, err)
			continue
		}

		data := make(map[string][]byte)
		data["password"] = []byte(generateRandomString())

		labels := make(map[string]string)
		labels["weblogic.domainUID"] = domain.Spec.DomainUID

		secretName := domain.Spec.Configuration.Model.RuntimeEncryptionSecret
		secretObj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels:    labels,
			},
			Data: data,
			Type: wlsRuntimeEncryptionSecret,
		}

		secrets = append(secrets, secretObj)
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
