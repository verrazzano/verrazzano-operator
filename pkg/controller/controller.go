// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
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
	vmoclientset "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	vmoinformers "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/informers/externalversions"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	v8olocal "github.com/verrazzano/verrazzano-operator/pkg/local"
	v8omanaged "github.com/verrazzano/verrazzano-operator/pkg/managed"
	"github.com/verrazzano/verrazzano-operator/pkg/signals"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	v8outil "github.com/verrazzano/verrazzano-operator/pkg/util"
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
	secretLister      corev1listers.SecretLister
	secretInformer    k8scache.SharedIndexInformer
	configMapLister   corev1listers.ConfigMapLister
	configMapInformer k8scache.SharedIndexInformer
	vmiLister         vmolisters.VerrazzanoMonitoringInstanceLister
	vmiInformer       k8scache.SharedIndexInformer
	secrets           v8omonitoring.Secrets
	// The current set of known managed clusters
	managedClusterConnections map[string]*v8outil.ManagedClusterConnection

	// The current set of known models
	applicationModels map[string]*types.SyntheticModel

	// The current set of known bindings
	applicationBindings map[string]*types.SyntheticBinding

	// The current set of known model/binding pairs
	SyntheticModelBindings map[string]*types.SyntheticModelBinding

	// Misc
	watchNamespace string
	stopCh         <-chan struct{}

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// Keep track of no. of times add/update events are encountered for unchanged bindings
	bindingSyncThreshold map[string]int
}

// Listers represents listers used by the controller.
type Listers struct {
	SyntheticModelBindings *map[string]*types.SyntheticModelBinding
	KubeClientSet          *kubernetes.Interface
}

// ListerSet returns a list of listers used by the controller.
func (c *Controller) ListerSet() Listers {
	return Listers{
		SyntheticModelBindings: &c.SyntheticModelBindings,
		KubeClientSet:          &c.kubeClientSet,
	}
}

// NewController returns a new Verrazzano Operator controller
func NewController(config *rest.Config, watchNamespace string, verrazzanoURI string, enableMonitoringStorage string) (*Controller, error) {
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
	if watchNamespace == "" {
		// Consider all namespaces if our namespace is left wide open our set to default
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeClientSet, constants.ResyncPeriod)
		verrazzanoOperatorInformerFactory = informers.NewSharedInformerFactory(verrazzanoOperatorClientSet, constants.ResyncPeriod)
		vmoInformerFactory = vmoinformers.NewSharedInformerFactory(vmoClientSet, constants.ResyncPeriod)
	} else {
		// Otherwise, restrict to a specific namespace
		kubeInformerFactory = kubeinformers.NewFilteredSharedInformerFactory(kubeClientSet, constants.ResyncPeriod, watchNamespace, nil)
		verrazzanoOperatorInformerFactory = informers.NewFilteredSharedInformerFactory(verrazzanoOperatorClientSet, constants.ResyncPeriod, watchNamespace, nil)
		vmoInformerFactory = vmoinformers.NewFilteredSharedInformerFactory(vmoClientSet, constants.ResyncPeriod, watchNamespace, nil)
	}
	secretsInformer := kubeInformerFactory.Core().V1().Secrets()
	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
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
		managed:                     newManagedPackage(),
		cache:                       newCachePackage(),
		util:                        newUtilPackage(),
		local:                       newLocalPackage(),
		monitoring:                  newMonitoringPackage(),
		watchNamespace:              watchNamespace,
		verrazzanoURI:               verrazzanoURI,
		enableMonitoringStorage:     enableMonitoringStorage,
		kubeClientSet:               kubeClientSet,
		verrazzanoOperatorClientSet: verrazzanoOperatorClientSet,
		vmoClientSet:                vmoClientSet,
		kubeExtClientSet:            kubeExtClientSet,
		secretLister:                secretsInformer.Lister(),
		secretInformer:              secretsInformer.Informer(),
		configMapLister:             configMapInformer.Lister(),
		configMapInformer:           configMapInformer.Informer(),
		vmiInformer:                 vmiInformer.Informer(),
		vmiLister:                   vmiInformer.Lister(),
		recorder:                    recorder,
		managedClusterConnections:   map[string]*v8outil.ManagedClusterConnection{},
		applicationModels:           map[string]*types.SyntheticModel{},
		applicationBindings:         map[string]*types.SyntheticBinding{},
		SyntheticModelBindings:      map[string]*types.SyntheticModelBinding{},
		bindingSyncThreshold:        map[string]int{},
		secrets:                     kubeSecrets,
	}

	// Set up signals so we handle the first shutdown signal gracefully
	zap.S().Debugw("Setting up signals")
	stopCh := setupSignalHandler()

	go kubeInformerFactory.Start(stopCh)
	go verrazzanoOperatorInformerFactory.Start(stopCh)
	go vmoInformerFactory.Start(stopCh)

	// Wait for the caches to be synced before starting watchers
	zap.S().Infow("Waiting for informer caches to sync")
	if ok := controller.cache.WaitForCacheSync(controller.stopCh, controller.secretInformer.HasSynced,
		controller.vmiInformer.HasSynced); !ok {
		return controller, errors.New("failed to wait for caches to sync")
	}

	// Install Global Entities
	executeCreateUpdateGlobalEntities(&types.SyntheticBinding{
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
func executeCreateUpdateGlobalEntitiesGoroutine(binding *types.SyntheticBinding, c *Controller) {
	go createUpdateGlobalEntitiesGoroutine(binding, c)
}

// CreateUpdateGlobalEntitiesGoroutine installs global entities and loops forever so it must be called in a goroutine
func createUpdateGlobalEntitiesGoroutine(binding *types.SyntheticBinding, c *Controller) {
	zap.S().Infow("Configuring System VMI...")
	for {
		createUpdateGlobalEntities(binding, c)
		<-time.After(60 * time.Second)
	}
}

func createUpdateGlobalEntities(binding *types.SyntheticBinding, c *Controller) {
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
func (c *Controller) Run(threadiness int, kubeconfigPath string) error {
	defer runtime.HandleCrash()

	// Start the informer factories to begin populating the informer caches
	zap.S().Infow("Starting Verrazzano Operator controller")

	// create a single in-memory managed cluster for the local cluster
	c.processManagedCluster("local", kubeconfigPath)

	<-c.stopCh
	return nil
}

// Process a change to a VerrazzanoManagedCluster
// NOTE: This used to be called when a VMC was created or updated, but the old VMC resources are deprecated so we will call this
// one time at startup to create an in-memory managed cluster for the local cluster. The kubeconfigPath parameter is optional
// and should generally only have a path when running outside of a Kubernetes cluster (e.g. when running the operator locally).
func (c *Controller) processManagedCluster(managedClusterName string, kubeconfigPath string) {
	// Obtain the optional list of imagePullSecrets from the verrazzano-operator service account
	sa, err := c.kubeClientSet.CoreV1().ServiceAccounts(constants.VerrazzanoSystem).Get(context.TODO(), constants.VerrazzanoOperatorServiceAccount, metav1.GetOptions{})
	if err != nil {
		zap.S().Errorf("Can't find ServiceAccount %s in namespace %s", constants.VerrazzanoOperatorServiceAccount, constants.VerrazzanoSystem)
		return
	}
	c.imagePullSecrets = sa.ImagePullSecrets

	// A synthetic binding will be constructed and passed to the managed clusters
	systemBinding := &types.SyntheticBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	// A synthetic model will be constructed and passed to the managed clusters
	systemModel := &types.SyntheticModel{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	// A synthetic SynModel binding pair will be constructed and passed to the managed clusters
	vzSynMB := CreateSyntheticModelBinding(systemModel, systemBinding, c.verrazzanoURI, c.imagePullSecrets)

	// Construct a new client to the managed cluster when it's added
	_, clusterExists := c.managedClusterConnections[managedClusterName]
	if !clusterExists {
		zap.S().Infof("(Re)creating k8s clients for Managed SynModel %s", managedClusterName)

		managedClusterConnection, err := c.managed.BuildManagedClusterConnection(kubeconfigPath, c.stopCh)
		if err != nil {
			zap.S().Error(err)
			return
		}

		// Sync the cache so that resources (like secrets) are in the cache
		zap.S().Infow("Waiting for informer caches to sync")
		if ok := c.cache.WaitForCacheSync(c.stopCh, managedClusterConnection.DeploymentInformer.HasSynced,
			managedClusterConnection.NamespaceInformer.HasSynced, managedClusterConnection.SecretInformer.HasSynced); !ok {
			zap.S().Error(errors.New("failed to wait for caches to sync"))
		}

		c.managedClusterConnections[managedClusterName] = managedClusterConnection

		mc := &types.ManagedCluster{
			Name:        managedClusterName,
			Secrets:     map[string][]string{},
			Ingresses:   map[string][]*types.Ingress{},
			RemoteRests: map[string][]*types.RemoteRestConnection{},
		}
		vzSynMB.ManagedClusters[managedClusterName] = mc

		// Add in the monitoring and logging namespace if not already added
		mc.Namespaces = append(mc.Namespaces, constants.MonitoringNamespace, constants.LoggingNamespace)

		/*********************
		 * Create Artifacts in the Managed SynModel
		 **********************/

		// Create all the components needed by logging and monitoring in managed clusters to push metrics and logs into System VMI in management cluster
		c.createManagedClusterResourcesForBinding(vzSynMB, c.getClusterInfo())
	}
}

// update the Istio destination rules and authorization policies for the given model/binding whenever a pod is added or deleted
func (c *Controller) updateIstioPolicies(vzSynMB *types.SyntheticModelBinding, pod *corev1.Pod, action string) {
}

// Create managed resources on clusters depending upon the binding
func (c *Controller) createManagedClusterResourcesForBinding(vzSynMB *types.SyntheticModelBinding, clusterInfo v8omonitoring.ClusterInfo) {
	filteredConnections, err := c.util.GetManagedClustersForVerrazzanoBinding(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to get filtered connections for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create Namespaces
	err = c.managed.CreateNamespaces(vzSynMB, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create namespaces for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create (copy) the Secrets needed from the management cluster to the managed clusters
	err = c.managed.CreateSecrets(vzSynMB, c.managedClusterConnections, c.kubeClientSet, c.secrets, clusterInfo)
	if err != nil {
		zap.S().Errorf("Failed to create secrets for binding %s: %v", vzSynMB.SynBinding.Namespace, err)
	}

	// Create ServiceAccounts
	err = c.managed.CreateServiceAccounts(vzSynMB.SynBinding.Name, vzSynMB.ImagePullSecrets, vzSynMB.ManagedClusters, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create service accounts for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create ConfigMaps
	err = c.managed.CreateConfigMaps(vzSynMB, filteredConnections, clusterInfo)
	if err != nil {
		zap.S().Errorf("Failed to create config maps for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create ClusterRoles
	err = c.managed.CreateClusterRoles(vzSynMB, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create cluster roles for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create ClusterRoleBindings
	err = c.managed.CreateClusterRoleBindings(vzSynMB, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create cluster role bindings for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create Deployments
	err = c.managed.CreateDeployments(vzSynMB, filteredConnections, c.verrazzanoURI, c.secrets)
	if err != nil {
		zap.S().Errorf("Failed to create deployments for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create Services
	err = c.managed.CreateServices(vzSynMB, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to create service for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Create DaemonSets
	err = c.managed.CreateDaemonSets(vzSynMB, filteredConnections, c.verrazzanoURI, clusterInfo)
	if err != nil {
		zap.S().Errorf("Failed to create DaemonSets for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}
}

// Process a change to a VerrazzanoModel
func (c *Controller) processApplicationModelAdded(verrazzanoModel interface{}) {
	model := verrazzanoModel.(*types.SyntheticModel)

	if existingModel, ok := c.applicationModels[model.Name]; ok {
		if existingModel.GetResourceVersion() == model.GetResourceVersion() {
			zap.S().Debugf("No changes to the model %s", model.Name)
			return
		}

		zap.S().Infof("Updating the model %s", model.Name)
	} else {
		zap.S().Infof("Adding the model %s", model.Name)
	}

	// Add or replace the model object
	c.applicationModels[model.Name] = model

	// During restart of the operator a binding can be added before a model so make sure we create
	// model/binding pair now.
	for _, binding := range c.applicationBindings {
		if _, ok := c.SyntheticModelBindings[binding.Name]; !ok {
			if model.Name == binding.Spec.ModelName {
				zap.S().Infof("Adding model/binding pair during add model for model %s and binding %s", binding.Spec.ModelName, binding.Name)
				vzSynMB := CreateSyntheticModelBinding(model, binding, c.verrazzanoURI, c.imagePullSecrets)
				c.SyntheticModelBindings[binding.Name] = vzSynMB
				break
			}
		}
	}
}

// Process a removal of a VerrazzanoModel
func (c *Controller) processApplicationModelDeleted(verrazzanoModel interface{}) {
	model := verrazzanoModel.(*types.SyntheticModel)

	if _, ok := c.applicationModels[model.Name]; ok {
		zap.S().Infof("Deleting the model %s", model.Name)
		delete(c.applicationModels, model.Name)
	}
}

func getSyntheticModelBinding(c *Controller, binding *types.SyntheticBinding) (*types.SyntheticModelBinding, bool) {
	vzSynMB, vzSynMBExists := c.SyntheticModelBindings[binding.Name]
	return vzSynMB, vzSynMBExists
}

// Process a change to a VerrazzanoBinding/Sync existing VerrazzanoBinding with cluster state
func (c *Controller) processApplicationBindingAdded(verrazzanoBinding interface{}) {
	binding := verrazzanoBinding.(*types.SyntheticBinding)

	if binding.GetDeletionTimestamp() != nil {
		if contains(binding.GetFinalizers(), bindingFinalizer) {
			// Add binding in the case where delete has been initiated before the binding is added.
			if _, ok := c.applicationBindings[binding.Name]; !ok {
				zap.S().Infof("Adding the binding %s", binding.Name)
				c.applicationBindings[binding.Name] = binding
			}
			_, vzSynMBExists := getSyntheticModelBinding(c, binding)
			if !vzSynMBExists {
				// During restart of the operator a delete can happen before a model/binding is created so create one now.
				if model, ok := c.applicationModels[binding.Spec.ModelName]; ok {
					zap.S().Infof("Adding model/binding pair during add binding for model %s and binding %s", binding.Spec.ModelName, binding.Name)
					vzSynMB := CreateSyntheticModelBinding(model, binding, c.verrazzanoURI, c.imagePullSecrets)
					c.SyntheticModelBindings[binding.Name] = vzSynMB
				}
			}
		}

		zap.S().Debugf("SynBinding %s is marked for deletion/already deleted", binding.Name)
		if contains(binding.GetFinalizers(), bindingFinalizer) {
			c.processApplicationBindingDeleted(verrazzanoBinding)
		}
		return
	}

	var err error
	if existingBinding, ok := c.applicationBindings[binding.Name]; ok {
		if existingBinding.GetResourceVersion() == binding.GetResourceVersion() {
			counter, ok := c.bindingSyncThreshold[binding.Name]
			if !ok {
				c.bindingSyncThreshold[binding.Name] = 0
			} else {
				c.bindingSyncThreshold[binding.Name] = counter + 1
			}

			if counter < 4 {
				zap.S().Debugf("No changes to binding %s. Skip sync as sync threshold has not passed", binding.Name)
				return
			}

			zap.S().Debugf("No changes to binding %s. Syncing with cluster state", binding.Name)
			c.bindingSyncThreshold[binding.Name] = 0
		} else {
			zap.S().Infof("Updating the binding %s", binding.Name)
			c.applicationBindings[binding.Name] = binding
		}

	} else {
		zap.S().Infof("Adding the binding %s", binding.Name)
		c.applicationBindings[binding.Name] = binding
	}

	// Does a model/binding pair already exist?
	vzSynMB, vzSynMBExists := getSyntheticModelBinding(c, binding)
	if !vzSynMBExists {
		// If a model exists for this binding, then create the model/binding pair
		if model, ok := c.applicationModels[binding.Spec.ModelName]; ok {
			zap.S().Infof("Adding new model/binding pair during add binding for model %s and binding %s", binding.Spec.ModelName, binding.Name)
			vzSynMB = CreateSyntheticModelBinding(model, binding, c.verrazzanoURI, c.imagePullSecrets)
			c.SyntheticModelBindings[binding.Name] = vzSynMB
			vzSynMBExists = true
		}
	} else {
		// Rebuild the existing model/binding pair
		if existingModel, ok := c.applicationModels[binding.Spec.ModelName]; ok {
			UpdateSyntheticModelBinding(vzSynMB, existingModel, binding, c.verrazzanoURI, c.imagePullSecrets)
		}

	}

	// Nothing further to do if the model/binding pair does not exist yet
	if !vzSynMBExists {
		return
	}

	err = c.waitForManagedClusters(vzSynMB, binding.Name)
	if err != nil {
		zap.S().Errorf("Skipping processing of VerrazzanoBinding %s until all referenced Managed Clusters are available, error %s", binding.Name, err.Error())
		return
	}

	/*********************
	 * Create Artifacts in the Local SynModel
	 **********************/
	// Create VMIs
	err = c.local.CreateUpdateVmi(binding, c.vmoClientSet, c.vmiLister, c.verrazzanoURI, c.enableMonitoringStorage)
	if err != nil {
		zap.S().Errorf("Failed to create VMIs for binding %s: %v", binding.Name, err)
	}

	// Create secrets
	err = c.monitoring.CreateVmiSecrets(binding, c.secrets)
	if err != nil {
		zap.S().Errorf("Failed to create secrets for binding %s: %v", binding.Name, err)
	}

	// Update ConfigMaps
	err = c.local.UpdateConfigMaps(binding, c.kubeClientSet, c.configMapLister)
	if err != nil {
		zap.S().Errorf("Failed to update ConfigMaps for binding %s: %v", binding.Name, err)
	}

	// Update the "bring your own DNS" credentials?
	err = c.local.UpdateAcmeDNSSecret(binding, c.kubeClientSet, c.secretLister, constants.AcmeDNSSecret, c.verrazzanoURI)
	if err != nil {
		zap.S().Errorf("Failed to update DNS credentials for binding %s: %v", binding.Name, err)
	}

	/*********************
	 * Create Artifacts in the Managed SynModel
	 **********************/
	c.createManagedClusterResourcesForBinding(vzSynMB, c.getClusterInfo())

	// Cleanup any resources no longer needed
	c.cleanupOrphanedResources(vzSynMB)
}

// Check for a duplicate namespaces being used across two differrent bindings
func checkForDuplicateNamespaces(placement *types.ClusterPlacement, binding *types.SyntheticBinding, currPlacement *types.ClusterPlacement, currBinding *types.SyntheticBinding) bool {
	dupFound := false
	for _, namespace := range placement.Namespaces {
		for _, currNamespace := range currPlacement.Namespaces {
			if currNamespace.Name == namespace.Name {
				if !dupFound {
					zap.S().Errorf("SynBinding %s has a conflicting namespace with binding %s.  Namespaces must be unique across bindings.", binding.Name, currBinding.Name)
				}
				zap.S().Errorf("Duplicate namespace %s found in placement %s", namespace.Name, placement.Name)
				dupFound = true
				break
			}
		}
	}

	return dupFound
}

func (c *Controller) cleanupOrphanedResources(vzSynMB *types.SyntheticModelBinding) {
	// Cleanup Custom Resources
	err := c.managed.CleanupOrphanedCustomResources(vzSynMB, c.managedClusterConnections, c.stopCh)
	if err != nil {
		zap.S().Errorf("Failed to cleanup custom resources for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup Services for generic components
	err = c.managed.CleanupOrphanedServices(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup services for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup Deployments for generic components
	err = c.managed.CleanupOrphanedDeployments(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup deployments for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup ServiceEntries
	err = c.managed.CleanupOrphanedServiceEntries(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ServiceEntries for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup Ingresses
	err = c.managed.CleanupOrphanedIngresses(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup Ingresses for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup ClusterRoleBindings
	err = c.managed.CleanupOrphanedClusterRoleBindings(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ClusterRoleBindings for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup ClusterRoles
	err = c.managed.CleanupOrphanedClusterRoles(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ClusterRoles for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup ConfigMaps
	err = c.managed.CleanupOrphanedConfigMaps(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to cleanup ConfigMaps for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Cleanup Namespaces - this will also cleanup any ServiceAccounts and Secrets within the namespace
	err = c.managed.CleanupOrphanedNamespaces(vzSynMB, c.managedClusterConnections, c.SyntheticModelBindings)
	if err != nil {
		zap.S().Errorf("Failed to cleanup Namespaces for binding %s: %v", vzSynMB.SynBinding.Name, err)
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
	binding := verrazzanoBinding.(*types.SyntheticBinding)

	if !contains(binding.GetFinalizers(), bindingFinalizer) {
		zap.S().Infof("Resources for binding %s already deleted", binding.Name)
		return
	}

	vzSynMB, vzSynMBExists := getSyntheticModelBinding(c, binding)
	if !vzSynMBExists {
		return
	}

	zap.S().Infof("Deleting resources for binding %s", binding.Name)

	/*********************
	 * Delete Artifacts in the Local SynModel
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
	 * Delete Artifacts in the Managed SynModel
	 **********************/

	filteredConnections, err := c.util.GetManagedClustersForVerrazzanoBinding(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to get filtered connections for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Delete Custom Resources
	err = c.managed.DeleteCustomResources(vzSynMB, c.managedClusterConnections)
	if err != nil {
		zap.S().Errorf("Failed to delete custom resources for binding %s: %v", vzSynMB.SynBinding.Name, err)
		return
	}

	// Delete Services for generic components
	err = c.managed.DeleteServices(vzSynMB, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to delete services for binding %s: %v", vzSynMB.SynBinding.Name, err)
		return
	}

	// Delete Deployments for generic components
	err = c.managed.DeleteDeployments(vzSynMB, filteredConnections)
	if err != nil {
		zap.S().Errorf("Failed to delete deployments for binding %s: %v", vzSynMB.SynBinding.Name, err)
		return
	}

	// Delete ClusterRoleBindings
	err = c.managed.DeleteClusterRoleBindings(vzSynMB, c.managedClusterConnections, true)
	if err != nil {
		zap.S().Errorf("Failed to delete cluster role bindings for binding %s: %v", vzSynMB.SynBinding.Name, err)
		return
	}

	// Delete ClusterRoles
	err = c.managed.DeleteClusterRoles(vzSynMB, c.managedClusterConnections, true)
	if err != nil {
		zap.S().Errorf("Failed to delete cluster roles for binding %s: %v", vzSynMB.SynBinding.Name, err)
		return
	}

	err = c.monitoring.DeletePomPusher(binding.Name, &kubeDeployment{kubeClientSet: c.kubeClientSet})
	if err != nil {
		zap.S().Errorf("Failed to delete prometheus-pusher for binding %s: %v", vzSynMB.SynBinding.Name, err)
	}

	// Delete Namespaces - this will also cleanup any Ingresses,
	// ServiceEntries, ServiceAccounts, ConfigMaps and Secrets within the namespace
	err = c.managed.DeleteNamespaces(vzSynMB, c.managedClusterConnections, true)
	if err != nil {
		zap.S().Errorf("Failed delete namespaces for binding %s: %v", binding.Name, err)
		return
	}

	// If this is the last model binding to be deleted then we need to cleanup additional resources that are shared
	// among model/binding pairs.
	if len(c.SyntheticModelBindings) == 1 {
		// Delete ClusterRoleBindings
		err = c.managed.DeleteClusterRoleBindings(vzSynMB, c.managedClusterConnections, false)
		if err != nil {
			zap.S().Errorf("Failed to delete cluster role bindings: %v", err)
			return
		}

		// Delete ClusterRoles
		err = c.managed.DeleteClusterRoles(vzSynMB, c.managedClusterConnections, false)
		if err != nil {
			zap.S().Errorf("Failed to delete cluster roles: %v", err)
			return
		}

		// Delete Namespaces - this will also cleanup any deployments, Ingresses,
		// ServiceEntries, ServiceAccounts, and Secrets within the namespace
		err = c.managed.DeleteNamespaces(vzSynMB, c.managedClusterConnections, false)
		if err != nil {
			zap.S().Errorf("Failed to delete a common namespace for all bindings: %v", err)
			return
		}

	}

	if _, ok := c.applicationBindings[binding.Name]; ok {
		delete(c.applicationBindings, binding.Name)
	}

	if vzSynMBExists {
		delete(c.SyntheticModelBindings, binding.Name)
	}

	if _, ok := c.bindingSyncThreshold[binding.Name]; ok {
		delete(c.bindingSyncThreshold, binding.Name)
	}
}

func (c *Controller) waitForManagedClusters(vzSynMB *types.SyntheticModelBinding, bindingName string) error {
	timeout := time.After(1 * time.Minute)
	tick := time.Tick(5 * time.Second)
	var err error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for Managed clusters referenced in binding %s, error %s", bindingName, err.Error())
		case <-tick:
			zap.S().Debugf("Waiting for all Managed Clusters referenced in binding %s to be available..", bindingName)
			_, err = c.util.GetManagedClustersForVerrazzanoBinding(vzSynMB, c.managedClusterConnections)
			if err == nil {
				zap.S().Debugf("All Managed Clusters referenced in binding %s are available..", bindingName)
				return nil
			}

			zap.S().Infow(err.Error())
		}
	}

}

// getContainerRuntime obtains the container runtime implementation used in the cluster.
// The returned value will be a string in the form of 'TYPE://VERSION'. If unable to obtain the
// container runtime then "unknown' is returned.
// Docker example: 'docker://19.3.11'
// Containerd example: 'containerd://1.4.0'
func (c *Controller) getContainerRuntime() string {
	nodes, err := c.kubeClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if nodes.Size() == 0 {
		zap.S().Warnf("Unable to determine container runtime because no nodes were returned.")
		return "unknown"
	}

	if err != nil {
		zap.S().Warnf("Unable to determine container runtime due to an error getting nodes: %v", err)
		return "unknown"
	}

	containerRuntime := nodes.Items[0].Status.NodeInfo.ContainerRuntimeVersion
	zap.S().Infof("Container runtime determined to be: %s", containerRuntime)

	return containerRuntime
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

// managedInterface defines the functions in the 'managed' package that are used  by the Controller
type managedInterface interface {
	BuildManagedClusterConnection(kubeconfigPath string, stopCh <-chan struct{}) (*v8outil.ManagedClusterConnection, error)
	CreateCrdDefinitions(managedClusterConnection *v8outil.ManagedClusterConnection, managedCluster *v1beta1v8o.VerrazzanoManagedCluster) error
	CreateNamespaces(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateSecrets(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec v8omonitoring.Secrets, clusterInfo v8omonitoring.ClusterInfo) error
	CreateServiceAccounts(bindingName string, imagePullSecrets []corev1.LocalObjectReference, managedClusters map[string]*types.ManagedCluster, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateConfigMaps(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection, clusterInfo v8omonitoring.ClusterInfo) error
	CreateClusterRoles(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateIngresses(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateServiceEntries(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, verrazzanoURI string, sec v8omonitoring.Secrets) error
	CreateCustomResources(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, stopCh <-chan struct{}) error
	UpdateIstioPrometheusConfigMaps(vzSynMB *types.SyntheticModelBinding, secretLister corev1listers.SecretLister, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateDaemonSets(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, verrazzanoURI string, clusterInfo v8omonitoring.ClusterInfo) error
	CreateDestinationRules(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CreateAuthorizationPolicies(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedCustomResources(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, stopCh <-chan struct{}) error
	CleanupOrphanedServiceEntries(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedIngresses(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedClusterRoles(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedConfigMaps(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, allvzSynMBs map[string]*types.SyntheticModelBinding) error
	CleanupOrphanedServices(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	CleanupOrphanedDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	DeleteCustomResources(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error
	DeleteClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error
	DeleteClusterRoles(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error
	DeleteNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error
	DeleteServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
	DeleteDeployments(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error
}

// managedPackage is the managedInterface implementation through which all 'managed' package functions are invoked
type managedPackage struct {
	managedInterface
}

// All managedPackage methods simply delegate to the corresponding function in the 'managed' package

func (*managedPackage) BuildManagedClusterConnection(kubeconfigPath string, stopCh <-chan struct{}) (*v8outil.ManagedClusterConnection, error) {
	return v8omanaged.BuildManagedClusterConnection(kubeconfigPath, stopCh)
}

func (*managedPackage) CreateNamespaces(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateNamespaces(vzSynMB, filteredConnections)
}

func (*managedPackage) CreateSecrets(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec v8omonitoring.Secrets, clusterInfo v8omonitoring.ClusterInfo) error {
	return v8omanaged.CreateSecrets(vzSynMB, availableManagedClusterConnections, kubeClientSet, sec, clusterInfo)
}

func (*managedPackage) CreateServiceAccounts(bindingName string, imagePullSecrets []corev1.LocalObjectReference, managedClusters map[string]*types.ManagedCluster, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateServiceAccounts(bindingName, imagePullSecrets, managedClusters, filteredConnections)
}

func (*managedPackage) CreateConfigMaps(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection, clusterInfo v8omonitoring.ClusterInfo) error {
	return v8omanaged.CreateConfigMaps(vzSynMB, filteredConnections, clusterInfo)
}

func (*managedPackage) CreateClusterRoles(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateClusterRoles(vzSynMB, filteredConnections)
}

func (*managedPackage) CreateClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateClusterRoleBindings(vzSynMB, filteredConnections)
}

func (*managedPackage) CreateServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CreateServices(vzSynMB, filteredConnections)
}

func (*managedPackage) CreateDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, verrazzanoURI string, sec v8omonitoring.Secrets) error {
	return v8omanaged.CreateDeployments(vzSynMB, availableManagedClusterConnections, verrazzanoURI, sec)
}

func (*managedPackage) CreateDaemonSets(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, verrazzanoURI string, clusterInfo v8omonitoring.ClusterInfo) error {
	return v8omanaged.CreateDaemonSets(vzSynMB, availableManagedClusterConnections, verrazzanoURI, clusterInfo)
}

func (*managedPackage) CleanupOrphanedClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedClusterRoleBindings(vzSynMB, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedClusterRoles(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedClusterRoles(vzSynMB, availableManagedClusterConnections)
}
func (*managedPackage) CleanupOrphanedConfigMaps(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedConfigMaps(vzSynMB, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, allvzSynMBs map[string]*types.SyntheticModelBinding) error {
	return v8omanaged.CleanupOrphanedNamespaces(vzSynMB, availableManagedClusterConnections, allvzSynMBs)
}

func (*managedPackage) CleanupOrphanedServices(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedServices(vzSynMB, availableManagedClusterConnections)
}

func (*managedPackage) CleanupOrphanedDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.CleanupOrphanedDeployments(vzSynMB, availableManagedClusterConnections)
}

func (*managedPackage) DeleteClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error {
	return v8omanaged.DeleteClusterRoleBindings(vzSynMB, availableManagedClusterConnections, bindingLabel)
}

func (*managedPackage) DeleteClusterRoles(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error {
	return v8omanaged.DeleteClusterRoles(vzSynMB, availableManagedClusterConnections, bindingLabel)
}

func (*managedPackage) DeleteNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection, bindingLabel bool) error {
	return v8omanaged.DeleteNamespaces(vzSynMB, availableManagedClusterConnections, bindingLabel)
}

func (*managedPackage) DeleteServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.DeleteServices(vzSynMB, filteredConnections)
}

func (*managedPackage) DeleteDeployments(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*v8outil.ManagedClusterConnection) error {
	return v8omanaged.DeleteDeployments(vzSynMB, filteredConnections)
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
	GetManagedClustersForVerrazzanoBinding(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) (map[string]*v8outil.ManagedClusterConnection, error)
}

// utilPackage is the utilInterface implementation through which all 'util' package functions are invoked
type utilPackage struct {
	utilInterface
}

// All utilPackage methods simply delegate to the corresponding function in the 'util' package

func (u *utilPackage) GetManagedClustersForVerrazzanoBinding(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*v8outil.ManagedClusterConnection) (map[string]*v8outil.ManagedClusterConnection, error) {
	return v8outil.GetManagedClustersForVerrazzanoBinding(vzSynMB, availableManagedClusterConnections)
}

// localInterface defines the functions in the 'local' package that are used  by the Controller
type localInterface interface {
	DeleteVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error
	DeleteSecrets(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error
	DeleteConfigMaps(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error
	CreateUpdateVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error
	UpdateConfigMaps(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error
	UpdateAcmeDNSSecret(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error
}

// localPackage is the localInterface implementation through which all 'local' package functions are invoked
type localPackage struct {
	localInterface
}

// All localPackage methods simply delegate to the corresponding function in the 'local' package

func (l *localPackage) DeleteVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {
	return v8olocal.DeleteVmi(binding, vmoClientSet, vmiLister)
}

func (l *localPackage) DeleteSecrets(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error {
	return v8olocal.DeleteSecrets(binding, kubeClientSet, secretLister)
}

func (l *localPackage) DeleteConfigMaps(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	return v8olocal.DeleteConfigMaps(binding, kubeClientSet, configMapLister)
}

func (l *localPackage) CreateUpdateVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error {
	return v8olocal.CreateUpdateVmi(binding, vmoClientSet, vmiLister, verrazzanoURI, enableMonitoringStorage)
}

func (l *localPackage) UpdateConfigMaps(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	return v8olocal.UpdateConfigMaps(binding, kubeClientSet, configMapLister)
}

func (l *localPackage) UpdateAcmeDNSSecret(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error {
	return v8olocal.UpdateAcmeDNSSecret(binding, kubeClientSet, secretLister, name, verrazzanoURI)
}

// monitoringInterface defines the functions in the 'monitoring' package that are used  by the Controller
type monitoringInterface interface {
	CreateVmiSecrets(binding *types.SyntheticBinding, secrets v8omonitoring.Secrets) error
	DeletePomPusher(binding string, helper v8outil.DeploymentHelper) error
}

// monitoringPackage is the monitoringInterface implementation through which all 'monitoring' package functions are invoked
type monitoringPackage struct {
	monitoringInterface
}

// All monitoringPackage methods simply delegate to the corresponding function in the 'monitoring' package

func (m *monitoringPackage) CreateVmiSecrets(binding *types.SyntheticBinding, secrets v8omonitoring.Secrets) error {
	return v8omonitoring.CreateVmiSecrets(binding, secrets)
}

func (m *monitoringPackage) DeletePomPusher(binding string, helper v8outil.DeploymentHelper) error {
	return v8omonitoring.DeletePomPusher(binding, helper)
}

// MCRegistrationSecret - the name of the secret that contains the cluster registration information
const MCRegistrationSecret = "verrazzano-cluster-registration"

// ClusterNameData - the field name in MCRegistrationSecret that contains this managed cluster's name
const ClusterNameData = "managed-cluster-name"

// ElasticsearchSecretName - the name of the secret in the Verrazzano System namespace,
// that contains credentials and other details for for the admin cluster's Elasticsearch endpoint
const ElasticsearchSecretName = "verrazzano-cluster-elasticsearch"

// ElasticsearchURLData - the field name in ElasticsearchSecret that contains the admin cluster's
// Elasticsearch endpoint's URL
const ElasticsearchURLData = "url"

// ElasticsearchUsernameData - the field name in ElasticsearchSecret that contains the admin
// cluster's Elasticsearch username
const ElasticsearchUsernameData = "username"

// ElasticsearchPasswordData - the field name in ElasticsearchSecret that contains the admin
// cluster's Elasticsearch password
const ElasticsearchPasswordData = "password"

// ElasticsearchCABundleData - the field name in ElasticsearchSecret that contains the admin
// cluster's Elasticsearch ca-bundle
const ElasticsearchCABundleData = "ca-bundle"

// getClusterInfo returns the cluster info, including container runtime, and cluster name
// as well as admin cluster Elasticsearch details if it is a managed cluster
func (c *Controller) getClusterInfo() v8omonitoring.ClusterInfo {
	clusterInfo := v8omonitoring.ClusterInfo{
		ContainerRuntime: c.getContainerRuntime(),
	}
	clusterSecret, err1 := c.secrets.Get(MCRegistrationSecret)
	elasticsearchSecret, err2 := c.secrets.Get(ElasticsearchSecretName)
	if err1 == nil && err2 == nil {
		clusterInfo.ManagedClusterName = string(clusterSecret.Data[ClusterNameData])
		clusterInfo.ElasticsearchURL = string(elasticsearchSecret.Data[ElasticsearchURLData])
		clusterInfo.ElasticsearchUsername = string(elasticsearchSecret.Data[ElasticsearchUsernameData])
		clusterInfo.ElasticsearchPassword = string(elasticsearchSecret.Data[ElasticsearchPasswordData])
		clusterInfo.ElasticsearchCABundle = elasticsearchSecret.Data[ElasticsearchCABundleData]
	}
	return clusterInfo
}
