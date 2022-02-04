// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of deployments based, on a VerrazzanoBinding

package managed

import (
	"context"
	"fmt"

	"time"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CreateNamespaces creates/updates namespaces needed for each managed cluster.
func CreateNamespaces(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	zap.S().Debugf("Creating/updating Namespaces for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// Construct namespaces for each ManagedCluster
	for clusterName, managedClusterObj := range vzSynMB.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var namespaces []*corev1.Namespace
		namespaces = newNamespaces(vzSynMB.SynBinding, managedClusterObj)

		// Create/Update Namespace
		err := createNamespace(vzSynMB.SynBinding, managedClusterConnection, namespaces, clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createNamespace(binding *types.SyntheticBinding, managedClusterConnection *util.ManagedClusterConnection, newNamespaces []*corev1.Namespace, clusterName string) error {
	// Construct the set of expected namespaces
	for _, newNamespace := range newNamespaces {
		existingNamespace, err := managedClusterConnection.NamespaceLister.Get(newNamespace.Name)
		if existingNamespace != nil {
			// Do nothing in the case of an existing namespace.  When a namespace is later associated with a project in Rancher, Rancher "takes control" of the namespace, adding additional labels and finalizers.  For now, we'll opt not to touch the namespace after it's created.
			zap.S().Debugf("Namespace %s already exists in cluster %s, doing nothing...", existingNamespace.Name, clusterName)
		} else {
			zap.S().Infof("Creating namespace %s in cluster %s", newNamespace.Name, clusterName)
			_, err = managedClusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), newNamespace, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupOrphanedNamespaces deletes namespaces that have been orphaned.
func CleanupOrphanedNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allvzSynMBs map[string]*types.SyntheticModelBinding) error {
	zap.S().Debugf("Cleaning up orphaned Namespace for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// Get the managed clusters that this binding applies to
	matchedClusters, err := util.GetManagedClustersForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	for clusterName, mc := range vzSynMB.ManagedClusters {
		managedClusterConnection := matchedClusters[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.SynBinding.Name, constants.VerrazzanoCluster: clusterName})

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
					zap.S().Debugf("Deleting Namespaces %s in cluster %s", namespace.Name, clusterName)
					err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), namespace.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}
			}
		}
	}

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.SynBinding.Name, constants.VerrazzanoCluster: clusterName})

		// First, get rid of any Namespace with the specified binding
		existingNamespaceList, err := managedClusterConnection.NamespaceLister.List(selector)
		if err != nil {
			return err
		}

		// Delete these Namespace since none are expected on this cluster
		for _, ns := range existingNamespaceList {
			// Skip deletion of namespaces Logging and Monitoring from management cluster
			if ns.Name != constants.MonitoringNamespace {
				zap.S().Infof("Deleting Namespace %s in cluster %s", ns.Name, clusterName)
				err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForNSDeletion(managedClusterConnection, ns.Name, clusterName, 2*time.Minute, 1*time.Second)
				if err != nil {
					zap.S().Errorf("Failed to delete namespace %s, for the reason (%v)", ns.Name, err)
					return err
				}
			}
		}

		// Second, get rid of any system-wide Namespaces if no bindings are using this cluster
		if !util.IsClusterInBinding(clusterName, allvzSynMBs) {
			selector = labels.SelectorFromSet(util.GetManagedLabelsNoBinding(clusterName))

			// Get list of system-wide Namespaces for this cluster
			existingNamespaceSystemList, err := managedClusterConnection.NamespaceLister.List(selector)
			if err != nil {
				return err
			}

			// Delete these Namespaces since none are expected on this cluster
			for _, ns := range existingNamespaceSystemList {
				// Skip deletion of namespaces Logging, Monitoring, and verrazzano system from management cluster
				if ns.Name != constants.MonitoringNamespace && ns.Name != constants.VerrazzanoNamespace {
					zap.S().Infof("Deleting Namespace %s in cluster %s", ns.Name, clusterName)
					err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}

					err = waitForNSDeletion(managedClusterConnection, ns.Name, clusterName, 2*time.Minute, 1*time.Second)
					if err != nil {
						zap.S().Errorf("Failed to delete namespace %s, for the reason (%v)", ns.Name, err)
						return err
					}
				}
			}
		}
	}

	return nil
}

// DeleteNamespaces deletes namespaces for a given binding.
func DeleteNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	zap.S().Debugf("Deleting Namespaces for VerrazzanoBinding %s", vzSynMB.SynBinding.Name)

	// Parse out the managed clusters that this binding applies to
	managedClusters, err := util.GetManagedClustersForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Delete namespaces associated with the given VerrazzanoBinding (based on labels selectors)
	for clusterName, managedClusterConnection := range managedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		var selector labels.Selector
		if bindingLabel {
			selector = labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: vzSynMB.SynBinding.Name})
		} else {
			selector = labels.SelectorFromSet(util.GetManagedLabelsNoBinding(clusterName))
		}
		existingNamespace, err := managedClusterConnection.NamespaceLister.List(selector)
		if err != nil {
			return err
		}
		for _, namespace := range existingNamespace {
			// Skip deletion of namespaces Logging, Monitoring, and verrazzano system namespace from management cluster
			if namespace.Name != constants.MonitoringNamespace && namespace.Name != constants.VerrazzanoNamespace {
				zap.S().Infof("Deleting Namespace %s in cluster %s", namespace.Name, clusterName)
				err := managedClusterConnection.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), namespace.Name, metav1.DeleteOptions{})
				if err != nil {
					zap.S().Errorf("Failed to delete namespace %s, for the reason (%v)", namespace.Name, err)
					return err
				}

				err = waitForNSDeletion(managedClusterConnection, namespace.Name, clusterName, 2*time.Minute, 1*time.Second)
				if err != nil {
					zap.S().Errorf("Failed to delete namespace %s, for the reason (%v)", namespace.Name, err)
					return err
				}
			}
		}
	}
	return nil
}

// Constructs the necessary Namespaces for the specified ManagedCluster in the given VerrazzanoBinding
func newNamespaces(binding *types.SyntheticBinding, managedCluster *types.ManagedCluster) []*corev1.Namespace {
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
					if util.GetManagedClusterNamespaceForSystem() != namespace && namespace != constants.MonitoringNamespace {
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
	timeout := time.After(timeoutDuration)
	tick := time.Tick(tickDuration)
	var err error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("Failed, timed out waiting for namespace %s to be removed in  managed cluster %s", namespace, cluster)
		case <-tick:
			zap.S().Infof("Waiting for namespace %s in managed cluster %s to be removed..", namespace, cluster)
			_, err = mc.KubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
			if err != nil && k8sErrors.IsNotFound(err) {
				zap.S().Infof("Removed namespace %s in managed cluster %s ..", namespace, cluster)
				return nil
			}

			if err != nil {
				return fmt.Errorf("Failed removing namespace %s in managed cluster %s, %v", namespace, cluster, err)
			}
		}
	}
}
