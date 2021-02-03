// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ComponentType component type
type ComponentType int

const (
	// Unknown component type
	Unknown ComponentType = 4
)

// IngressDestination represents a destination ingress.
type IngressDestination struct {
	Host string
	// VirtualService destination Port
	Port       uint32
	Match      []MatchRequest
	DomainName string
}

// MatchRequest represents an istio virtual service match criteria.
// see verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3/virtual_service.go
//    istio.io/api/networking/v1alpha3/virtual_service.json
type MatchRequest struct {
	URI  map[string]string //`json:"uri,omitempty"`
	Port int               //`json:"port,omitempty"`
}

// Ingress represents an ingress and its destinations.
type Ingress struct {
	// Name of the ingress in the binding
	Name string

	// Destinations for the ingress
	Destination []*IngressDestination
}

// RemoteRestConnection represents a rest connection to a remote cluster.
type RemoteRestConnection struct {
	// Name of remote service
	Name string

	// Namespace for remote service
	RemoteNamespace string

	// Cluster name for remote service
	RemoteClusterName string

	// Namespace for local service
	LocalNamespace string

	// Port for remote service
	Port uint32
}

// ManagedCluster defines the environment of a single managed cluster
// within the model.
type ManagedCluster struct {
	// Name of the managed cluster
	Name string

	// Namespaces within the cluster
	Namespaces []string

	// Names of secrets within each namespace for the cluster
	Secrets map[string][]string

	// Names of ingresses to generate within each namespace for this cluster
	Ingresses map[string][]*Ingress

	// Deployments for this cluster
	Deployments []*appsv1.Deployment

	// Services for this cluster
	Services []*corev1.Service

	// ConfigMaps for this cluster
	ConfigMaps []*corev1.ConfigMap

	// Remote rest connections (istio ServicEntries) to generate within each namespace for this cluster
	RemoteRests map[string][]*RemoteRestConnection

}

type ClusterModel struct {
	metav1.ObjectMeta
}

// ModelBindingPair represents an instance of a model/binding pair and
// the objects for creating the model.
type ModelBindingPair struct {
	Binding *ClusterBinding
	Model *ClusterModel

	// The set of managed clusters
	ManagedClusters map[string]*ManagedCluster

	// Lock for synchronizing write access
	Lock *sync.RWMutex

	VerrazzanoURI string

	// Optional list of image pull secrets to add to service accounts created by the operator
	ImagePullSecrets []corev1.LocalObjectReference
}

type ClusterBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoBindingSpec   `json:"spec"`
}

type VerrazzanoBindingSpec struct {
	// A description of the binding
	Description string `json:"description" yaml:"description"`

	// The model name to associate the bindings
	ModelName string `json:"modelName" yaml:"modelName"`

	// The set of Placement definitions
	// +x-kubernetes-list-type=set
	Placement []VerrazzanoPlacement `json:"placement" yaml:"placement"`
}

type VerrazzanoPlacement struct {
	// The name of the placement
	Name string `json:"name" yaml:"name"`

	// Namespaces for this placement
	// +x-kubernetes-list-type=set
	Namespaces []KubernetesNamespace `json:"namespaces" yaml:"namespaces"`
}

type KubernetesNamespace struct {
	// Name of the namespace
	Name string `json:"name" yaml:"name"`
}
