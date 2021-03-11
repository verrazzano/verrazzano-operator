// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"io/ioutil"
	"os"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
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

var newExtClientSet = func(c *rest.Config) (extclientset.Interface, error) {
	clientSet, err := extclientset.NewForConfig(c)
	return clientSet, err
}

var buildConfigFromFlags = clientcmd.BuildConfigFromFlags
var osRemove = os.Remove
var ioWriteFile = ioutil.WriteFile

// BuildManagedClusterConnection builds a ManagedClusterConnection for the given KubeConfig contents.
func BuildManagedClusterConnection(kubeconfigPath string, stopCh <-chan struct{}) (*util.ManagedClusterConnection, error) {
	managedClusterConnection := &util.ManagedClusterConnection{}

	// Build client connections
	// NOTE: Passing empty strings here results in the client falling back to the in-cluster config
	cfg, err := buildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	clientSet, err := newKubernetesClientSet(cfg)
	if err != nil {
		return nil, err
	}
	managedClusterConnection.KubeClient = clientSet

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

	return managedClusterConnection, nil
}

// GetFilteredConnections given a map of available ManagedClusterConnections, returns a filtered set of those NOT
// applicable to the given VerrazzanoBinding.
func GetFilteredConnections(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) (map[string]*util.ManagedClusterConnection, error) {
	var filteredConnections map[string]*util.ManagedClusterConnection
	var err error
	// Include the management cluster in case of System binding
	if vzSynMB.SynBinding.Name == constants.VmiSystemBindingName {
		filteredConnections = availableManagedClusterConnections
	} else {
		// Parse out the managed clusters that this binding applies to
		filteredConnections, err = util.GetManagedClustersForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)
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
