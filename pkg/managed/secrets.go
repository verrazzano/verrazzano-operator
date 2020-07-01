// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"

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
			secrets = newSecrets(mbPair.Binding, managedClusterObj, kubeClientSet)
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
			specDiffs := diff.CompareIgnoreTargetEmpties(existingSecret, newSecret)
			if specDiffs != "" {
				glog.V(6).Infof("Secret %s : Spec differences %s", newSecret.Name, specDiffs)
				glog.V(4).Infof("Updating secret %s:%s in cluster %s", newSecret.Namespace, newSecret.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.CoreV1().Secrets(newSecret.Namespace).Update(context.TODO(), newSecret, metav1.UpdateOptions{})
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
func newSecrets(binding *v1beta1v8o.VerrazzanoBinding, managedCluster *types.ManagedCluster, kubeClientSet kubernetes.Interface) []*corev1.Secret {

	var secrets []*corev1.Secret

	for namespace, secretNames := range managedCluster.Secrets {
		for _, secretName := range secretNames {
			var secretInMgmtCluster *corev1.Secret
			secretInMgmtCluster, err := local.GetSecret(secretName, constants.DefaultNamespace, kubeClientSet)
			if err != nil {
				// Check verrazzano system namespace if we don't find the secret in the default namespace since
				// VMI binding secrets are located in the verrazzano system namespace.
				secretInMgmtCluster, err = local.GetSecret(secretName, constants.VerrazzanoNamespace, kubeClientSet)
				if err != nil {
					continue
				}
			}

			secretObj := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: secretInMgmtCluster.Data,
				Type: secretInMgmtCluster.Type,
			}
			secrets = append(secrets, secretObj)
		}
	}

	return secrets
}
