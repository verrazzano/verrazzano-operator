// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding

package managed

import (
	"context"
	"fmt"

	"os"
	"time"

	"github.com/rs/zerolog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CreateNamespaces creates/updates namespaces needed for each managed cluster.
func CreateNamespaces(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for create namespaces
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Namespaces").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Creating/updating Namespaces for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Construct namespaces for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var namespaces []*corev1.Namespace
		namespaces = newNamespaces(mbPair.Binding, managedClusterObj)

		// Create/Update Namespace
		err := createNamespace(mbPair.Binding, managedClusterConnection, namespaces, clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createNamespace(binding *v1beta1v8o.VerrazzanoBinding, managedClusterConnection *util.ManagedClusterConnection, newNamespaces []*corev1.Namespace, clusterName string) error {
	// Create log instance for create namespaces
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Namespaces").Str("name", "Creation").Logger()

	// Construct the set of expected namespaces
	for _, newNamespace := range newNamespaces {
		existingNamespace, err := managedClusterConnection.NamespaceLister.Get(newNamespace.Name)
		if existingNamespace != nil {
			// Do nothing in the case of an existing namespace.  When a namespace is later associated with a project in Rancher, Rancher "takes control" of the namespace, adding additional labels and finalizers.  For now, we'll opt not to touch the namespace after it's created.
			logger.Debug().Msgf("Namespace %s already exists in cluster %s, doing nothing...", existingNamespace.Name, clusterName)
		} else {
			logger.Info().Msgf("Creating namespace %s in cluster %s", newNamespace.Name, clusterName)
			_, err = managedClusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), newNamespace, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupOrphanedNamespaces deletes namespaces that have been orphaned.
func CleanupOrphanedNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error {
	// Create log instance for creating namespaces
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Namespaces").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Cleaning up orphaned Namespace for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Get the managed clusters that this binding applies to
	matchedClusters, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	for clusterName, mc := range mbPair.ManagedClusters {
		managedClusterConnection := matchedClusters[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		if mc.Namespaces != nil {
			// Create list of Namespaces expected on this cluster
			var Namespaces []string
			for _, ns := range mc.Namespaces {
				Namespaces = append(Namespaces, ns)
			}

			// Get list of Namespaces for this cluster and given binding
			existingNamespaceList, err := managedClusterConnection.NamespaceLister.List(selector)
			if err != nil {
				return err
			}

			// Delete any Namespaces apps not expected on this cluster
			for _, namespace := range existingNamespaceList {
				if !util.Contains(Namespaces, namespace.Name) {
					logger.Debug().Msgf("Deleting Namespaces %s in cluster %s", namespace.Name, clusterName)
					err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), namespace.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}
			}
		}
	}

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(mbPair, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// First, get rid of any Namespace with the specified binding
		existingNamespaceList, err := managedClusterConnection.NamespaceLister.List(selector)
		if err != nil {
			return err
		}

		// Delete these Namespace since none are expected on this cluster
		for _, ns := range existingNamespaceList {
			// Skip deletion of namespaces Logging and Monitoring from management cluster
			if ns.Name != constants.MonitoringNamespace && ns.Name != constants.LoggingNamespace {
				logger.Info().Msgf("Deleting Namespace %s in cluster %s", ns.Name, clusterName)
				err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForNSDeletion(managedClusterConnection, ns.Name, clusterName, 2*time.Minute, 1*time.Second)
				if err != nil {
					logger.Error().Msgf("Failed to delete namespace %s, for the reason (%v)", ns.Name, err)
					return err
				}
			}
		}

		// Second, get rid of any system-wide Namespaces if no bindings are using this cluster
		if !util.IsClusterInBinding(clusterName, allMbPairs) {
			selector = labels.SelectorFromSet(util.GetManagedLabelsNoBinding(clusterName))

			// Get list of system-wide Namespaces for this cluster
			existingNamespaceSystemList, err := managedClusterConnection.NamespaceLister.List(selector)
			if err != nil {
				return err
			}

			// Delete these Namespaces since none are expected on this cluster
			for _, ns := range existingNamespaceSystemList {
				// Skip deletion of namespaces Logging, Monitoring, and verrazzano system from management cluster
				if ns.Name != constants.MonitoringNamespace && ns.Name != constants.LoggingNamespace && ns.Name != constants.VerrazzanoNamespace {
					logger.Info().Msgf("Deleting Namespace %s in cluster %s", ns.Name, clusterName)
					err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}

					err = waitForNSDeletion(managedClusterConnection, ns.Name, clusterName, 2*time.Minute, 1*time.Second)
					if err != nil {
						logger.Error().Msgf("Failed to delete namespace %s, for the reason (%v)", ns.Name, err)
						return err
					}
				}
			}
		}
	}

	return nil
}

// DeleteNamespaces deletes namespaces for a given binding.
func DeleteNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	// Create log instance for deleting namespaces
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Namespaces").Str("name", "Deletion").Logger()

	logger.Debug().Msgf("Deleting Namespaces for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	managedClusters, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Delete namespaces associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range managedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var selector labels.Selector
		if bindingLabel {
			selector = labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name})
		} else {
			selector = labels.SelectorFromSet(util.GetManagedLabelsNoBinding(clusterName))
		}
		existingNamespace, err := managedClusterConnection.NamespaceLister.List(selector)
		if err != nil {
			return err
		}
		for _, namespace := range existingNamespace {
			// Skip deletion of namespaces Logging, Monitoring, and verrazzano system namespace from management cluster
			if namespace.Name != constants.MonitoringNamespace && namespace.Name != constants.LoggingNamespace && namespace.Name != constants.VerrazzanoNamespace {
				logger.Info().Msgf("Deleting Namespace %s in cluster %s", namespace.Name, clusterName)
				err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), namespace.Name, metav1.DeleteOptions{})
				if err != nil {
					logger.Error().Msgf("Failed to delete namespace %s, for the reason (%v)", namespace.Name, err)
					return err
				}

				err = waitForNSDeletion(managedClusterConnection, namespace.Name, clusterName, 2*time.Minute, 1*time.Second)
				if err != nil {
					logger.Error().Msgf("Failed to delete namespace %s, for the reason (%v)", namespace.Name, err)
					return err
				}
			}
		}
	}
	return nil
}

// Constructs the necessary Namespaces for the specified ManagedCluster in the given VerrazzanoBinding
func newNamespaces(binding *v1beta1v8o.VerrazzanoBinding, managedCluster *types.ManagedCluster) []*corev1.Namespace {
	var namespaces []*corev1.Namespace

	for _, namespace := range managedCluster.Namespaces {
		namespaceObj := &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: func() map[string]string {
					var bindingLabels map[string]string
					// Don't add the binding label for the namespace used by the verrazzano system.
					// This namespace is common across model/binding pairs.
					if util.GetManagedClusterNamespaceForSystem() == namespace {
						bindingLabels = util.GetManagedLabelsNoBinding(managedCluster.Name)
					} else {
						bindingLabels = util.GetManagedBindingLabels(binding, managedCluster.Name)
					}
					// Don't enable istio for namespaces used for the verrazzano system namespace, the monitoring
					// namespace, and the logging namespace.
					if util.GetManagedClusterNamespaceForSystem() != namespace && namespace != constants.MonitoringNamespace && namespace != constants.LoggingNamespace {
						bindingLabels["istio-injection"] = "enabled"
					}
					return bindingLabels
				}(),
			},
			Spec: corev1.NamespaceSpec{
				Finalizers: []corev1.FinalizerName{corev1.FinalizerName("kubernetes")},
			},
			Status: corev1.NamespaceStatus{},
		}
		namespaces = append(namespaces, namespaceObj)
	}

	return namespaces
}

func waitForNSDeletion(mc *util.ManagedClusterConnection, namespace string, cluster string, timeoutDuration time.Duration, tickDuration time.Duration) error {
	// Create log instance for deleting NS
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Namespace").Str("name", "Deletion").Logger()

	timeout := time.After(timeoutDuration)
	tick := time.Tick(tickDuration)
	var err error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for namespace %s to be removed in  managed cluster %s", namespace, cluster)
		case <-tick:
			logger.Info().Msgf("Waiting for namespace %s in managed cluster %s to be removed..", namespace, cluster)
			_, err = mc.KubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
			if err != nil && k8sErrors.IsNotFound(err) {
				logger.Info().Msgf("Removed namespace %s in managed cluster %s ..", namespace, cluster)
				return nil
			}

			if err != nil {
				return fmt.Errorf("Error removing namespace %s in managed cluster %s, error %s", namespace, cluster, err.Error())
			}
		}
	}
}
