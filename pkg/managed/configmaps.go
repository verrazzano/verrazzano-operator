// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func CreateConfigMaps(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for create config maps
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ConfigMaps").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Creating/updating ConfigMap for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the clusters that this binding applies to Construct configMaps for each Cluster
	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		if mbPair.Binding.Name == constants.VmiSystemBindingName {
			newConfigMaps, err := newConfigMaps(mbPair.Binding.Name, clusterName)
			if err != nil {
				return err
			}
			for _, newConfigMap := range newConfigMaps {
				err = createUpdateConfigMaps(managedClusterConnection, newConfigMap, clusterName)
			}
		} else {
			for _, newConfigMap := range managedClusterObj.ConfigMaps {
				err = createUpdateConfigMaps(managedClusterConnection, newConfigMap, clusterName)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func createUpdateConfigMaps(managedClusterConnection *util.ManagedClusterConnection, newConfigMap *corev1.ConfigMap, clusterName string) error {
	// Create log instance for create update config maps
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "UpdateConfigMaps").Str("name", "Creation").Logger()

	existingcm, err := managedClusterConnection.ConfigMapLister.ConfigMaps(newConfigMap.Namespace).Get(newConfigMap.Name)
	if existingcm != nil {
		// If config map already exists, check the spec differences
		specDiffs := diff.CompareIgnoreTargetEmpties(existingcm, newConfigMap)
		if specDiffs != "" {
			logger.Debug().Msgf("ConfigMap %s : Spec differences %s", newConfigMap.Name, specDiffs)
			logger.Info().Msgf("Updating ConfigMap %s in cluster %s", newConfigMap.Name, clusterName)
			_, err = managedClusterConnection.KubeClient.CoreV1().ConfigMaps(newConfigMap.Namespace).Update(context.TODO(), newConfigMap, metav1.UpdateOptions{})
		}
	} else {
		logger.Info().Msgf("Creating ConfigMap %s in cluster %s", newConfigMap.Name, clusterName)
		_, err = managedClusterConnection.KubeClient.CoreV1().ConfigMaps(newConfigMap.Namespace).Create(context.TODO(), newConfigMap, metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}
	return nil
}

func CleanupOrphanedConfigMaps(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error {
	// Create log instance for cleaning up orphaned config maps
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "OrphanedConfigMaps").Str("name", "Clean").Logger()

	logger.Debug().Msgf("Cleaning up orphaned ConfigMaps for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(mbPair, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// First, get rid of any ConfigMaps with the specified binding
		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of ConfigMaps for this cluster and given binding
		existingConfigMapsList, err := managedClusterConnection.ConfigMapLister.List(selector)
		if err != nil {
			return err
		}
		// Delete these ConfigMaps since none are expected on this cluster
		for _, configMap := range existingConfigMapsList {
			logger.Info().Msgf("Deleting ConfigMap %s in cluster %s", configMap.Name, clusterName)
			err := managedClusterConnection.KubeClient.CoreV1().ConfigMaps(configMap.Namespace).Delete(context.TODO(), configMap.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Constructs the necessary ConfigMaps for the specified ManagedCluster in the given VerrazzanoBinding
func newConfigMaps(bindingName string, managedClusterName string) ([]*corev1.ConfigMap, error) {
	var configMaps []*corev1.ConfigMap
	configMapsLogging := monitoring.LoggingConfigMaps(managedClusterName)
	for _, configMap := range configMapsLogging {
		configMaps = append(configMaps, configMap)
	}
	return configMaps, nil
}
