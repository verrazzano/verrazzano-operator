// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	"sync"

	v1cohoperator "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/apis/verrazzano/v1beta1"
	v1cohcluster "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	v1helidonapp "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	v1wlsopr "github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type ComponentType int

const (
	Wls       ComponentType = 0
	Helidon   ComponentType = 1
	Coherence ComponentType = 2
	Unknown   ComponentType = 3
)

type IngressDestination struct {
	Host string
	//VirtualService destination Port
	Port       int
	Match      []MatchRequest
	DomainName string
}

//see verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3/virtual_service.go
//    istio.io/api/networking/v1alpha3/virtual_service.json
type MatchRequest struct {
	Uri  map[string]string //`json:"uri,omitempty"`
	Port int               //`json:"port,omitempty"`
}

type Ingress struct {
	// Name of the ingress in the binding
	Name string

	// Destinations for the ingress
	Destination []*IngressDestination
}

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

	// ConfigMaps for this cluster
	ConfigMaps []*corev1.ConfigMap

	// Remote rest connections (istio ServicEntries) to generate within each namespace for this cluster
	RemoteRests map[string][]*RemoteRestConnection

	// wls-operator deployment for this cluster
	WlsOperator *v1wlsopr.WlsOperator

	// WebLogic domain CRs for this cluster
	WlsDomainCRs []*v8weblogic.Domain

	// Helidon applications for this cluster
	HelidonApps []*v1helidonapp.HelidonApp

	// Coherence operator for this cluster
	CohOperatorCRs []*v1cohoperator.CohCluster

	// Coherence cluster CRs for this cluster
	CohClusterCRs []*v1cohcluster.CoherenceCluster
}

// ModelBindingPair represents an instance of a model/binding pair and
// the objects for creating the model.
type ModelBindingPair struct {
	// A binding and the model it is associated with
	Model   *v1beta1v8o.VerrazzanoModel
	Binding *v1beta1v8o.VerrazzanoBinding

	// The set of managed clusters
	ManagedClusters map[string]*ManagedCluster

	// Lock for synchronizing write access
	Lock sync.RWMutex

	VerrazzanoUri   string
	ImagePullSecret string
}
