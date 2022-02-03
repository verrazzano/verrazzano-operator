// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"go.uber.org/zap"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	appslistersv1 "k8s.io/client-go/listers/apps/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

// LookupEnvFunc is the function used to lookup env vars - it is made available here so that tests
// can override it rather than setting env vars
var LookupEnvFunc = os.LookupEnv

// ManagedClusterConnection maintains the connection details to a ManagedCluster.
type ManagedClusterConnection struct {
	KubeClient                 kubernetes.Interface
	KubeExtClientSet           apiextensionsclient.Interface
	Lock                       sync.RWMutex
	DeploymentLister           appslistersv1.DeploymentLister
	DeploymentInformer         cache.SharedIndexInformer
	PodLister                  corelistersv1.PodLister
	PodInformer                cache.SharedIndexInformer
	ServiceAccountLister       corelistersv1.ServiceAccountLister
	ServiceAccountInformer     cache.SharedIndexInformer
	NamespaceLister            corelistersv1.NamespaceLister
	NamespaceInformer          cache.SharedIndexInformer
	SecretLister               corelistersv1.SecretLister
	SecretInformer             cache.SharedIndexInformer
	ClusterRoleLister          rbaclistersv1.ClusterRoleLister
	ClusterRoleInformer        cache.SharedIndexInformer
	ClusterRoleBindingLister   rbaclistersv1.ClusterRoleBindingLister
	ClusterRoleBindingInformer cache.SharedIndexInformer
	ConfigMapLister            corelistersv1.ConfigMapLister
	ConfigMapInformer          cache.SharedIndexInformer
	DaemonSetLister            appslistersv1.DaemonSetLister
	DaemonSetInformer          cache.SharedIndexInformer
	ServiceLister              corelistersv1.ServiceLister
	ServiceInformer            cache.SharedIndexInformer
}

// DeploymentHelper defines an interface for deployments.
type DeploymentHelper interface {
	DeleteDeployment(namespace, name string) error
}

// GetManagedBindingLabels returns binding labels for managed cluster.
func GetManagedBindingLabels(binding *types.SyntheticBinding, managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.VerrazzanoGroup, constants.VerrazzanoBinding: binding.Name, constants.VerrazzanoCluster: managedClusterName, constants.IstioInjection: constants.Enabled}
}

// GetManagedLabelsNoBinding return labels for managed cluster with no binding.
func GetManagedLabelsNoBinding(managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.VerrazzanoGroup, constants.VerrazzanoCluster: managedClusterName}
}

// GetManagedNamespaceForBinding return the namespace for a given binding.
func GetManagedNamespaceForBinding(binding *types.SyntheticBinding) string {
	return fmt.Sprintf("%s-%s", constants.VerrazzanoPrefix, binding.Name)
}

// GetLocalBindingLabels returns binding labels for local cluster.
func GetLocalBindingLabels(binding *types.SyntheticBinding) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.VerrazzanoGroup, constants.VerrazzanoBinding: binding.Name}
}

// GetManagedClusterNamespaceForSystem returns the system namespace for Verrazzano.
func GetManagedClusterNamespaceForSystem() string {
	return constants.VerrazzanoSystem
}

// GetVmiNameForBinding returns a Verrazzano Monitoring Instance name.
func GetVmiNameForBinding(bindingName string) string {
	return bindingName
}

// GetVmiURI returns a Verrazzano Monitoring Instance URI.
func GetVmiURI(bindingName string, verrazzanoURI string) string {
	return fmt.Sprintf("vmi.%s.%s", bindingName, verrazzanoURI)
}

// GetServiceAccountNameForSystem return the system service account for Verrazzano.
func GetServiceAccountNameForSystem() string {
	return constants.VerrazzanoSystem
}

// NewVal returns an address of an int32 value.
func NewVal(value int32) *int32 {
	var val = value
	return &val
}

// New64Val returns an address of an int64 value.
func New64Val(value int64) *int64 {
	var val = value
	return &val
}

// Contains checks if a string exists in array of strings.
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// GetManagedClustersForVerrazzanoBinding returns a filtered set of only those applicable to a given
// VerrazzanoBinding given a map of available ManagedClusterConnections.
func GetManagedClustersForVerrazzanoBinding(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*ManagedClusterConnection) (
	map[string]*ManagedClusterConnection, error) {
	filteredManagedClusters := map[string]*ManagedClusterConnection{}
	for _, managedCluster := range vzSynMB.ManagedClusters {
		if _, ok := availableManagedClusterConnections[managedCluster.Name]; !ok {
			return nil, fmt.Errorf("Failed, managed cluster %s referenced in binding %s not found", managedCluster.Name, vzSynMB.SynBinding.Name)
		}
		filteredManagedClusters[managedCluster.Name] = availableManagedClusterConnections[managedCluster.Name]

	}
	return filteredManagedClusters, nil
}

// GetManagedClustersNotForVerrazzanoBinding returns a filtered set of those NOT applicable to a given
// VerrazzanoBinding given a map of available ManagedClusterConnections.
func GetManagedClustersNotForVerrazzanoBinding(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*ManagedClusterConnection) map[string]*ManagedClusterConnection {
	filteredManagedClusters := map[string]*ManagedClusterConnection{}
	for clusterName := range availableManagedClusterConnections {
		found := false
		for _, managedCluster := range vzSynMB.ManagedClusters {
			if clusterName == managedCluster.Name {
				found = true
				break
			}
		}
		if !found {
			filteredManagedClusters[clusterName] = availableManagedClusterConnections[clusterName]
		}
	}
	return filteredManagedClusters
}

// IsClusterInBinding checks if a cluster was found in a binding.
func IsClusterInBinding(clusterName string, allvzSynMBs map[string]*types.SyntheticModelBinding) bool {
	for _, mb := range allvzSynMBs {
		for _, placement := range mb.SynBinding.Spec.Placement {
			if placement.Name == clusterName {
				return true
			}
		}
	}
	return false
}

// SharedVMIDefault return true if the env var SHARED_VMI_DEFAULT is true; this may be overridden by an app binding (future)
func SharedVMIDefault() bool {
	useSharedVMI := false
	envValue, present := LookupEnvFunc(sharedVMIDefault)
	if present {
		zap.S().Debugf("Env var %s = %s", sharedVMIDefault, envValue)
		value, err := strconv.ParseBool(envValue)
		if err == nil {
			useSharedVMI = value
		} else {
			zap.S().Errorf("Failed, invalid value for %s: %s", sharedVMIDefault, envValue)
		}
	}
	return useSharedVMI
}

// GetProfileBindingName will return the binding name based on the profile
// if the profile doesn't have a special binding name, the binding name supplied is returned
// for Dev profile the VMI system binding name is returned
func GetProfileBindingName(bindingName string) string {
	if SharedVMIDefault() {
		return constants.VmiSystemBindingName
	}
	return bindingName
}

// RemoveDuplicateValues removes duplicates from a slice containing string values
func RemoveDuplicateValues(stringSlice []string) []string {
	keys := make(map[string]bool)
	var list []string

	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// IsSystemProfileBindingName return true if the specified binding name is the system VMI name
func IsSystemProfileBindingName(bindingName string) bool {
	return constants.VmiSystemBindingName == bindingName
}
