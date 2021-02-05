// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"sync"

	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// CreateSyntheticModelBinding creates a Verrazzano location.
func CreateSyntheticModelBinding(model *types.SyntheticModel, binding *types.SyntheticBinding, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) *types.SyntheticModelBinding {

	vzSynMB := &types.SyntheticModelBinding{
		SynModel:         model,
		SynBinding:       binding,
		ManagedClusters:  map[string]*types.ManagedCluster{},
		Lock:             &sync.RWMutex{},
		VerrazzanoURI:    VerrazzanoURI,
		ImagePullSecrets: imagePullSecrets,
	}
	return buildSyntheticModelBinding(vzSynMB)
}

func acquireLock(vzSynMB *types.SyntheticModelBinding) {
	vzSynMB.Lock.Lock()
}

func releaseLock(vzSynMB *types.SyntheticModelBinding) {
	vzSynMB.Lock.Unlock()
}

// UpdateSyntheticModelBinding updates a model/binding pair by rebuilding it.
func UpdateSyntheticModelBinding(vzSynMB *types.SyntheticModelBinding, model *types.SyntheticModel, binding *types.SyntheticBinding, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) {
	newvzSynMB := CreateSyntheticModelBinding(model, binding, VerrazzanoURI, imagePullSecrets)
	lock := vzSynMB.Lock
	lock.Lock()
	defer lock.Unlock()
	*vzSynMB = *newvzSynMB
}

// Common function for building a model/binding pair
func buildSyntheticModelBinding(vzSynMB *types.SyntheticModelBinding) *types.SyntheticModelBinding {
	// Acquire write lock
	acquireLock(vzSynMB)
	defer releaseLock(vzSynMB)

	// Obtain the placement objects
	placements := getPlacements(vzSynMB.SynBinding)

	for _, placement := range placements {
		var mc *types.ManagedCluster
		mc, clusterFound := vzSynMB.ManagedClusters[placement.Name]
		// Create the ManagedCluster object and add it to the map if we don't have one
		// for a given placement (cluster)
		if !clusterFound {
			mc = &types.ManagedCluster{
				Name:        placement.Name,
				Secrets:     map[string][]string{},
				Ingresses:   map[string][]*types.Ingress{},
				RemoteRests: map[string][]*types.RemoteRestConnection{},
			}
			vzSynMB.ManagedClusters[placement.Name] = mc
		}

		// Add in the Verrazzano system namespace if not already added
		managedNamespace := util.GetManagedClusterNamespaceForSystem()
		appendNamespace(&mc.Namespaces, managedNamespace)
	}

	return vzSynMB
}

func getPlacements(binding *types.SyntheticBinding) []types.ClusterPlacement {
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
