// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"time"

	"go.uber.org/zap"
	restclient "k8s.io/client-go/rest"

	cohoprclientset "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/clientset/versioned"
	cohoprinformers "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/informers/externalversions"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	informers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/informers/externalversions"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned"
	helidionclientset "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/clientset/versioned"
	helidoninformers "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/informers/externalversions"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	wlsoprclientset "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned"
	wlsoprinformers "github.com/verrazzano/verrazzano-wko-operator/pkg/client/informers/externalversions"
	istioauthclientset "istio.io/client-go/pkg/clientset/versioned"
	istioInformers "istio.io/client-go/pkg/informers/externalversions"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// functions to set client sets
var newKubernetesClientSet = func(c *rest.Config) (kubernetes.Interface, error) {
	clientSet, err := kubernetes.NewForConfig(c)
	return clientSet, err
}

var newVerrazzanoOperatorClientSet = func(c *rest.Config) (clientset.Interface, error) {
	clientSet, err := clientset.NewForConfig(c)
	return clientSet, err
}

var newWLSOperatorClientSet = func(c *rest.Config) (wlsoprclientset.Interface, error) {
	clientSet, err := wlsoprclientset.NewForConfig(c)
	return clientSet, err
}

var newDomainClientSet = func(c *rest.Config) (domclientset.Interface, error) {
	clientSet, err := domclientset.NewForConfig(c)
	return clientSet, err
}

var newHelidonClientSet = func(c *rest.Config) (helidionclientset.Interface, error) {
	clientSet, err := helidionclientset.NewForConfig(c)
	return clientSet, err
}

var newCOHOperatorClientSet = func(c *rest.Config) (cohoprclientset.Interface, error) {
	clientSet, err := cohoprclientset.NewForConfig(c)
	return clientSet, err
}

var newCOHClusterClientSet = func(c *rest.Config) (cohcluclientset.Interface, error) {
	clientSet, err := cohcluclientset.NewForConfig(c)
	return clientSet, err
}

var newIstioClientSet = func(c *rest.Config) (istioauthclientset.Interface, error) {
	clientSet, err := istioauthclientset.NewForConfig(c)
	return clientSet, err
}

var newIstioAuthClientSet = func(c *rest.Config) (istioauthclientset.Interface, error) {
	clientSet, err := istioauthclientset.NewForConfig(c)
	return clientSet, err
}

var newExtClientSet = func(c *rest.Config) (extclientset.Interface, error) {
	clientSet, err := extclientset.NewForConfig(c)
	return clientSet, err
}

var buildConfigFromFlags = clientcmd.BuildConfigFromFlags
var osRemove = os.Remove
var ioWriteFile = ioutil.WriteFile
var createKubeconfig = createTempKubeconfigFile

// When the in-cluster accessible host is different from the outside accessible URL's host (parsedHost),
// do a 'curl --resolve' equivalent
func setupHTTPResolve(cfg *restclient.Config) error {
	rancherURL := util.GetRancherURL()
	if rancherURL != "" {
		urlObj, err := url.Parse(rancherURL)
		if err != nil {
			zap.S().Fatalf("Invalid URL '%s': %v", rancherURL, err)
			return err
		}
		parsedHost := urlObj.Host
		host := util.GetRancherHost()
		if host != "" && host != parsedHost {
			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			cfg.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
				zap.S().Debugf("address original: %s \n", addr)
				if addr == parsedHost+":443" {
					addr = host + ":443"
					zap.S().Debugf("address modified: %s \n", addr)
				}
				return dialer.DialContext(ctx, network, addr)
			}
		}
	}

	return nil
}

// BuildManagedClusterConnection builds a ManagedClusterConnection for the given KubeConfig contents.
func BuildManagedClusterConnection(kubeConfigContents []byte, stopCh <-chan struct{}) (*util.ManagedClusterConnection, error) {
	managedClusterConnection := &util.ManagedClusterConnection{}

	// Create a temporary kubeconfig file on disk
	tmpFileName, err := createKubeconfig()
	if err != nil {
		return nil, err
	}
	err = ioWriteFile(tmpFileName, kubeConfigContents, 0777)
	defer osRemove(tmpFileName)
	if err != nil {
		return nil, err
	}

	// Build client connections
	managedClusterConnection.KubeConfig = string(kubeConfigContents)
	cfg, err := buildConfigFromFlags("", tmpFileName)
	if err != nil {
		return nil, err
	}
	setupHTTPResolve(cfg)
	if err != nil {
		return nil, err
	}
	clientSet, err := newKubernetesClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.KubeClient = clientSet

	// Build client connections for verrazzanoOperatorClientSet
	verrazzanoOperatorClientSet, err := newVerrazzanoOperatorClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.VerrazzanoOperatorClientSet = verrazzanoOperatorClientSet

	// Build client connections for wlsOperatorClientSet
	wlsoprClientset, err := newWLSOperatorClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.WlsOprClientSet = wlsoprClientset

	// Build client connections for domainClientSet
	domainClientSet, err := newDomainClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.DomainClientSet = domainClientSet

	// Build client connections for helidonClientSet
	helidonClientSet, err := newHelidonClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.HelidonClientSet = helidonClientSet

	// Build client connections for cohOprClientSet
	cohClientSet, err := newCOHOperatorClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.CohOprClientSet = cohClientSet

	// Build client connections for cohCluClientSet
	cohCluClientSet, err := newCOHClusterClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.CohClusterClientSet = cohCluClientSet

	// Build client connections for istioClient
	istioClientSet, err := newIstioClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.IstioClientSet = istioClientSet

	// Build client connections for istioAuthClient
	istioAuthClientSet, err := newIstioAuthClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.IstioAuthClientSet = istioAuthClientSet

	kubeClientExt, err := newExtClientSet(cfg)
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

// GetFilteredConnections given a map of available ManagedClusterConnections, returns a filtered set of those NOT
// applicable to the given VerrazzanoBinding.
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

func createTempKubeconfigFile() (string, error) {
	tmpFile, err := ioutil.TempFile("/tmp", "kubeconfig")
	if err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}
