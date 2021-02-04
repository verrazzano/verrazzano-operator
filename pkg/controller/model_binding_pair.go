// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"fmt"
	"sync"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// CreateModelBindingPair creates a unique pairing of a model and binding.
func CreateModelBindingPair(model *types.ClusterModel, binding *types.ClusterBinding, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) *types.ModelBindingPair {

	mbPair := &types.ModelBindingPair{
		Model:            model,
		Binding:          binding,
		ManagedClusters:  map[string]*types.ManagedCluster{},
		Lock:             &sync.RWMutex{},
		VerrazzanoURI:    VerrazzanoURI,
		ImagePullSecrets: imagePullSecrets,
	}
	return buildModelBindingPair(mbPair)
}

func acquireLock(mbPair *types.ModelBindingPair) {
	mbPair.Lock.Lock()
}

func releaseLock(mbPair *types.ModelBindingPair) {
	mbPair.Lock.Unlock()
}

// UpdateModelBindingPair updates a model/binding pair by rebuilding it.
func UpdateModelBindingPair(mbPair *types.ModelBindingPair, model *types.ClusterModel, binding *types.ClusterBinding, VerrazzanoURI string, imagePullSecrets []corev1.LocalObjectReference) {
	newMBPair := CreateModelBindingPair(model, binding, VerrazzanoURI, imagePullSecrets)
	lock := mbPair.Lock
	lock.Lock()
	defer lock.Unlock()
	*mbPair = *newMBPair
}

// Common function for building a model/binding pair
func buildModelBindingPair(mbPair *types.ModelBindingPair) *types.ModelBindingPair {
	// Acquire write lock
	acquireLock(mbPair)
	defer releaseLock(mbPair)

	// Obtain the placement objects
	placements := getPlacements(mbPair.Binding)

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

func getPlacements(binding *types.ClusterBinding) []types.VerrazzanoPlacement {
	placements := binding.Spec.Placement
	return placements
}

// Add the name of a secret to the array of secret names in a managed cluster.
func addSecret(mc *types.ManagedCluster, secretName string, namespace string) {
	// Check to see if the name of the secret has already been added
	secretsList, ok := mc.Secrets[namespace]
	if ok {
		for _, name := range secretsList {
			if name == secretName {
				return
			}
		}
	}
	mc.Secrets[namespace] = append(mc.Secrets[namespace], secretName)
}

func addMatch(matches []types.MatchRequest, key, value string) []types.MatchRequest {
	found := false
	for _, match := range matches {
		if match.URI[key] == value {
			found = true
		}
	}
	if !found {
		matches = append(matches, types.MatchRequest{
			URI: map[string]string{key: value},
		})
	}
	return matches
}

func getOrNewIngress(mc *types.ManagedCluster, ingressConn v1beta1v8o.VerrazzanoIngressConnection, namespace string) *types.Ingress {
	ingressList, ok := mc.Ingresses[namespace]
	if ok {
		for _, ingress := range ingressList {
			if ingress.Name == ingressConn.Name {
				return ingress
			}
		}
	}
	ingress := types.Ingress{
		Name:        ingressConn.Name,
		Destination: []*types.IngressDestination{},
	}
	mc.Ingresses[namespace] = append(mc.Ingresses[namespace], &ingress)
	return &ingress
}

func getOrNewDest(ingress *types.Ingress, destinationHost string, virtualSerivceDestinationPort uint32) *types.IngressDestination {
	for _, destination := range ingress.Destination {
		if destination.Host == destinationHost && destination.Port == virtualSerivceDestinationPort {
			return destination
		}
	}
	dest := types.IngressDestination{
		Host: destinationHost,
		Port: virtualSerivceDestinationPort,
	}
	ingress.Destination = append(ingress.Destination, &dest)
	return &dest
}

// Add a remote rest connection to the array of remote rest connections in a managed cluster
func addRemoteRest(mc *types.ManagedCluster, restName string, localNamespace string, remoteMc *types.ManagedCluster, remoteNamespace string, remotePort uint32, remoteClusterName string, remoteType types.ComponentType) {
	// If already added for a local namespace just return
	for _, restConnections := range mc.RemoteRests {
		for _, rest := range restConnections {
			if rest.Name == restName && rest.LocalNamespace == localNamespace {
				return
			}
		}
	}

}

func getGenericComponentDestinationHost(name string, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace)
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

// Get the datasource name from the database connection in the given domain that targets the given database binding
func getDatasourceName(domain v1beta1v8o.VerrazzanoWebLogicDomain, databaseBinding v1beta1v8o.VerrazzanoDatabaseBinding) string {
	for _, connection := range domain.Connections {
		for _, databaseConnection := range connection.Database {
			if databaseConnection.Target == databaseBinding.Name {
				return databaseConnection.DatasourceName
			}
		}
	}
	return ""
}
