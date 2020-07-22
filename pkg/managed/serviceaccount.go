// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding
package managed

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateServiceAccounts(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating crd definitions
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ServiceAccounts").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Creating/updating Deployments for VerrazzanoBinding %s", mbPair.Binding.Name)

	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct service account for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var serviceAccounts []*corev1.ServiceAccount
		if mbPair.Binding.Name == constants.VmiSystemBindingName {
			// Add add the service accounts needed for monitoring and logging
			for _, serviceAccountName := range monitoring.GetMonitoringComponents() {
				serviceAccounts = append(serviceAccounts, newServiceAccounts(mbPair.Binding, managedClusterObj, serviceAccountName, monitoring.GetMonitoringComponentLabels(clusterName, serviceAccountName), monitoring.GetMonitoringNamespace(serviceAccountName))...)
			}
		} else {
			serviceAccounts = newServiceAccounts(mbPair.Binding, managedClusterObj, util.GetServiceAccountNameForSystem(), util.GetManagedLabelsNoBinding(clusterName), "")
		}

		// Create/Update ServiceAccount
		err := createServiceAccount(mbPair.Binding, managedClusterConnection, serviceAccounts, clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createServiceAccount(binding *v1beta1v8o.VerrazzanoBinding, managedClusterConnection *util.ManagedClusterConnection, newServiceAccounts []*corev1.ServiceAccount, clusterName string) error {
	// Create log instance for creating crd definitions
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ServiceAccounts").Str("name", "Creation").Logger()

	// Create/Update Service Account
	var serviceAccountNames = []string{}

	for _, newServiceAccount := range newServiceAccounts {
		serviceAccountNames = append(serviceAccountNames, newServiceAccount.Name)
		existingServiceAccount, err := managedClusterConnection.ServiceAccountLister.ServiceAccounts(newServiceAccount.Namespace).Get(newServiceAccount.Name)
		if existingServiceAccount != nil {
			specDiffs := diff.CompareIgnoreTargetEmpties(existingServiceAccount, newServiceAccount)
			if specDiffs != "" {
				logger.Debug().Msgf("ServiceAccount %s : Spec differences %s", newServiceAccount.Name, specDiffs)
				logger.Info().Msgf("Updating ServiceAccount %s:%s in cluster %s", newServiceAccount.Namespace, newServiceAccount.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.CoreV1().ServiceAccounts(newServiceAccount.Namespace).Update(context.TODO(), newServiceAccount, metav1.UpdateOptions{})
			}
		} else {
			logger.Info().Msgf("Creating ServiceAccount %s:%s in cluster %s", newServiceAccount.Namespace, newServiceAccount.Name, clusterName)
			_, err = managedClusterConnection.KubeClient.CoreV1().ServiceAccounts(newServiceAccount.Namespace).Create(context.TODO(), newServiceAccount, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// Constructs the necessary ServiceAccounts for the specified ManagedCluster in the given VerrazzanoBinding
func newServiceAccounts(binding *v1beta1v8o.VerrazzanoBinding, managedCluster *types.ManagedCluster, name string, labels map[string]string, namespaceName string) []*corev1.ServiceAccount {
	var serviceAccounts []*corev1.ServiceAccount

	for _, namespace := range managedCluster.Namespaces {
		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Namespace: func() string {
					// Get namespace for monitoring components in case of System binding.
					if binding.Name == constants.VmiSystemBindingName {
						return namespaceName
					} else {
						return namespace
					}
				}(),
				Labels: labels,
			},
		}
		serviceAccounts = append(serviceAccounts, serviceAccount)
		// Only add service account resource once in case of system binding
		if binding.Name == constants.VmiSystemBindingName {
			break
		}
	}
	return serviceAccounts
}
