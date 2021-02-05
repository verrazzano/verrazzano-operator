// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"sync"

	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// CreateVerrazzanoLocation creates a Verrazzano location.
func CreateVerrazzanoLocation(model *types.ClusterInfo, binding *types.ResourceLocation, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) *types.VerrazzanoLocation {

	vzLocation := &types.VerrazzanoLocation{
		Cluster:          model,
		Location:         binding,
		ManagedClusters:  map[string]*types.ManagedCluster{},
		Lock:             &sync.RWMutex{},
		VerrazzanoURI:    VerrazzanoURI,
		ImagePullSecrets: imagePullSecrets,
	}
	return buildVerrazzanoLocation(vzLocation)
}

func acquireLock(vzLocation *types.VerrazzanoLocation) {
	vzLocation.Lock.Lock()
}

func releaseLock(vzLocation *types.VerrazzanoLocation) {
	vzLocation.Lock.Unlock()
}

// UpdateVerrazzanoLocation updates a model/binding pair by rebuilding it.
func UpdateVerrazzanoLocation(vzLocation *types.VerrazzanoLocation, model *types.ClusterInfo, binding *types.ResourceLocation, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) {
	newvzLocation := CreateVerrazzanoLocation(model, binding, VerrazzanoURI, imagePullSecrets)
	lock := vzLocation.Lock
	lock.Lock()
	defer lock.Unlock()
	*vzLocation = *newvzLocation
}

// Common function for building a model/binding pair
func buildVerrazzanoLocation(vzLocation *types.VerrazzanoLocation) *types.VerrazzanoLocation {
	// Acquire write lock
	acquireLock(vzLocation)
	defer releaseLock(vzLocation)

	// Obtain the placement objects
	placements := getPlacements(vzLocation.Location)

	for _, placement := range placements {
		var mc *types.ManagedCluster
		mc, clusterFound := vzLocation.ManagedClusters[placement.Name]
		// Create the ManagedCluster object and add it to the map if we don't have one
		// for a given placement (cluster)
		if !clusterFound {
			mc = &types.ManagedCluster{
				Name:        placement.Name,
				Secrets:     map[string][]string{},
				Ingresses:   map[string][]*types.Ingress{},
				RemoteRests: map[string][]*types.RemoteRestConnection{},
			}
			vzLocation.ManagedClusters[placement.Name] = mc
		}

		// Add in the Verrazzano system namespace if not already added
		managedNamespace := util.GetManagedClusterNamespaceForSystem()
		appendNamespace(&mc.Namespaces, managedNamespace)
	}

	return vzLocation
}

func getPlacements(binding *types.ResourceLocation) []types.ClusterPlacement {
	placements := binding.Spec.Placement
	return placements
}

func appendNamespace(Namespaces *[]string, namespace string) {
	nsFound := false
	for _, ns := range *Namespaces {
		if ns == namespace {
			nsFound = true
			break
		}
	}
	if !nsFound {
		*Namespaces = append(*Namespaces, namespace)
	}

}
