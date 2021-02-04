// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"sync"

	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// CreateModelBindingPair creates a unique pairing of a model and binding.
func CreateModelBindingPair(model *types.ClusterInfo, binding *types.ResourceLocation, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) *types.VerrazzanoLocation {

	mbPair := &types.VerrazzanoLocation{
		Cluster:          model,
		Location:         binding,
		ManagedClusters:  map[string]*types.ManagedCluster{},
		Lock:             &sync.RWMutex{},
		VerrazzanoURI:    VerrazzanoURI,
		ImagePullSecrets: imagePullSecrets,
	}
	return buildModelBindingPair(mbPair)
}

func acquireLock(mbPair *types.VerrazzanoLocation) {
	mbPair.Lock.Lock()
}

func releaseLock(mbPair *types.VerrazzanoLocation) {
	mbPair.Lock.Unlock()
}

// UpdateModelBindingPair updates a model/binding pair by rebuilding it.
func UpdateModelBindingPair(mbPair *types.VerrazzanoLocation, model *types.ClusterInfo, binding *types.ResourceLocation, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) {
	newMBPair := CreateModelBindingPair(model, binding, VerrazzanoURI, imagePullSecrets)
	lock := mbPair.Lock
	lock.Lock()
	defer lock.Unlock()
	*mbPair = *newMBPair
}

// Common function for building a model/binding pair
func buildModelBindingPair(mbPair *types.VerrazzanoLocation) *types.VerrazzanoLocation {
	// Acquire write lock
	acquireLock(mbPair)
	defer releaseLock(mbPair)

	// Obtain the placement objects
	placements := getPlacements(mbPair.Location)

	for _, placement := range placements {
		var mc *types.ManagedCluster
		mc, clusterFound := mbPair.ManagedClusters[placement.Name]
		// Create the ManagedCluster object and add it to the map if we don't have one
		// for a given placement (cluster)
		if !clusterFound {
			mc = &types.ManagedCluster{
				Name:        placement.Name,
				Secrets:     map[string][]string{},
				Ingresses:   map[string][]*types.Ingress{},
				RemoteRests: map[string][]*types.RemoteRestConnection{},
			}
			mbPair.ManagedClusters[placement.Name] = mc
		}

		// Add in the Verrazzano system namespace if not already added
		managedNamespace := util.GetManagedClusterNamespaceForSystem()
		appendNamespace(&mc.Namespaces, managedNamespace)
	}

	return mbPair
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
