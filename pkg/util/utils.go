// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	cohoprclientset "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/clientset/versioned"
	cohoprlister "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/listers/verrazzano/v1beta1"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned"
	cohclulister "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/listers/coherence/v1"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned"
	domlister "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/listers/weblogic/v8"
	helidonclientset "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/clientset/versioned"
	helidionlister "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/listers/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/assets"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	wlsoprclientset "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned"
	wlsoprlister "github.com/verrazzano/verrazzano-wko-operator/pkg/client/listers/verrazzano/v1beta1"
	"go.uber.org/zap"
	istioAuthClientset "istio.io/client-go/pkg/clientset/versioned"
	istioLister "istio.io/client-go/pkg/listers/networking/v1alpha3"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"
	appslistersv1 "k8s.io/client-go/listers/apps/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

// ManagedClusterConnection maintains the connection details to a ManagedCluster.
type ManagedClusterConnection struct {
	KubeClient                  kubernetes.Interface
	KubeExtClientSet            apiextensionsclient.Interface
	VerrazzanoOperatorClientSet clientset.Interface
	WlsOprClientSet             wlsoprclientset.Interface
	DomainClientSet             domclientset.Interface
	HelidonClientSet            helidonclientset.Interface
	CohOprClientSet             cohoprclientset.Interface
	CohClusterClientSet         cohcluclientset.Interface
	IstioClientSet              istioAuthClientset.Interface
	IstioAuthClientSet          istioAuthClientset.Interface
	KubeConfig                  string
	Lock                        sync.RWMutex
	DeploymentLister            appslistersv1.DeploymentLister
	DeploymentInformer          cache.SharedIndexInformer
	PodLister                   corelistersv1.PodLister
	PodInformer                 cache.SharedIndexInformer
	ServiceAccountLister        corelistersv1.ServiceAccountLister
	ServiceAccountInformer      cache.SharedIndexInformer
	NamespaceLister             corelistersv1.NamespaceLister
	NamespaceInformer           cache.SharedIndexInformer
	SecretLister                corelistersv1.SecretLister
	SecretInformer              cache.SharedIndexInformer
	ClusterRoleLister           rbaclistersv1.ClusterRoleLister
	ClusterRoleInformer         cache.SharedIndexInformer
	ClusterRoleBindingLister    rbaclistersv1.ClusterRoleBindingLister
	ClusterRoleBindingInformer  cache.SharedIndexInformer
	WlsOperatorLister           wlsoprlister.WlsOperatorLister
	WlsOperatorInformer         cache.SharedIndexInformer
	DomainLister                domlister.DomainLister
	DomainInformer              cache.SharedIndexInformer
	HelidonLister               helidionlister.HelidonAppLister
	HelidonInformer             cache.SharedIndexInformer
	CohOperatorLister           cohoprlister.CohClusterLister
	CohOperatorInformer         cache.SharedIndexInformer
	CohClusterLister            cohclulister.CoherenceClusterLister
	CohClusterInformer          cache.SharedIndexInformer
	IstioGatewayLister          istioLister.GatewayLister
	IstioGatewayInformer        cache.SharedIndexInformer
	IstioVirtualServiceLister   istioLister.VirtualServiceLister
	IstioVirtualServiceInformer cache.SharedIndexInformer
	IstioServiceEntryLister     istioLister.ServiceEntryLister
	IstioServiceEntryInformer   cache.SharedIndexInformer
	ConfigMapLister             corelistersv1.ConfigMapLister
	ConfigMapInformer           cache.SharedIndexInformer
	DaemonSetLister             appslistersv1.DaemonSetLister
	DaemonSetInformer           cache.SharedIndexInformer
	ServiceLister               corelistersv1.ServiceLister
	ServiceInformer             cache.SharedIndexInformer
}

// Manifest maintains the concrete set of images for components the Verrazzano Operator launches.
type Manifest struct {
	WlsMicroOperatorImage   string `json:"wlsMicroOperatorImage"`
	WlsMicroOperatorCrd     string `json:"wlsMicroOperatorCrd"`
	HelidonAppOperatorImage string `json:"helidonAppOperatorImage"`
	HelidonAppOperatorCrd   string `json:"helidonAppOperatorCrd"`
	CohClusterOperatorImage string `json:"cohClusterOperatorImage"`
	CohClusterOperatorCrd   string `json:"cohClusterOperatorCrd"`
}

// DeploymentHelper defines an interface for deployments.
type DeploymentHelper interface {
	DeleteDeployment(namespace, name string) error
}

// GetManagedBindingLabels returns binding labels for managed cluster.
func GetManagedBindingLabels(binding *v1beta1v8o.VerrazzanoBinding, managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.VerrazzanoGroup, constants.VerrazzanoBinding: binding.Name, constants.VerrazzanoCluster: managedClusterName}
}

// GetManagedLabelsNoBinding return labels for managed cluster with no binding.
func GetManagedLabelsNoBinding(managedClusterName string) map[string]string {
	return map[string]string{constants.K8SAppLabel: constants.VerrazzanoGroup, constants.VerrazzanoCluster: managedClusterName}
}

// GetManagedNamespaceForBinding return the namespace for a given binding.
func GetManagedNamespaceForBinding(binding *v1beta1v8o.VerrazzanoBinding) string {
	return fmt.Sprintf("%s-%s", constants.VerrazzanoPrefix, binding.Name)
}

// GetLocalBindingLabels returns binding labels for local cluster.
func GetLocalBindingLabels(binding *v1beta1v8o.VerrazzanoBinding) map[string]string {
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
func GetManagedClustersForVerrazzanoBinding(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*ManagedClusterConnection) (
	map[string]*ManagedClusterConnection, error) {
	filteredManagedClusters := map[string]*ManagedClusterConnection{}
	for _, managedCluster := range mbPair.ManagedClusters {
		if _, ok := availableManagedClusterConnections[managedCluster.Name]; !ok {
			return nil, fmt.Errorf("Managed cluster %s referenced in binding %s not found", managedCluster.Name, mbPair.Binding.Name)
		}
		filteredManagedClusters[managedCluster.Name] = availableManagedClusterConnections[managedCluster.Name]

	}
	return filteredManagedClusters, nil
}

// GetManagedClustersNotForVerrazzanoBinding returns a filtered set of those NOT applicable to a given
// VerrazzanoBinding given a map of available ManagedClusterConnections.
func GetManagedClustersNotForVerrazzanoBinding(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*ManagedClusterConnection) map[string]*ManagedClusterConnection {
	filteredManagedClusters := map[string]*ManagedClusterConnection{}
	for clusterName := range availableManagedClusterConnections {
		found := false
		for _, managedCluster := range mbPair.ManagedClusters {
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

// LoadManifest loads and verifies the Verrazzano Operator Manifest.
func LoadManifest() (*Manifest, error) {
	manifestString, err := assets.Asset(constants.ManifestFile)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	err = json.Unmarshal(manifestString, &manifest)
	if err != nil {
		return nil, err
	}
	allErrs := field.ErrorList{}

	// Get the image info from ENV vars, defaulting to manifest.yaml values
	manifest.CohClusterOperatorImage = getCohMicroImage()
	manifest.HelidonAppOperatorImage = getHelidonMicroImage()
	manifest.WlsMicroOperatorImage = getWlsMicroImage()

	if manifest.WlsMicroOperatorImage == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("WlsMicroOperatorImage"), ""))
	}
	if manifest.WlsMicroOperatorCrd == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("WlsMicroOperatorCrd"), ""))
	}
	if manifest.HelidonAppOperatorImage == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("HelidonAppOperatorImage"), ""))
	}
	if manifest.HelidonAppOperatorCrd == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("HelidonAppOperatorCrd"), ""))
	}
	if manifest.CohClusterOperatorImage == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("CohClusterOperatorImage"), ""))
	}
	if manifest.CohClusterOperatorCrd == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("CohClusterOperatorCrd"), ""))
	}
	return &manifest, allErrs.ToAggregate()
}

// IsClusterInBinding checks if a cluster was found in a binding.
func IsClusterInBinding(clusterName string, allMbPairs map[string]*types.ModelBindingPair) bool {
	for _, mb := range allMbPairs {
		for _, placement := range mb.Binding.Spec.Placement {
			if placement.Name == clusterName {
				return true
			}
		}
	}
	return false
}

// GetComponentNamespace finds the namespace for the component from the given binding placements.
func GetComponentNamespace(componentName string, binding *v1beta1v8o.VerrazzanoBinding) (string, error) {
	for _, placement := range binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			for _, component := range namespace.Components {
				if component.Name == componentName {
					return namespace.Name, nil
				}
			}
		}
	}
	return "", fmt.Errorf("No placement found for component %s", componentName)
}

// IsDevProfile return true if the installProfile env var is set to dev
func IsDevProfile() bool {
	installProfile, present := os.LookupEnv("INSTALL_PROFILE")
	if present {
		zap.S().Infof("Env var INSTALL_PROFILE = %s", installProfile)
		if installProfile == constants.DevelopmentProfile {
			return true
		}
	}
	return false
}

// GetProfileBindingName will return the binding name based on the profile
// if the profile doesn't have a special binding name, the binding name supplied is returned
// for Dev profile the VMI system binding name is returned
func GetProfileBindingName(bindingName string) string {
	if IsDevProfile() {
		return constants.VmiSystemBindingName
	}
	return bindingName
}

// Remove duplicates from a slice containing string values
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