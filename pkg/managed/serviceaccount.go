// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding

package managed

import (
	"context"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceAccounts creates/updates service accounts needed for each managed cluster.
func CreateServiceAccounts(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {

	glog.V(6).Infof("Creating/updating Deployments for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Construct service account for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var serviceAccounts []*corev1.ServiceAccount
		if mbPair.Binding.Name == constants.VmiSystemBindingName {
			// Add add the service accounts needed for monitoring and logging
			for _, serviceAccountName := range monitoring.GetMonitoringComponents() {
				serviceAccounts = append(serviceAccounts, newServiceAccounts(mbPair.Binding, managedClusterObj, serviceAccountName, monitoring.GetMonitoringComponentLabels(clusterName, serviceAccountName), monitoring.GetMonitoringNamespace(serviceAccountName), mbPair.ImagePullSecrets)...)
			}
		} else {
			serviceAccounts = newServiceAccounts(mbPair.Binding, managedClusterObj, util.GetServiceAccountNameForSystem(), util.GetManagedLabelsNoBinding(clusterName), "", mbPair.ImagePullSecrets)
		}

		// Create/Update ServiceAccount
		err := createServiceAccount(managedClusterConnection, serviceAccounts, clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createServiceAccount(managedClusterConnection *util.ManagedClusterConnection, newServiceAccounts []*corev1.ServiceAccount, clusterName string) error {
	// Create/Update Service Account

	for _, newServiceAccount := range newServiceAccounts {
		existingServiceAccount, err := managedClusterConnection.ServiceAccountLister.ServiceAccounts(newServiceAccount.Namespace).Get(newServiceAccount.Name)
		if existingServiceAccount != nil {
			specDiffs := diff.CompareIgnoreTargetEmpties(existingServiceAccount, newServiceAccount)
			if specDiffs != "" {
				glog.V(6).Infof("ServiceAccount %s : Spec differences %s", newServiceAccount.Name, specDiffs)
				glog.V(4).Infof("Updating ServiceAccount %s:%s in cluster %s", newServiceAccount.Namespace, newServiceAccount.Name, clusterName)
				_, err = managedClusterConnection.KubeClient.CoreV1().ServiceAccounts(newServiceAccount.Namespace).Update(context.TODO(), newServiceAccount, metav1.UpdateOptions{})
			}
		} else {
			glog.V(4).Infof("Creating ServiceAccount %s:%s in cluster %s", newServiceAccount.Namespace, newServiceAccount.Name, clusterName)
			_, err = managedClusterConnection.KubeClient.CoreV1().ServiceAccounts(newServiceAccount.Namespace).Create(context.TODO(), newServiceAccount, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// Constructs the necessary ServiceAccounts for the specified ManagedCluster in the given VerrazzanoBinding
func newServiceAccounts(binding *v1beta1v8o.VerrazzanoBinding, managedCluster *types.ManagedCluster, name string, labels map[string]string, namespaceName string, imagePullSecerts []corev1.LocalObjectReference) []*corev1.ServiceAccount {
	var serviceAccounts []*corev1.ServiceAccount

	for _, namespace := range managedCluster.Namespaces {
		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Namespace: func() string {
					// Get namespace for monitoring components in case of System binding.
					if binding.Name == constants.VmiSystemBindingName {
						return namespaceName
					}
					return namespace
				}(),
				Labels: labels,
			},
			ImagePullSecrets: imagePullSecerts,
		}
		serviceAccounts = append(serviceAccounts, serviceAccount)
		// Only add service account resource once in case of system binding
		if binding.Name == constants.VmiSystemBindingName {
			break
		}
	}
	return serviceAccounts
}
