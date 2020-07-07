// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"io/ioutil"
	"os"

	"github.com/verrazzano/verrazzano-operator/pkg/types"

	cohoprclientset "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/clientset/versioned"
	cohoprinformers "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/informers/externalversions"
	helidionclientset "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/clientset/versioned"
	helidoninformers "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/informers/externalversions"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	informers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/informers/externalversions"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned"
	istioclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/clientset/versioned"
	istioInformers "github.com/verrazzano/verrazzano-crd-generator/pkg/clientistio/informers/externalversions"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned"
	wlsoprclientset "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned"
	wlsoprinformers "github.com/verrazzano/verrazzano-wko-operator/pkg/client/informers/externalversions"
	istioauthclientset "istio.io/client-go/pkg/clientset/versioned"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Builds a ManagedClusterConnection for the given KubeConfig contents
func BuildManagedClusterConnection(kubeConfigContents []byte, stopCh <-chan struct{}) (*util.ManagedClusterConnection, error) {
	managedClusterConnection := &util.ManagedClusterConnection{}

	// Create a temporary kubeconfig file on disk
	tmpFile, err := ioutil.TempFile("/tmp", "kubeconfig")
	err = ioutil.WriteFile(tmpFile.Name(), kubeConfigContents, 0777)
	defer os.Remove(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	// Build client connections
	managedClusterConnection.KubeConfig = string(kubeConfigContents)
	cfg, err := clientcmd.BuildConfigFromFlags("", tmpFile.Name())
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.KubeClient = clientSet

	// Build client connections for verrazzanoOperatorClientSet
	verrazzanoOperatorClientSet, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.VerrazzanoOperatorClientSet = verrazzanoOperatorClientSet

	// Build client connections for wlsOperatorClientSet
	wlsoprClientset, err := wlsoprclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.WlsOprClientSet = wlsoprClientset

	// Build client connections for domainClientSet
	domainClientSet, err := domclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.DomainClientSet = domainClientSet

	// Build client connections for helidonClientSet
	helidonClientSet, err := helidionclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.HelidonClientSet = helidonClientSet

	// Build client connections for cohOprClientSet
	cohClientSet, err := cohoprclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.CohOprClientSet = cohClientSet

	// Build client connections for domainClientSet
	cohCluClientSet, err := cohcluclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.CohClusterClientSet = cohCluClientSet

	// Build client connections for istioClient
	istioClientSet, err := istioclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.IstioClientSet = istioClientSet

	// Build client connections for istioAuthClient
	istioAuthClientSet, err := istioauthclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.IstioAuthClientSet = istioAuthClientSet

	kubeClientExt, err := extclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.KubeExtClientSet = kubeClientExt

	// Informers on core k8s objects
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(clientSet, constants.ResyncPeriod)

	deploymentInformer := kubeInformerFactory.Apps().V1().Deployments()
	managedClusterConnection.DeploymentInformer = deploymentInformer.Informer()
	managedClusterConnection.DeploymentLister = deploymentInformer.Lister()

	podInformer := kubeInformerFactory.Core().V1().Pods()
	managedClusterConnection.PodInformer = podInformer.Informer()
	managedClusterConnection.PodLister = podInformer.Lister()

	serviceAccountInformer := kubeInformerFactory.Core().V1().ServiceAccounts()
	managedClusterConnection.ServiceAccountInformer = serviceAccountInformer.Informer()
	managedClusterConnection.ServiceAccountLister = serviceAccountInformer.Lister()

	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	managedClusterConnection.NamespaceInformer = namespaceInformer.Informer()
	managedClusterConnection.NamespaceLister = namespaceInformer.Lister()
	managedClusterConnection.SecretInformer = secretInformer.Informer()
	managedClusterConnection.SecretLister = secretInformer.Lister()

	clusterRoleInformer := kubeInformerFactory.Rbac().V1().ClusterRoles()
	managedClusterConnection.ClusterRoleInformer = clusterRoleInformer.Informer()
	managedClusterConnection.ClusterRoleLister = clusterRoleInformer.Lister()

	clusterRoleBindingInformer := kubeInformerFactory.Rbac().V1().ClusterRoleBindings()
	managedClusterConnection.ClusterRoleBindingInformer = clusterRoleBindingInformer.Informer()
	managedClusterConnection.ClusterRoleBindingLister = clusterRoleBindingInformer.Lister()

	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	managedClusterConnection.ConfigMapInformer = configMapInformer.Informer()
	managedClusterConnection.ConfigMapLister = configMapInformer.Lister()

	daemonSetInformer := kubeInformerFactory.Apps().V1().DaemonSets()
	managedClusterConnection.DaemonSetInformer = daemonSetInformer.Informer()
	managedClusterConnection.DaemonSetLister = daemonSetInformer.Lister()

	serviceInformer := kubeInformerFactory.Core().V1().Services()
	managedClusterConnection.ServiceInformer = serviceInformer.Informer()
	managedClusterConnection.ServiceLister = serviceInformer.Lister()

	go kubeInformerFactory.Start(stopCh)

	// Informers on our CRs
	verrazzanoOperatorInformerFactory := informers.NewSharedInformerFactory(verrazzanoOperatorClientSet, constants.ResyncPeriod)
	go verrazzanoOperatorInformerFactory.Start(stopCh)

	// Informers on WlsOperator CRs
	wlsOperatorInformerFactory := wlsoprinformers.NewSharedInformerFactory(wlsoprClientset, constants.ResyncPeriod)
	wlsoprInformer := wlsOperatorInformerFactory.Verrazzano().V1beta1().WlsOperators()
	managedClusterConnection.WlsOperatorInformer = wlsoprInformer.Informer()
	managedClusterConnection.WlsOperatorLister = wlsoprInformer.Lister()
	go wlsOperatorInformerFactory.Start(stopCh)

	// Informers on Domain CRs - delay setup until weblogic-operator is deployed

	// Informers on Helidion App CRs
	helidionOperatorInformerFactory := helidoninformers.NewSharedInformerFactory(helidonClientSet, constants.ResyncPeriod)
	helidonInformer := helidionOperatorInformerFactory.Verrazzano().V1beta1().HelidonApps()
	managedClusterConnection.HelidonInformer = helidonInformer.Informer()
	managedClusterConnection.HelidonLister = helidonInformer.Lister()
	go helidionOperatorInformerFactory.Start(stopCh)

	// Informers on Coherence Operator CRs
	cohOperatorInformerFactory := cohoprinformers.NewSharedInformerFactory(cohClientSet, constants.ResyncPeriod)
	cohInformer := cohOperatorInformerFactory.Verrazzano().V1beta1().CohClusters()
	managedClusterConnection.CohOperatorInformer = cohInformer.Informer()
	managedClusterConnection.CohOperatorLister = cohInformer.Lister()
	go cohOperatorInformerFactory.Start(stopCh)

	// Informers on Coherence Cluster CRs - delay setup until coherence-operator is deployed

	// Informers on istio CRs
	istioOperatorInformerFactory := istioInformers.NewSharedInformerFactory(istioClientSet, constants.ResyncPeriod)
	gatewayInformer := istioOperatorInformerFactory.Networking().V1alpha3().Gateways()
	managedClusterConnection.IstioGatewayInformer = gatewayInformer.Informer()
	managedClusterConnection.IstioGatewayLister = gatewayInformer.Lister()
	virtualServiceInformer := istioOperatorInformerFactory.Networking().V1alpha3().VirtualServices()
	managedClusterConnection.IstioVirtualServiceInformer = virtualServiceInformer.Informer()
	managedClusterConnection.IstioVirtualServiceLister = virtualServiceInformer.Lister()
	serviceEntryInformer := istioOperatorInformerFactory.Networking().V1alpha3().ServiceEntries()
	managedClusterConnection.IstioServiceEntryInformer = serviceEntryInformer.Informer()
	managedClusterConnection.IstioServiceEntryLister = serviceEntryInformer.Lister()
	go istioOperatorInformerFactory.Start(stopCh)

	return managedClusterConnection, nil
}

// Given a map of available ManagedClusterConnections, returns a filtered set of those NOT applicable to the given VerrazzanoBinding
func GetFilteredConnections(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) (map[string]*util.ManagedClusterConnection, error) {
	var filteredConnections map[string]*util.ManagedClusterConnection
	var err error
	// Include the management cluster in case of System binding
	if mbPair.Binding.Name == constants.VmiSystemBindingName {
		filteredConnections = availableManagedClusterConnections
	} else {
		// Parse out the managed clusters that this binding applies to
		filteredConnections, err = util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
		if err != nil {
			return nil, err
		}
	}
	return filteredConnections, nil
}
