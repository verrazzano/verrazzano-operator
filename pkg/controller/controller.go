// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"k8s.io/client-go/rest"

	v8omonitoring "github.com/verrazzano/verrazzano-operator/pkg/monitoring"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	clientsetscheme "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/scheme"
	informers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/informers/externalversions"
	listers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/listers/verrazzano/v1beta1"
	vmoclientset "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	vmoinformers "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/informers/externalversions"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	v8olocal "github.com/verrazzano/verrazzano-operator/pkg/local"
	v8omanaged "github.com/verrazzano/verrazzano-operator/pkg/managed"
	"github.com/verrazzano/verrazzano-operator/pkg/signals"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	v8outil "github.com/verrazzano/verrazzano-operator/pkg/util"
	wlsoprclientset "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned"
	wlsoprinformers "github.com/verrazzano/verrazzano-wko-operator/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

const controllerAgentName = "verrazzano-controller"
const bindingFinalizer = "vb.verrazzano.io"

// setupSignalHandler is a function to create signal handler during construction of a new Controller
var setupSignalHandler = signals.SetupSignalHandler

// buildKubeClientSet is a function to build Kube client from config during construction of a new Controller
var buildKubeClientSet = buildKubeClientsetFromConfig

// executeCreateUpdateGlobalEntities is a function which creates and updates various global entities in a goroutine
var executeCreateUpdateGlobalEntities = executeCreateUpdateGlobalEntitiesGoroutine

// newManagedPackage is a function to obtain managed package implementation during construction of a new Controller
var newManagedPackage = func() managedInterface {
	return &managedPackage{}
}

// newCachePackage is a function to get cache package implementation
var newCachePackage = func() cacheInterface {
	return &cachePackage{}
}

// newUtilPackage is a function to get util package implementation
var newUtilPackage = func() utilInterface {
	return &utilPackage{}
}

// newLocalPackage is a function to get local package implementation
var newLocalPackage = func() localInterface {
	return &localPackage{}
}

// newMonitoringPackage is a function to get monitoring package implementation
var newMonitoringPackage = func() monitoringInterface {
	return &monitoringPackage{}
}

// Controller represents the primary controller structure.
type Controller struct {
	managed                     managedInterface
	cache                       cacheInterface
	util                        utilInterface
	local                       localInterface
	monitoring                  monitoringInterface
	kubeClientSet               kubernetes.Interface
	kubeExtClientSet            apiextensionsclient.Interface
	verrazzanoOperatorClientSet clientset.Interface
	vmoClientSet                vmoclientset.Interface
	verrazzanoURI               string
	enableMonitoringStorage     string
	imagePullSecrets            []corev1.LocalObjectReference

	// Local cluster listers and informers
	secretLister                     corev1listers.SecretLister
	secretInformer                   k8scache.SharedIndexInformer
	configMapLister                  corev1listers.ConfigMapLister
	configMapInformer                k8scache.SharedIndexInformer
	verrazzanoManagedClusterLister   listers.VerrazzanoManagedClusterLister
	verrazzanoManagedClusterInformer k8scache.SharedIndexInformer
	VerrazzanoModelLister            listers.VerrazzanoModelLister
	VerrazzanoModelInformer          k8scache.SharedIndexInformer
	VerrazzanoBindingLister          listers.VerrazzanoBindingLister
	VerrazzanoBindingInformer        k8scache.SharedIndexInformer
	vmiLister                        vmolisters.VerrazzanoMonitoringInstanceLister
	vmiInformer                      k8scache.SharedIndexInformer
	secrets                          v8omonitoring.Secrets
	// The current set of known managed clusters
	managedClusterConnections map[string]*v8outil.ManagedClusterConnection

	// The current set of known models
	applicationModels map[string]*v1beta1v8o.VerrazzanoModel

	// The current set of known bindings
	applicationBindings map[string]*v1beta1v8o.VerrazzanoBinding

	// The current set of known model/binding pairs
	modelBindingPairs map[string]*types.ModelBindingPair

	// Misc
	watchNamespace string
	stopCh         <-chan struct{}

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	Manifest *v8outil.Manifest

	// Keep track of no. of times add/update events are encountered for unchanged bindings
	bindingSyncThreshold map[string]int
}

// Listers represents listers used by the controller.
type Listers struct {
	ManagedClusterLister *listers.VerrazzanoManagedClusterLister
	ModelLister          *listers.VerrazzanoModelLister
	BindingLister        *listers.VerrazzanoBindingLister
	ModelBindingPairs    *map[string]*types.ModelBindingPair
	KubeClientSet        *kubernetes.Interface
}

// ListerSet returns a list of listers used by the controller.
func (c *Controller) ListerSet() Listers {
	return Listers{
		ManagedClusterLister: &c.verrazzanoManagedClusterLister,
		ModelLister:          &c.VerrazzanoModelLister,
		BindingLister:        &c.VerrazzanoBindingLister,
		ModelBindingPairs:    &c.modelBindingPairs,
		KubeClientSet:        &c.kubeClientSet,
	}
}

// NewController returns a new Verrazzano Operator controller
func NewController(config *rest.Config, manifest *v8outil.Manifest, watchNamespace string, verrazzanoURI string, enableMonitoringStorage string) (*Controller, error) {
	zap.S().Debugw("Building kubernetes clientset")
	kubeClientSet, err := buildKubeClientSet(config)
	if err != nil {
		zap.S().Fatalf("Error building kubernetes clientset: %v", err)
	}

	zap.S().Debugw("Building Verrazzano Operator clientset")
	verrazzanoOperatorClientSet, err := clientset.NewForConfig(config)
	if err != nil {
		zap.S().Fatalf("Error building verrazzano operator clientset: %v", err)
	}

	zap.S().Debugw("Building VMO clientset")
	vmoClientSet, err := vmoclientset.NewForConfig(config)
	if err != nil {
		zap.S().Fatalf("Error building VMO clientset: %v", err)
	}

	zap.S().Debugw("Building wls-operator clientset")
	wlsoprClientSet, err := wlsoprclientset.NewForConfig(config)
	if err != nil {
		zap.S().Fatalf("Error building wls-operator clientset: %v", err)
	}

	zap.S().Debugw("Building api extensions clientset")
	kubeExtClientSet, err := extclientset.NewForConfig(config)
	if err != nil {
		zap.S().Fatalf("Error building apiextensions-apiserver clientset: %v", err)
	}

	//
	// Set up informers and listers for the local k8s cluster
	//
	var kubeInformerFactory kubeinformers.SharedInformerFactory
	var verrazzanoOperatorInformerFactory informers.SharedInformerFactory
	var vmoInformerFactory vmoinformers.SharedInformerFactory
	var wlsoprInformerFactory wlsoprinformers.SharedInformerFactory
	if watchNamespace == "" {
		// Consider all namespaces if our namespace is left wide open our set to default
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeClientSet, constants.ResyncPeriod)
		verrazzanoOperatorInformerFactory = informers.NewSharedInformerFactory(verrazzanoOperatorClientSet, constants.ResyncPeriod)
		wlsoprInformerFactory = wlsoprinformers.NewSharedInformerFactory(wlsoprClientSet, constants.ResyncPeriod)
		vmoInformerFactory = vmoinformers.NewSharedInformerFactory(vmoClientSet, constants.ResyncPeriod)
	} else {
		// Otherwise, restrict to a specific namespace
		kubeInformerFactory = kubeinformers.NewFilteredSharedInformerFactory(kubeClientSet, constants.ResyncPeriod, watchNamespace, nil)
		verrazzanoOperatorInformerFactory = informers.NewFilteredSharedInformerFactory(verrazzanoOperatorClientSet, constants.ResyncPeriod, watchNamespace, nil)
		wlsoprInformerFactory = wlsoprinformers.NewFilteredSharedInformerFactory(wlsoprClientSet, constants.ResyncPeriod, watchNamespace, nil)
		vmoInformerFactory = vmoinformers.NewFilteredSharedInformerFactory(vmoClientSet, constants.ResyncPeriod, watchNamespace, nil)
	}
	secretsInformer := kubeInformerFactory.Core().V1().Secrets()
	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	verrazzanoManagedClusterInformer := verrazzanoOperatorInformerFactory.Verrazzano().V1beta1().VerrazzanoManagedClusters()
	VerrazzanoBindingInformer := verrazzanoOperatorInformerFactory.Verrazzano().V1beta1().VerrazzanoBindings()
	VerrazzanoModelInformer := verrazzanoOperatorInformerFactory.Verrazzano().V1beta1().VerrazzanoModels()
	vmiInformer := vmoInformerFactory.Verrazzano().V1().VerrazzanoMonitoringInstances()
	vmiInformer.Informer().AddEventHandler(k8scache.ResourceEventHandlerFuncs{})

	clientsetscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(zap.S().Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	kubeSecrets := &KubeSecrets{
		namespace: constants.VerrazzanoNamespace, kubeClientSet: kubeClientSet, secretLister: secretsInformer.Lister(),
	}
	controller := &Controller{
		managed:                          newManagedPackage(),
		cache:                            newCachePackage(),
		util:                             newUtilPackage(),
		local:                            newLocalPackage(),
		monitoring:                       newMonitoringPackage(),
		watchNamespace:                   watchNamespace,
		verrazzanoURI:                    verrazzanoURI,
		enableMonitoringStorage:          enableMonitoringStorage,
		kubeClientSet:                    kubeClientSet,
		verrazzanoOperatorClientSet:      verrazzanoOperatorClientSet,
		vmoClientSet:                     vmoClientSet,
		kubeExtClientSet:                 kubeExtClientSet,
		secretLister:                     secretsInformer.Lister(),
		secretInformer:                   secretsInformer.Informer(),
		configMapLister:                  configMapInformer.Lister(),
		configMapInformer:                configMapInformer.Informer(),
		verrazzanoManagedClusterLister:   verrazzanoManagedClusterInformer.Lister(),
		verrazzanoManagedClusterInformer: verrazzanoManagedClusterInformer.Informer(),
		VerrazzanoModelLister:            VerrazzanoModelInformer.Lister(),
		VerrazzanoModelInformer:          VerrazzanoModelInformer.Informer(),
		VerrazzanoBindingLister:          VerrazzanoBindingInformer.Lister(),
		VerrazzanoBindingInformer:        VerrazzanoBindingInformer.Informer(),
		vmiInformer:                      vmiInformer.Informer(),
		vmiLister:                        vmiInformer.Lister(),
		recorder:                         recorder,
		managedClusterConnections:        map[string]*v8outil.ManagedClusterConnection{},
		applicationModels:                map[string]*v1beta1v8o.VerrazzanoModel{},
		applicationBindings:              map[string]*v1beta1v8o.VerrazzanoBinding{},
		modelBindingPairs:                map[string]*types.ModelBindingPair{},
		Manifest:                         manifest,
		bindingSyncThreshold:             map[string]int{},
		secrets:                          kubeSecrets,
	}

	// Set up signals so we handle the first shutdown signal gracefully
	zap.S().Debugw("Setting up signals")
	stopCh := setupSignalHandler()

	go kubeInformerFactory.Start(stopCh)
	go verrazzanoOperatorInformerFactory.Start(stopCh)
	go wlsoprInformerFactory.Start(stopCh)
	go vmoInformerFactory.Start(stopCh)

	// Wait for the caches to be synced before starting watchers
	zap.S().Infow("Waiting for informer caches to sync")
	if ok := controller.cache.WaitForCacheSync(controller.stopCh, controller.secretInformer.HasSynced,
		controller.verrazzanoManagedClusterInformer.HasSynced, controller.VerrazzanoBindingInformer.HasSynced,
		controller.VerrazzanoModelInformer.HasSynced, controller.vmiInformer.HasSynced); !ok {
		return controller, errors.New("failed to wait for caches to sync")
	}

	// Install Global Entities
	executeCreateUpdateGlobalEntities(&v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}, controller)

	return controller, nil
}

// buildKubeClientsetFromConfig builds a kubernetes clientset from configuration
func buildKubeClientsetFromConfig(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}

// executeCreateUpdateGlobalEntitiesGoroutine executes createUpdateGlobalEntitiesGoroutine() in a goroutine
func executeCreateUpdateGlobalEntitiesGoroutine(binding *v1beta1v8o.VerrazzanoBinding, c *Controller) {
	go createUpdateGlobalEntitiesGoroutine(binding, c)
}

// CreateUpdateGlobalEntitiesGoroutine installs global entities and loops forever so it must be called in a goroutine
func createUpdateGlobalEntitiesGoroutine(binding *v1beta1v8o.VerrazzanoBinding, c *Controller) {
	zap.S().Infow("Configuring System VMI...")
	for {
		createUpdateGlobalEntities(binding, c)
		<-time.After(60 * time.Second)
	}
}

func createUpdateGlobalEntities(binding *v1beta1v8o.VerrazzanoBinding, c *Controller) {
	err := c.local.CreateUpdateVmi(binding, c.vmoClientSet, c.vmiLister, c.verrazzanoURI, c.enableMonitoringStorage)
	if err != nil {
		zap.S().Errorf("Failed to create System VMI %s: %v", constants.VmiSystemBindingName, err)
	}

	// Create secrets
	err = c.monitoring.CreateVmiSecrets(binding, c.secrets)
	if err != nil {
		zap.S().Errorf("Failed to create secrets for binding %s: %v", binding.Name, err)
	}

	// Update config maps for system vmi
	err = c.local.UpdateConfigMaps(binding, c.kubeClientSet, c.configMapLister)
	if err != nil {
		zap.S().Errorf("Failed to update ConfigMaps for binding %s: %v", binding.Name, err)
	}
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
//
func (c *Controller) Run(threadiness int) error {
	defer runtime.HandleCrash()

	// Start the informer factories to begin populating the informer caches
	zap.S().Infow("Starting Verrazzano Operator controller")

	zap.S().Infow("Starting watchers")
	c.startLocalWatchers()
	<-c.stopCh
	return nil
}

// Start watchers on the local k8s cluster
func (c *Controller) startLocalWatchers() {

	//
	// VerrazzanoManagedClusters
	//
	c.verrazzanoManagedClusterInformer.AddEventHandler(k8scache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { c.processManagedCluster(new) },
		UpdateFunc: func(old, new interface{}) { c.processManagedCluster(new) },
	})

}

// Process a change to a VerrazzanoManagedCluster
func (c *Controller) processManagedCluster(cluster interface{}) {
	// Obtain the optional list of imagePullSecrets from the verrazzano-operator service account
	sa, err := c.kubeClientSet.CoreV1().ServiceAccounts(constants.VerrazzanoSystem).Get(context.TODO(), constants.VerrazzanoOperatorServiceAccount, metav1.GetOptions{})
	if err != nil {
		zap.S().Errorf("Can't find ServiceAccount %s in namespace %s", constants.VerrazzanoOperatorServiceAccount, constants.VerrazzanoSystem)
		return
	}
	c.imagePullSecrets = sa.ImagePullSecrets

	managedCluster := cluster.(*v1beta1v8o.VerrazzanoManagedCluster)
	secret, err := c.secretLister.Secrets(managedCluster.Namespace).Get(managedCluster.Spec.KubeconfigSecret)
	if err != nil {
		zap.S().Errorf("Can't find secret %s for ManagedCluster %s", managedCluster.Spec.KubeconfigSecret, managedCluster.Name)
		return
	}

	// Construct a new client to the managed cluster when it's added or changed
	kubeConfigContents := secret.Data["kubeconfig"]
	_, clusterExists := c.managedClusterConnections[managedCluster.Name]
	if !clusterExists || string(kubeConfigContents) != c.managedClusterConnections[managedCluster.Name].KubeConfig {
		zap.S().Infof("(Re)creating k8s clients for Managed Cluster %s", managedCluster.Name)

		managedClusterConnection, err := c.managed.BuildManagedClusterConnection(kubeConfigContents, c.stopCh)
		if err != nil {
			zap.S().Error(err)
			return
		}

		c.managedClusterConnections[managedCluster.Name] = managedClusterConnection

		mc := &types.ManagedCluster{
			Name:        managedCluster.Name,
			Secrets:     map[string][]string{},
			Ingresses:   map[string][]*types.Ingress{},
			RemoteRests: map[string][]*types.RemoteRestConnection{},
		}

		// Add in the monitoring and logging namespace if not already added
		mc.Namespaces = append(mc.Namespaces, constants.MonitoringNamespace, constants.LoggingNamespace)

		/*********************
		 * Create Artifacts in the Managed Cluster
		 **********************/
		// Create CRD Definitions
		err = c.managed.CreateCrdDefinitions(c.managedClusterConnections[managedCluster.Name], managedCluster, c.Manifest)
		if err != nil {
			zap.S().Errorf("Failed to create CRD definitions for ManagedCluster %s: %v", managedCluster.Name, err)
		}

		// Wait for the caches to be synced before starting workers
		zap.S().Infow("Waiting for informer caches to sync")
		if ok := c.cache.WaitForCacheSync(c.stopCh, managedClusterConnection.DeploymentInformer.HasSynced,
			managedClusterConnection.NamespaceInformer.HasSynced, managedClusterConnection.SecretInformer.HasSynced); !ok {
			zap.S().Error(errors.New("failed to wait for caches to sync"))
		}
	}
}


// Create managed resources on clusters depending upon the binding
func (c *Controller) createManagedClusterResourcesForBinding(mbPair *types.ModelBindingPair) {
	filteredConnections, err := c.util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to get filtered connections for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Namespaces
	err = c.managed.CreateNamespaces(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create namespaces for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create (copy) the Secrets needed from the management cluster to the managed clusters
	err = c.managed.CreateSecrets(mbPair, c.managedClusterConnections, c.kubeClientSet, c.secrets)
	if err != nil {
		zap.S().Errorf("Failed to create secrets for binding %s: %v", mbPair.Binding.Namespace, err)
	}

	// Create ServiceAccounts
	err = c.managed.CreateServiceAccounts(mbPair.Binding.Name, mbPair.ImagePullSecrets, mbPair.ManagedClusters, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create service accounts for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create ConfigMaps
	err = c.managed.CreateConfigMaps(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create config maps for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create ClusterRoles
	err = c.managed.CreateClusterRoles(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create cluster roles for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create ClusterRoleBindings
	err = c.managed.CreateClusterRoleBindings(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create cluster role bindings for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Ingresses
	err = c.managed.CreateIngresses(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create ingresses for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create istio ServiceEntries for each remote rest connection
	err = c.managed.CreateServiceEntries(mbPair, filteredConnections, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to create service entries for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Deployments
	err = c.managed.CreateDeployments(mbPair, filteredConnections, c.Manifest, c.verrazzanoURI, c.secrets)
	if err != nil {
		zap.S().Errorf("Failed to create deployments for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Services
	err = c.managed.CreateServices(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create service for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Custom Resources
	err = c.managed.CreateCustomResources(mbPair, c.managedClusterConnections, c.stopCh, c.VerrazzanoBindingLister)
	if err != nil {
		zap.S().Errorf("Failed to create custom resources for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Update Istio Prometheus ConfigMaps for a given ModelBindingPair on all managed clusters
	err = c.managed.UpdateIstioPrometheusConfigMaps(mbPair, c.secretLister, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to update Istio Config Map for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create DaemonSets
	err = c.managed.CreateDaemonSets(mbPair, filteredConnections, c.verrazzanoURI)

	if err != nil {
		zap.S().Errorf("Failed to create DaemonSets for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create DestinationRules
	err = c.managed.CreateDestinationRules(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create destination rules for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create AuthorizationPolicies
	err = c.managed.CreateAuthorizationPolicies(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create authorization policies for binding %s: %v", mbPair.Binding.Name, err)
	}
}

// Process a change to a VerrazzanoModel
func (c *Controller) processApplicationModelAdded(verrazzanoModel interface{}) {
	model := verrazzanoModel.(*v1beta1v8o.VerrazzanoModel)

	if existingModel, ok := c.applicationModels[model.Name]; ok {
		if existingModel.GetResourceVersion() == model.GetResourceVersion() {
			zap.S().Debugf("No changes to the model %s", model.Name)
			return
		}

		zap.S().Infof("Updating the model %s", model.Name)
	} else {
		zap.S().Infof("Adding the model %s", model.Name)
	}

}

// Process a removal of a VerrazzanoModel
func (c *Controller) processApplicationModelDeleted(verrazzanoModel interface{}) {
	model := verrazzanoModel.(*v1beta1v8o.VerrazzanoModel)

	if _, ok := c.applicationModels[model.Name]; ok {
		zap.S().Infof("Deleting the model %s", model.Name)
		delete(c.applicationModels, model.Name)
	}
}

func getModelBindingPair(c *Controller, binding *v1beta1v8o.VerrazzanoBinding) (*types.ModelBindingPair, bool) {
	mbPair, mbPairExists := c.modelBindingPairs[binding.Name]
	return mbPair, mbPairExists
}

// Make sure the namespaces in binding are unique across all bindings
func (c *Controller) checkNamespacesFound(binding *v1beta1v8o.VerrazzanoBinding) bool {
	for _, placement := range binding.Spec.Placement {
		for _, currBinding := range c.applicationBindings {
			if binding.Name != currBinding.Name {
				for _, currPlacement := range currBinding.Spec.Placement {
					if placement.Name == currPlacement.Name {
						return checkForDuplicateNamespaces(&placement, binding, &currPlacement, currBinding)
					}
				}
			}
		}
	}

	return false
}

// Check for a duplicate namespaces being used across two differrent bindings
func checkForDuplicateNamespaces(placement *v1beta1v8o.VerrazzanoPlacement, binding *v1beta1v8o.VerrazzanoBinding, currPlacement *v1beta1v8o.VerrazzanoPlacement, currBinding *v1beta1v8o.VerrazzanoBinding) bool {
	dupFound := false
	for _, namespace := range placement.Namespaces {
		for _, currNamespace := range currPlacement.Namespaces {
			if currNamespace.Name == namespace.Name {
				if !dupFound {
					zap.S().Errorf("Binding %s has a conflicting namespace with binding %s.  Namespaces must be unique across bindings.", binding.Name, currBinding.Name)
				}
				zap.S().Errorf("Duplicate namespace %s found in placement %s", namespace.Name, placement.Name)
				dupFound = true
				break
			}
		}
	}

	return dupFound
}

func (c *Controller) cleanupOrphanedResources(mbPair *types.ModelBindingPair) {
	// Cleanup Custom Resources
	err := c.managed.CleanupOrphanedCustomResources(mbPair, c.managedClusterConnections, c.stopCh)
	if err != nil {
		zap.S().Errorf("Failed to cleanup custom resources for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Services for generic components
	err = c.managed.CleanupOrphanedServices(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup services for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Deployments for generic components
	err = c.managed.CleanupOrphanedDeployments(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup deployments for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ServiceEntries
	err = c.managed.CleanupOrphanedServiceEntries(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ServiceEntries for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Ingresses
	err = c.managed.CleanupOrphanedIngresses(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup Ingresses for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ClusterRoleBindings
	err = c.managed.CleanupOrphanedClusterRoleBindings(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ClusterRoleBindings for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ClusterRoles
	err = c.managed.CleanupOrphanedClusterRoles(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ClusterRoles for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ConfigMaps
	err = c.managed.CleanupOrphanedConfigMaps(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ConfigMaps for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Namespaces - this will also cleanup any ServiceAccounts and Secrets within the namespace
	err = c.managed.CleanupOrphanedNamespaces(mbPair, c.managedClusterConnections, c.modelBindingPairs)
	if err != nil {
		zap.S().Errorf("Failed to cleanup Namespaces for binding %s: %v", mbPair.Binding.Name, err)
	}
}

type kubeDeployment struct {
	kubeClientSet kubernetes.Interface
}

func (me *kubeDeployment) DeleteDeployment(namespace, name string) error {
	dep, err := me.kubeClientSet.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if dep != nil {
		return me.kubeClientSet.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	}
	return err
}

// Process a removal of a VerrazzanoBinding
func (c *Controller) processApplicationBindingDeleted(verrazzanoBinding interface{}) {
	binding := verrazzanoBinding.(*v1beta1v8o.VerrazzanoBinding)

	if !contains(binding.GetFinalizers(), bindingFinalizer) {
		zap.S().Infof("Resources for binding %s already deleted", binding.Name)
		return
	}

	mbPair, mbPairExists := getModelBindingPair(c, binding)
	if !mbPairExists {
		return
	}

	zap.S().Infof("Deleting resources for binding %s", binding.Name)

	/*********************
	 * Delete Artifacts in the Local Cluster
	 **********************/
	// Delete VMIs
	err := c.local.DeleteVmi(binding, c.vmoClientSet, c.vmiLister)
	if err != nil {
		zap.S().Errorf("Failed to delete VMIs for binding %s: %v", binding.Name, err)
	}

	// Delete Secrets
	err = c.local.DeleteSecrets(binding, c.kubeClientSet, c.secretLister)
	if err != nil {
		zap.S().Errorf("Failed to delete secrets for binding %s: %v", binding.Name, err)
		return
	}

	// Delete ConfigMaps
	err = c.local.DeleteConfigMaps(binding, c.kubeClientSet, c.configMapLister)
	if err != nil {
		zap.S().Errorf("Failed to delete ConfigMaps for binding %s: %v", binding.Name, err)
		return
	}

	/*********************
	 * Delete Artifacts in the Managed Cluster
	 **********************/

	filteredConnections, err := c.util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to get filtered connections for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Delete Custom Resources
	err = c.managed.DeleteCustomResources(mbPair, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to delete custom resources for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete Services for generic components
	err = c.managed.DeleteServices(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to delete services for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete Deployments for generic components
	err = c.managed.DeleteDeployments(mbPair, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to delete deployments for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete ClusterRoleBindings
	err = c.managed.DeleteClusterRoleBindings(mbPair, c.managedClusterConnections, true)
	if err != nil {
		zap.S().Errorf("Failed to delete cluster role bindings for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete ClusterRoles
	err = c.managed.DeleteClusterRoles(mbPair, c.managedClusterConnections, true)
	if err != nil {
		zap.S().Errorf("Failed to delete cluster roles for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	err = c.monitoring.DeletePomPusher(binding.Name, &kubeDeployment{kubeClientSet: c.kubeClientSet})
	if err != nil {
		zap.S().Errorf("Failed to delete prometheus-pusher for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Delete Namespaces - this will also cleanup any Ingresses,
	// ServiceEntries, ServiceAccounts, ConfigMaps and Secrets within the namespace
	err = c.managed.DeleteNamespaces(mbPair, c.managedClusterConnections, true)
	if err != nil {
		zap.S().Errorf("Failed delete namespaces for binding %s: %v", binding.Name, err)
		return
	}

	// If this is the last model binding to be deleted then we need to cleanup additional resources that are shared
	// among model/binding pairs.
	if len(c.modelBindingPairs) == 1 {
		// Delete ClusterRoleBindings
		err = c.managed.DeleteClusterRoleBindings(mbPair, c.managedClusterConnections, false)
		if err != nil {
			zap.S().Errorf("Failed to delete cluster role bindings: %v", err)
			return
		}

		// Delete ClusterRoles
		err = c.managed.DeleteClusterRoles(mbPair, c.managedClusterConnections, false)
		if err != nil {
			zap.S().Errorf("Failed to delete cluster roles: %v", err)
			return
		}

		// Delete Namespaces - this will also cleanup any deployments, Ingresses,
		// ServiceEntries, ServiceAccounts, and Secrets within the namespace
		err = c.managed.DeleteNamespaces(mbPair, c.managedClusterConnections, false)
		if err != nil {
			zap.S().Errorf("Failed to delete a common namespace for all bindings: %v", err)
			return
		}

	}

	if _, ok := c.applicationBindings[binding.Name]; ok {
		delete(c.applicationBindings, binding.Name)
	}

	if mbPairExists {
		delete(c.modelBindingPairs, binding.Name)
	}

	if _, ok := c.bindingSyncThreshold[binding.Name]; ok {
		delete(c.bindingSyncThreshold, binding.Name)
	}

	if contains(binding.GetFinalizers(), bindingFinalizer) {
		c.removeFinalizer(binding)
	}

}

func (c *Controller) waitForManagedClusters(mbPair *types.ModelBindingPair, bindingName string) error {
	timeout := time.After(1 * time.Minute)
	tick := time.Tick(5 * time.Second)
	var err error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for Managed clusters referenced in binding %s, error %s", bindingName, err.Error())
		case <-tick:
			zap.S().Debugf("Waiting for all Managed Clusters referenced in binding %s to be available..", bindingName)
			_, err = c.util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
			if err == nil {
				zap.S().Debugf("All Managed Clusters referenced in binding %s are available..", bindingName)
				return nil
			}

			zap.S().Infow(err.Error())
		}
	}

}

// Check if string is found in list
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Remove string from list
func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// Add finalizer to binding for cleanup when binding is deleted
func (c *Controller) addFinalizer(binding *v1beta1v8o.VerrazzanoBinding) (*v1beta1v8o.VerrazzanoBinding, error) {
	zap.S().Infof("Adding finalizer %s to binding %s", bindingFinalizer, binding.Name)
	binding.SetFinalizers(append(binding.GetFinalizers(), bindingFinalizer))

	// Update binding
	binding, err := c.verrazzanoOperatorClientSet.VerrazzanoV1beta1().VerrazzanoBindings(binding.Namespace).Update(context.TODO(), binding, metav1.UpdateOptions{})
	if err != nil {
		zap.S().Errorf("Failed adding finalizer %s to binding %s, error %s", bindingFinalizer, binding.Name, err.Error())
		return nil, err
	}
	return binding, nil
}

// Remove finalizer from binding after binding is deleted
func (c *Controller) removeFinalizer(binding *v1beta1v8o.VerrazzanoBinding) error {
	zap.S().Infof("Removing finalizer %s from binding %s", bindingFinalizer, binding.Name)
	binding.SetFinalizers(remove(binding.GetFinalizers(), bindingFinalizer))

	// Update binding
	_, err := c.verrazzanoOperatorClientSet.VerrazzanoV1beta1().VerrazzanoBindings(binding.Namespace).Update(context.TODO(), binding, metav1.UpdateOptions{})
	if err != nil {
		zap.S().Errorf("Failed removing finalizer %s from binding %s, error %s", bindingFinalizer, binding.Name, err.Error())
		return err
	}
	return nil
}

// managedInterface defines the functions in the 'managed' package that are used  by the Controller
type managedInterface interface {
	BuildManagedClusterConnection(kubeConfigContents []byte, stopCh <-chan struct{}) (*v8outil.ManagedClusterConnection, error)
	CreateCrdDefinitions(managedClusterConnection *v8outil.ManagedClusterConnection, managedCluster *v1beta1v8o.VerrazzanoManagedCluster, manifest *v8outil.Manifest) error
	CreateNamespaces(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateSecrets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec v8omonitoring.Secrets) error
	CreateServiceAccounts(bindingName string, imagePullSecrets []corev1.LocalObjectReference, managedClusters map[string]*types.ManagedCluster, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateConfigMaps(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateClusterRoles(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateClusterRoleBindings(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateIngresses(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateServiceEntries(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateServices(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, manifest *v8outil.Manifest, verrazzanoURI string, sec v8omonitoring.Secrets) error
	CreateCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, stopCh <-chan struct{}, vbLister listers.VerrazzanoBindingLister) error
	UpdateIstioPrometheusConfigMaps(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateDaemonSets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, verrazzanoURI string) error
	CreateDestinationRules(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateAuthorizationPolicies(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, stopCh <-chan struct{}) error
	CleanupOrphanedServiceEntries(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedIngresses(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedConfigMaps(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error
	CleanupOrphanedServices(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	DeleteCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	DeleteClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error
	DeleteClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error
	DeleteNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error
	DeleteServices(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	DeleteDeployments(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
}

// managedPackage is the managedInterface implementation through which all 'managed' package functions are invoked
type managedPackage struct {
	managedInterface
}

// All managedPackage methods simply delegate to the corresponding function in the 'managed' package

func (*managedPackage) BuildManagedClusterConnection(kubeConfigContents []byte, stopCh <-chan struct{}) (*v8outil.ManagedClusterConnection, error) {
	return v8omanaged.BuildManagedClusterConnection(kubeConfigContents, stopCh)
}

func (*managedPackage) CreateCrdDefinitions(managedClusterConnection *v8outil.ManagedClusterConnection, managedCluster *v1beta1v8o.VerrazzanoManagedCluster, manifest *v8outil.Manifest) error {
	return v8omanaged.CreateCrdDefinitions(managedClusterConnection, managedCluster, manifest)
}

func (*managedPackage) CreateNamespaces(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateNamespaces(mbPair, filteredConnections)
}

func (*managedPackage) CreateSecrets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec v8omonitoring.Secrets) error {
	return v8omanaged.CreateSecrets(mbPair, availableManagedClusterConnections, kubeClientSet, sec)
}

func (*managedPackage) CreateServiceAccounts(bindingName string, imagePullSecrets []corev1.LocalObjectReference, managedClusters map[string]*types.ManagedCluster, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateServiceAccounts(bindingName, imagePullSecrets, managedClusters, filteredConnections)
}

func (*managedPackage) CreateConfigMaps(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateConfigMaps(mbPair, filteredConnections)
}

func (*managedPackage) CreateClusterRoles(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateClusterRoles(mbPair, filteredConnections)
}

func (*managedPackage) CreateClusterRoleBindings(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateClusterRoleBindings(mbPair, filteredConnections)
}

func (*managedPackage) CreateIngresses(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateIngresses(mbPair, filteredConnections)
}

func (*managedPackage) CreateServiceEntries(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateServiceEntries(mbPair, filteredConnections, availableManagedClusterConnections)
}

func (*managedPackage) CreateServices(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateServices(mbPair, filteredConnections)
}

func (*managedPackage) CreateDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, manifest *v8outil.Manifest, verrazzanoURI string, sec v8omonitoring.Secrets) error {
	return v8omanaged.CreateDeployments(mbPair, availableManagedClusterConnections, manifest, verrazzanoURI, sec)
}

func (*managedPackage) CreateCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, stopCh <-chan struct{}, vbLister listers.VerrazzanoBindingLister) error {
	return v8omanaged.CreateCustomResources(mbPair, availableManagedClusterConnections, stopCh, vbLister)
}

func (*managedPackage) UpdateIstioPrometheusConfigMaps(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.UpdateIstioPrometheusConfigMaps(mbPair, secretLister, availableManagedClusterConnections)
}

func (*managedPackage) CreateDaemonSets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, verrazzanoURI string) error {
	return v8omanaged.CreateDaemonSets(mbPair, availableManagedClusterConnections, verrazzanoURI)
}

func (*managedPackage) CreateDestinationRules(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateDestinationRules(mbPair, filteredConnections)
}

func (*managedPackage) CreateAuthorizationPolicies(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateAuthorizationPolicies(mbPair, filteredConnections)
}

func (*managedPackage) CleanupOrphanedCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, stopCh <-chan struct{}) error {
	return v8omanaged.CleanupOrphanedCustomResources(mbPair, availableManagedClusterConnections, stopCh)
}

func (*managedPackage) CleanupOrphanedServiceEntries(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedServiceEntries(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedIngresses(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedIngresses(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedClusterRoleBindings(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedClusterRoles(mbPair, availableManagedClusterConnections)
}
func (*managedPackage) CleanupOrphanedConfigMaps(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedConfigMaps(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error {
	return v8omanaged.CleanupOrphanedNamespaces(mbPair, availableManagedClusterConnections, allMbPairs)
}

func (*managedPackage) CleanupOrphanedServices(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedServices(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedDeployments(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) DeleteCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.DeleteCustomResources(mbPair, availableManagedClusterConnections)
}

func (*managedPackage) DeleteClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error {
	return v8omanaged.DeleteClusterRoleBindings(mbPair, availableManagedClusterConnections, bindingLabel)
}

func (*managedPackage) DeleteClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error {
	return v8omanaged.DeleteClusterRoles(mbPair, availableManagedClusterConnections, bindingLabel)
}

func (*managedPackage) DeleteNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error {
	return v8omanaged.DeleteNamespaces(mbPair, availableManagedClusterConnections, bindingLabel)
}

func (*managedPackage) DeleteServices(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.DeleteServices(mbPair, filteredConnections)
}

func (*managedPackage) DeleteDeployments(mbPair *types.ModelBindingPair, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.DeleteDeployments(mbPair, filteredConnections)
}

// cacheInterface defines the functions in the 'cache' package that are used  by the Controller
type cacheInterface interface {
	WaitForCacheSync(stopCh <-chan struct{}, cacheSyncs ...k8scache.InformerSynced) bool
}

// cachePackage is the cacheInterface implementation through which all 'cache' package functions are invoked
type cachePackage struct {
	cacheInterface
}

// All cachePackage methods simply delegate to the corresponding function in the 'cache' package

func (c *cachePackage) WaitForCacheSync(stopCh <-chan struct{}, cacheSyncs ...k8scache.InformerSynced) bool {
	return k8scache.WaitForCacheSync(stopCh, cacheSyncs...)
}

// utilInterface defines the functions in the 'util' package that are used  by the Controller
type utilInterface interface {
	GetManagedClustersForVerrazzanoBinding(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) (map[string]*v8outil.ManagedClusterConnection, error)
}

// utilPackage is the utilInterface implementation through which all 'util' package functions are invoked
type utilPackage struct {
	utilInterface
}

// All utilPackage methods simply delegate to the corresponding function in the 'util' package

func (u *utilPackage) GetManagedClustersForVerrazzanoBinding(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) (map[string]*v8outil.ManagedClusterConnection, error) {
	return v8outil.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
}

// localInterface defines the functions in the 'local' package that are used  by the Controller
type localInterface interface {
	DeleteVmi(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error
	DeleteSecrets(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error
	DeleteConfigMaps(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error
	CreateUpdateVmi(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error
	UpdateConfigMaps(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error
	UpdateAcmeDNSSecret(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error
}

// localPackage is the localInterface implementation through which all 'local' package functions are invoked
type localPackage struct {
	localInterface
}

// All localPackage methods simply delegate to the corresponding function in the 'local' package

func (l *localPackage) DeleteVmi(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {
	return v8olocal.DeleteVmi(binding, vmoClientSet, vmiLister)
}

func (l *localPackage) DeleteSecrets(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error {
	return v8olocal.DeleteSecrets(binding, kubeClientSet, secretLister)
}

func (l *localPackage) DeleteConfigMaps(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	return v8olocal.DeleteConfigMaps(binding, kubeClientSet, configMapLister)
}

func (l *localPackage) CreateUpdateVmi(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error {
	return v8olocal.CreateUpdateVmi(binding, vmoClientSet, vmiLister, verrazzanoURI, enableMonitoringStorage)
}

func (l *localPackage) UpdateConfigMaps(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	return v8olocal.UpdateConfigMaps(binding, kubeClientSet, configMapLister)
}

func (l *localPackage) UpdateAcmeDNSSecret(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error {
	return v8olocal.UpdateAcmeDNSSecret(binding, kubeClientSet, secretLister, name, verrazzanoURI)
}

// monitoringInterface defines the functions in the 'monitoring' package that are used  by the Controller
type monitoringInterface interface {
	CreateVmiSecrets(binding *v1beta1v8o.VerrazzanoBinding, secrets v8omonitoring.Secrets) error
	DeletePomPusher(binding string, helper v8outil.DeploymentHelper) error
}

// monitoringPackage is the monitoringInterface implementation through which all 'monitoring' package functions are invoked
type monitoringPackage struct {
	monitoringInterface
}

// All monitoringPackage methods simply delegate to the corresponding function in the 'monitoring' package

func (m *monitoringPackage) CreateVmiSecrets(binding *v1beta1v8o.VerrazzanoBinding, secrets v8omonitoring.Secrets) error {
	return v8omonitoring.CreateVmiSecrets(binding, secrets)
}

func (m *monitoringPackage) DeletePomPusher(binding string, helper v8outil.DeploymentHelper) error {
	return v8omonitoring.DeletePomPusher(binding, helper)
}
