// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	clientsetscheme "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/scheme"
	informers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/informers/externalversions"
	listers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/listers/verrazzano/v1beta1"
	vmoclientset "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	vmoinformers "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/informers/externalversions"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/local"
	"github.com/verrazzano/verrazzano-operator/pkg/managed"
	"github.com/verrazzano/verrazzano-operator/pkg/signals"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
)

const controllerAgentName = "verrazzano-controller"
const bindingFinalizer = "vb.verrazzano.io"

// Controller represents the primary controller structure.
type Controller struct {
	kubeClientSet               kubernetes.Interface
	kubeExtClientSet            apiextensionsclient.Interface
	verrazzanoOperatorClientSet clientset.Interface
	vmoClientSet                vmoclientset.Interface
	verrazzanoURI               string
	enableMonitoringStorage     string
	imagePullSecrets            []corev1.LocalObjectReference

	// Local cluster listers and informers
	secretLister                     corev1listers.SecretLister
	secretInformer                   cache.SharedIndexInformer
	configMapLister                  corev1listers.ConfigMapLister
	configMapInformer                cache.SharedIndexInformer
	verrazzanoManagedClusterLister   listers.VerrazzanoManagedClusterLister
	verrazzanoManagedClusterInformer cache.SharedIndexInformer
	VerrazzanoModelLister            listers.VerrazzanoModelLister
	VerrazzanoModelInformer          cache.SharedIndexInformer
	VerrazzanoBindingLister          listers.VerrazzanoBindingLister
	VerrazzanoBindingInformer        cache.SharedIndexInformer
	vmiLister                        vmolisters.VerrazzanoMonitoringInstanceLister
	vmiInformer                      cache.SharedIndexInformer
	secrets                          monitoring.Secrets
	// The current set of known managed clusters
	managedClusterConnections map[string]*util.ManagedClusterConnection

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

	Manifest *util.Manifest

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
func NewController(kubeconfig string, manifest *util.Manifest, masterURL string, watchNamespace string, verrazzanoURI string, enableMonitoringStorage string) (*Controller, error) {
	//
	// Instantiate connection and clients to local k8s cluster
	//
	glog.V(6).Info("Building config")
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %v", err)
	}

	glog.V(6).Info("Building kubernetes clientset")
	kubeClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %v", err)
	}

	glog.V(6).Info("Building Verrazzano Operator clientset")
	verrazzanoOperatorClientSet, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building verrazzano operator clientset: %v", err)
	}

	glog.V(6).Info("Building VMO clientset")
	vmoClientSet, err := vmoclientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building VMO clientset: %v", err)
	}

	glog.V(6).Info("Building wls-operator clientset")
	wlsoprClientSet, err := wlsoprclientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building wls-operator clientset: %v", err)
	}

	glog.V(6).Info("Building api extensions clientset")
	kubeExtClientSet, err := extclientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building apiextensions-apiserver clientset: %v", err)
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
	vmiInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{})

	clientsetscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	kubeSecrets := &KubeSecrets{
		namespace: constants.VerrazzanoNamespace, kubeClientSet: kubeClientSet, secretLister: secretsInformer.Lister(),
	}
	controller := &Controller{
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
		managedClusterConnections:        map[string]*util.ManagedClusterConnection{},
		applicationModels:                map[string]*v1beta1v8o.VerrazzanoModel{},
		applicationBindings:              map[string]*v1beta1v8o.VerrazzanoBinding{},
		modelBindingPairs:                map[string]*types.ModelBindingPair{},
		Manifest:                         manifest,
		bindingSyncThreshold:             map[string]int{},
		secrets:                          kubeSecrets,
	}

	// Set up signals so we handle the first shutdown signal gracefully
	glog.V(6).Info("Setting up signals")
	stopCh := signals.SetupSignalHandler()

	go kubeInformerFactory.Start(stopCh)
	go verrazzanoOperatorInformerFactory.Start(stopCh)
	go wlsoprInformerFactory.Start(stopCh)
	go vmoInformerFactory.Start(stopCh)

	// Wait for the caches to be synced before starting watchers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(controller.stopCh, controller.secretInformer.HasSynced,
		controller.verrazzanoManagedClusterInformer.HasSynced, controller.VerrazzanoBindingInformer.HasSynced,
		controller.VerrazzanoModelInformer.HasSynced, controller.vmiInformer.HasSynced); !ok {
		return controller, errors.New("Failed to wait for caches to sync")
	}

	// Install Global Entities
	go controller.CreateUpdateGlobalEntities()

	return controller, nil
}

// CreateUpdateGlobalEntities installs global entities
func (c *Controller) CreateUpdateGlobalEntities() error {
	// Create or update System VMI
	// A synthetic binding will be constructed and passed to the generic CreateUpdateVmi API to create the System VMI
	systemBinding := &v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}
	glog.V(4).Info("Configuring System VMI...")
	for {
		err := local.CreateUpdateVmi(systemBinding, c.vmoClientSet, c.vmiLister, c.verrazzanoURI, c.enableMonitoringStorage)
		if err != nil {
			glog.Errorf("Failed to create System VMI %s: %v", constants.VmiSystemBindingName, err)
		}

		// Create secrets
		err = monitoring.CreateVmiSecrets(systemBinding, c.secrets)
		if err != nil {
			glog.Errorf("Failed to create secrets for binding %s: %v", systemBinding.Name, err)
		}

		// Update config maps for system vmi
		err = local.UpdateConfigMaps(systemBinding, c.kubeClientSet, c.configMapLister)
		if err != nil {
			glog.Errorf("Failed to update ConfigMaps for binding %s: %v", systemBinding.Name, err)
		}

		<-time.After(60 * time.Second)
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
	glog.Info("Starting Verrazzano Operator controller")

	glog.Info("Starting watchers")
	c.startLocalWatchers()
	<-c.stopCh
	return nil
}

// Start watchers on the local k8s cluster
func (c *Controller) startLocalWatchers() {

	//
	// VerrazzanoManagedClusters
	//
	c.verrazzanoManagedClusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { c.processManagedCluster(new) },
		UpdateFunc: func(old, new interface{}) { c.processManagedCluster(new) },
	})

	//
	// VerrazzanoBindings
	//
	c.VerrazzanoBindingInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { c.processApplicationBindingAdded(new) },
		UpdateFunc: func(old, new interface{}) { c.processApplicationBindingAdded(new) },
		DeleteFunc: func(new interface{}) { c.processApplicationBindingDeleted(new) },
	})

	//
	// VerrazzanoModels
	//
	c.VerrazzanoModelInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { c.processApplicationModelAdded(new) },
		UpdateFunc: func(old, new interface{}) { c.processApplicationModelAdded(new) },
		DeleteFunc: func(new interface{}) { c.processApplicationModelDeleted(new) },
	})
}

// Process a change to a VerrazzanoManagedCluster
func (c *Controller) processManagedCluster(cluster interface{}) {
	// Obtain the optional list of imagePullSecrets from the verrazzano-operator service account
	sa, err := c.kubeClientSet.CoreV1().ServiceAccounts(constants.VerrazzanoSystem).Get(context.TODO(), constants.VerrazzanoOperatorServiceAccount, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("Can't find ServiceAccount %s in namespace %s", constants.VerrazzanoOperatorServiceAccount, constants.VerrazzanoSystem)
		return
	}
	c.imagePullSecrets = sa.ImagePullSecrets

	// A synthetic binding will be constructed and passed to the managed clusters
	systemBinding := &v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	// A synthetic model will be constructed and passed to the managed clusters
	systemModel := &v1beta1v8o.VerrazzanoModel{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	// A synthetic Model binding pair will be constructed and passed to the managed clusters
	mbPair := CreateModelBindingPair(systemModel, systemBinding, c.verrazzanoURI, c.imagePullSecrets)

	managedCluster := cluster.(*v1beta1v8o.VerrazzanoManagedCluster)
	secret, err := c.secretLister.Secrets(managedCluster.Namespace).Get(managedCluster.Spec.KubeconfigSecret)
	if err != nil {
		glog.Errorf("Can't find secret %s for ManagedCluster %s", managedCluster.Spec.KubeconfigSecret, managedCluster.Name)
		return
	}

	// Construct a new client to the managed cluster when it's added or changed
	kubeConfigContents := secret.Data["kubeconfig"]
	_, clusterExists := c.managedClusterConnections[managedCluster.Name]
	if !clusterExists || string(kubeConfigContents) != c.managedClusterConnections[managedCluster.Name].KubeConfig {
		glog.Infof("(Re)creating k8s clients for Managed Cluster %s", managedCluster.Name)

		managedClusterConnection, err := managed.BuildManagedClusterConnection(kubeConfigContents, c.stopCh)
		if err != nil {
			glog.Error(err)
			return
		}

		c.managedClusterConnections[managedCluster.Name] = managedClusterConnection

		mc := &types.ManagedCluster{
			Name:        managedCluster.Name,
			Secrets:     map[string][]string{},
			Ingresses:   map[string][]*types.Ingress{},
			RemoteRests: map[string][]*types.RemoteRestConnection{},
		}
		mbPair.ManagedClusters[managedCluster.Name] = mc

		// Add in the monitoring and logging namespace if not already added
		mc.Namespaces = append(mc.Namespaces, constants.MonitoringNamespace, constants.LoggingNamespace)

		/*********************
		 * Create Artifacts in the Managed Cluster
		 **********************/
		// Create CRD Definitions
		err = managed.CreateCrdDefinitions(c.managedClusterConnections[managedCluster.Name], managedCluster, c.Manifest)
		if err != nil {
			glog.Errorf("Failed to create CRD definitions for ManagedCluster %s: %v", managedCluster.Name, err)
		}

		glog.Info("Starting watchers on " + managedCluster.Name)
		c.startManagedClusterWatchers(managedCluster.Name, mbPair)

		// Create all the components needed by logging and monitoring in managed clusters to push metrics and logs into System VMI in management cluster
		c.createManagedClusterResourcesForBinding(mbPair)

		// Wait for the caches to be synced before starting workers
		glog.Info("Waiting for informer caches to sync")
		if ok := cache.WaitForCacheSync(c.stopCh, managedClusterConnection.DeploymentInformer.HasSynced,
			managedClusterConnection.NamespaceInformer.HasSynced, managedClusterConnection.SecretInformer.HasSynced); !ok {
			fmt.Println(errors.New("failed to wait for caches to sync"))
		}
	}
}

// Start watchers on the given ManagedCluster
func (c *Controller) startManagedClusterWatchers(managedClusterName string, mbPair *types.ModelBindingPair) {
	// Pod event handlers
	c.managedClusterConnections[managedClusterName].PodInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			c.updateIstioPolicies(mbPair, new.(*corev1.Pod), "added")
		},
		DeleteFunc: func(old interface{}) {
			c.updateIstioPolicies(mbPair, old.(*corev1.Pod), "deleted")
		},
	})
}

// update the Istio destination rules and authorization policies for the given model/binding whenever a pod is added or deleted
func (c *Controller) updateIstioPolicies(mbPair *types.ModelBindingPair, pod *corev1.Pod, action string) {
	glog.V(4).Infof("Pod %s : %s %s.", pod.Name, pod.Status.PodIP, action)
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to get filtered connections for binding %s: %v", mbPair.Binding.Name, err)
	}
	err = managed.CreateAuthorizationPolicies(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to update authorization policies for binding %s after pod %s: %v", mbPair.Binding.Name, action, err)
	}
}

// Create managed resources on clusters depending upon the binding
func (c *Controller) createManagedClusterResourcesForBinding(mbPair *types.ModelBindingPair) {
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to get filtered connections for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Namespaces
	err = managed.CreateNamespaces(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create namespaces for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create (copy) the Secrets needed from the management cluster to the managed clusters
	err = managed.CreateSecrets(mbPair, c.managedClusterConnections, c.kubeClientSet, c.secrets)
	if err != nil {
		glog.Errorf("Failed to create secrets for binding %s: %v", mbPair.Binding.Namespace, err)
	}

	// Create ServiceAccounts
	err = managed.CreateServiceAccounts(mbPair.Binding.Name, mbPair.ImagePullSecrets, mbPair.ManagedClusters, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create service accounts for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create ConfigMaps
	err = managed.CreateConfigMaps(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create config maps for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create ClusterRoles
	err = managed.CreateClusterRoles(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create cluster roles for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create ClusterRoleBindings
	err = managed.CreateClusterRoleBindings(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create cluster role bindings for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Ingresses
	err = managed.CreateIngresses(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create ingresses for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create istio ServiceEntries for each remote rest connection
	err = managed.CreateServiceEntries(mbPair, filteredConnections, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to create service entries for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Services
	err = managed.CreateServices(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create service for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Deployments
	err = managed.CreateDeployments(mbPair, filteredConnections, c.Manifest, c.verrazzanoURI, c.secrets)
	if err != nil {
		glog.Errorf("Failed to create deployments for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create Custom Resources
	err = managed.CreateCustomResources(mbPair, c.managedClusterConnections, c.stopCh, c.VerrazzanoBindingLister)
	if err != nil {
		glog.Errorf("Failed to create custom resources for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Update Istio Prometheus ConfigMaps for a given ModelBindingPair on all managed clusters
	err = managed.UpdateIstioPrometheusConfigMaps(mbPair, c.secretLister, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to update Istio Config Map for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create DaemonSets
	err = managed.CreateDaemonSets(mbPair, filteredConnections, c.verrazzanoURI)
	if err != nil {
		glog.Errorf("Failed to create DaemonSets for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create DestinationRules
	err = managed.CreateDestinationRules(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create destination rules for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Create AuthorizationPolicies
	err = managed.CreateAuthorizationPolicies(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to create authorization policies for binding %s: %v", mbPair.Binding.Name, err)
	}
}

// Process a change to a VerrazzanoModel
func (c *Controller) processApplicationModelAdded(cluster interface{}) {
	model := cluster.(*v1beta1v8o.VerrazzanoModel)

	if existingModel, ok := c.applicationModels[model.Name]; ok {
		if existingModel.GetResourceVersion() == model.GetResourceVersion() {
			glog.V(6).Infof("No changes to the model %s", model.Name)
			return
		}

		glog.Infof("Updating the model %s", model.Name)
	} else {
		glog.Infof("Adding the model %s", model.Name)
	}

	// Add or replace the model object
	c.applicationModels[model.Name] = model

	// During restart of the operator a binding can be added before a model so make sure we create
	// model/binding pair now.
	for _, binding := range c.applicationBindings {
		if _, ok := c.modelBindingPairs[binding.Name]; !ok {
			if model.Name == binding.Spec.ModelName {
				glog.Infof("Adding model/binding pair during add model for model %s and binding %s", binding.Spec.ModelName, binding.Name)
				mbPair := CreateModelBindingPair(model, binding, c.verrazzanoURI, c.imagePullSecrets)
				c.modelBindingPairs[binding.Name] = mbPair
				break
			}
		}
	}
}

// Process a removal of a VerrazzanoModel
func (c *Controller) processApplicationModelDeleted(cluster interface{}) {
	model := cluster.(*v1beta1v8o.VerrazzanoModel)

	if _, ok := c.applicationModels[model.Name]; ok {
		glog.Infof("Deleting the model %s", model.Name)
		delete(c.applicationModels, model.Name)
	}

}

func getModelBindingPair(c *Controller, binding *v1beta1v8o.VerrazzanoBinding) (*types.ModelBindingPair, bool) {
	mbPair, mbPairExists := c.modelBindingPairs[binding.Name]
	return mbPair, mbPairExists
}

// Process a change to a VerrazzanoBinding/Sync existing VerrazzanoBinding with cluster state
func (c *Controller) processApplicationBindingAdded(cluster interface{}) {
	binding := cluster.(*v1beta1v8o.VerrazzanoBinding)

	// Make sure the namespaces in this binding are unique across all bindings
	namespaceFound := c.checkNamespacesFound(binding)
	if namespaceFound {
		return
	}

	if binding.GetDeletionTimestamp() != nil {
		if contains(binding.GetFinalizers(), bindingFinalizer) {
			// Add binding in the case where delete has been initiated before the binding is added.
			if _, ok := c.applicationBindings[binding.Name]; !ok {
				glog.Infof("Adding the binding %s", binding.Name)
				c.applicationBindings[binding.Name] = binding
			}
			_, mbPairExists := getModelBindingPair(c, binding)
			if !mbPairExists {
				// During restart of the operator a delete can happen before a model/binding is created so create one now.
				if model, ok := c.applicationModels[binding.Spec.ModelName]; ok {
					glog.Infof("Adding model/binding pair during add binding for model %s and binding %s", binding.Spec.ModelName, binding.Name)
					mbPair := CreateModelBindingPair(model, binding, c.verrazzanoURI, c.imagePullSecrets)
					c.modelBindingPairs[binding.Name] = mbPair
				}
			}
		}

		glog.V(6).Infof("Binding %s is marked for deletion/already deleted", binding.Name)
		if contains(binding.GetFinalizers(), bindingFinalizer) {
			c.processApplicationBindingDeleted(cluster)
		}
		return
	}

	var err error
	if !contains(binding.GetFinalizers(), bindingFinalizer) {
		binding, err = c.addFinalizer(binding)
		if err != nil {
			return
		}
	}

	if existingBinding, ok := c.applicationBindings[binding.Name]; ok {
		if existingBinding.GetResourceVersion() == binding.GetResourceVersion() {
			counter, ok := c.bindingSyncThreshold[binding.Name]
			if !ok {
				c.bindingSyncThreshold[binding.Name] = 0
			} else {
				c.bindingSyncThreshold[binding.Name] = counter + 1
			}

			if counter < 4 {
				glog.V(6).Infof("No changes to binding %s. Skip sync as sync threshold has not passed", binding.Name)
				return
			}

			glog.V(6).Infof("No changes to binding %s. Syncing with cluster state", binding.Name)
			c.bindingSyncThreshold[binding.Name] = 0
		} else {
			glog.Infof("Updating the binding %s", binding.Name)
			c.applicationBindings[binding.Name] = binding
		}

	} else {
		glog.Infof("Adding the binding %s", binding.Name)
		c.applicationBindings[binding.Name] = binding
	}

	// Does a model/binding pair already exist?
	mbPair, mbPairExists := getModelBindingPair(c, binding)
	if !mbPairExists {
		// If a model exists for this binding, then create the model/binding pair
		if model, ok := c.applicationModels[binding.Spec.ModelName]; ok {
			glog.Infof("Adding new model/binding pair during add binding for model %s and binding %s", binding.Spec.ModelName, binding.Name)
			mbPair = CreateModelBindingPair(model, binding, c.verrazzanoURI, c.imagePullSecrets)
			c.modelBindingPairs[binding.Name] = mbPair
			mbPairExists = true
		}
	} else {
		// Rebuild the existing model/binding pair
		if existingModel, ok := c.applicationModels[binding.Spec.ModelName]; ok {
			UpdateModelBindingPair(mbPair, existingModel, binding, c.verrazzanoURI, c.imagePullSecrets)
		}

	}

	// Nothing further to do if the model/binding pair does not exist yet
	if !mbPairExists {
		return
	}

	err = c.waitForManagedClusters(mbPair, binding.Name)
	if err != nil {
		glog.Errorf("Skipping processing of VerrazzanoBinding %s until all referenced Managed Clusters are available, error %s", binding.Name, err.Error())
		return
	}

	/*********************
	 * Create Artifacts in the Local Cluster
	 **********************/
	// Create VMIs
	err = local.CreateUpdateVmi(binding, c.vmoClientSet, c.vmiLister, c.verrazzanoURI, c.enableMonitoringStorage)
	if err != nil {
		glog.Errorf("Failed to create VMIs for binding %s: %v", binding.Name, err)
	}

	// Create secrets
	err = monitoring.CreateVmiSecrets(binding, c.secrets)
	if err != nil {
		glog.Errorf("Failed to create secrets for binding %s: %v", binding.Name, err)
	}

	// Update ConfigMaps
	err = local.UpdateConfigMaps(binding, c.kubeClientSet, c.configMapLister)
	if err != nil {
		glog.Errorf("Failed to update ConfigMaps for binding %s: %v", binding.Name, err)
	}

	// Update the "bring your own DNS" credentials?
	err = local.UpdateAcmeDNSSecret(binding, c.kubeClientSet, c.secretLister, constants.AcmeDNSSecret, c.verrazzanoURI)
	if err != nil {
		glog.Errorf("Failed to update DNS credentials for binding %s: %v", binding.Name, err)
	}

	/*********************
	 * Create Artifacts in the Managed Cluster
	 **********************/
	c.createManagedClusterResourcesForBinding(mbPair)

	// Cleanup any resources no longer needed
	c.cleanupOrphanedResources(mbPair)
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
					glog.Errorf("Binding %s has a conflicting namespace with binding %s.  Namespaces must be unique across bindings.", binding.Name, currBinding.Name)
				}
				glog.Errorf("Duplicate namespace %s found in placement %s", namespace.Name, placement.Name)
				dupFound = true
				break
			}
		}
	}

	return dupFound
}

func (c *Controller) cleanupOrphanedResources(mbPair *types.ModelBindingPair) {
	// Cleanup Custom Resources
	err := managed.CleanupOrphanedCustomResources(mbPair, c.managedClusterConnections, c.stopCh)
	if err != nil {
		glog.Errorf("Failed to cleanup custom resources for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Deployments for generic components
	err = managed.CleanupOrphanedDeployments(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup deployments for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Services for generic components
	err = managed.CleanupOrphanedServices(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup services for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ServiceEntries
	err = managed.CleanupOrphanedServiceEntries(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup ServiceEntries for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Ingresses
	err = managed.CleanupOrphanedIngresses(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup Ingresses for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ClusterRoleBindings
	err = managed.CleanupOrphanedClusterRoleBindings(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup ClusterRoleBindings for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ClusterRoles
	err = managed.CleanupOrphanedClusterRoles(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup ClusterRoles for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup ConfigMaps
	err = managed.CleanupOrphanedConfigMaps(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to cleanup ConfigMaps for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Cleanup Namespaces - this will also cleanup any ServiceAccounts and Secrets within the namespace
	err = managed.CleanupOrphanedNamespaces(mbPair, c.managedClusterConnections, c.modelBindingPairs)
	if err != nil {
		glog.Errorf("Failed to cleanup Namespaces for binding %s: %v", mbPair.Binding.Name, err)
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
func (c *Controller) processApplicationBindingDeleted(cluster interface{}) {
	binding := cluster.(*v1beta1v8o.VerrazzanoBinding)

	if !contains(binding.GetFinalizers(), bindingFinalizer) {
		glog.Infof("Resources for binding %s already deleted", binding.Name)
		return
	}

	mbPair, mbPairExists := getModelBindingPair(c, binding)
	if !mbPairExists {
		return
	}

	glog.Infof("Deleting resources for binding %s", binding.Name)

	/*********************
	 * Delete Artifacts in the Local Cluster
	 **********************/
	// Delete VMIs
	err := local.DeleteVmi(binding, c.vmoClientSet, c.vmiLister)
	if err != nil {
		glog.Errorf("Failed to delete VMIs for binding %s: %v", binding.Name, err)
	}

	// Delete Secrets
	err = local.DeleteSecrets(binding, c.kubeClientSet, c.secretLister)
	if err != nil {
		glog.Errorf("Failed to delete secrets for binding %s: %v", binding.Name, err)
		return
	}

	// Delete ConfigMaps
	err = local.DeleteConfigMaps(binding, c.kubeClientSet, c.configMapLister)
	if err != nil {
		glog.Errorf("Failed to delete ConfigMaps for binding %s: %v", binding.Name, err)
		return
	}

	/*********************
	 * Delete Artifacts in the Managed Cluster
	 **********************/

	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to get filtered connections for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Delete Custom Resources
	err = managed.DeleteCustomResources(mbPair, c.managedClusterConnections)
	if err != nil {
		glog.Errorf("Failed to delete custom resources for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete Deployments for generic components
	err = managed.DeleteDeployments(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to delete deployments for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete Services for generic components
	err = managed.DeleteServices(mbPair, filteredConnections)
	if err != nil {
		glog.Errorf("Failed to delete services for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete ClusterRoleBindings
	err = managed.DeleteClusterRoleBindings(mbPair, c.managedClusterConnections, true)
	if err != nil {
		glog.Errorf("Failed to delete cluster role bindings for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	// Delete ClusterRoles
	err = managed.DeleteClusterRoles(mbPair, c.managedClusterConnections, true)
	if err != nil {
		glog.Errorf("Failed to delete cluster roles for binding %s: %v", mbPair.Binding.Name, err)
		return
	}

	err = monitoring.DeletePomPusher(binding.Name, &kubeDeployment{kubeClientSet: c.kubeClientSet})
	if err != nil {
		glog.Errorf("Failed to delete prometheus-pusher for binding %s: %v", mbPair.Binding.Name, err)
	}

	// Delete Namespaces - this will also cleanup any Ingresses,
	// ServiceEntries, ServiceAccounts, ConfigMaps and Secrets within the namespace
	err = managed.DeleteNamespaces(mbPair, c.managedClusterConnections, true)
	if err != nil {
		glog.Errorf("Failed delete namespaces for binding %s: %v", binding.Name, err)
		return
	}

	// If this is the last model binding to be deleted then we need to cleanup additional resources that are shared
	// among model/binding pairs.
	if len(c.modelBindingPairs) == 1 {
		// Delete ClusterRoleBindings
		err = managed.DeleteClusterRoleBindings(mbPair, c.managedClusterConnections, false)
		if err != nil {
			glog.Errorf("Failed to delete cluster role bindings: %v", err)
			return
		}

		// Delete ClusterRoles
		err = managed.DeleteClusterRoles(mbPair, c.managedClusterConnections, false)
		if err != nil {
			glog.Errorf("Failed to delete cluster roles: %v", err)
			return
		}

		// Delete Namespaces - this will also cleanup any deployments, Ingresses,
		// ServiceEntries, ServiceAccounts, and Secrets within the namespace
		err = managed.DeleteNamespaces(mbPair, c.managedClusterConnections, false)
		if err != nil {
			glog.Errorf("Failed to delete a common namespace for all bindings: %v", err)
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
			glog.V(6).Infof("Waiting for all Managed Clusters referenced in binding %s to be available..", bindingName)
			_, err = util.GetManagedClustersForVerrazzanoBinding(mbPair, c.managedClusterConnections)
			if err == nil {
				glog.V(6).Infof("All Managed Clusters referenced in binding %s are available..", bindingName)
				return nil
			}

			glog.V(4).Info(err.Error())
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
	glog.Infof("Adding finalizer %s to binding %s", bindingFinalizer, binding.Name)
	binding.SetFinalizers(append(binding.GetFinalizers(), bindingFinalizer))

	// Update binding
	binding, err := c.verrazzanoOperatorClientSet.VerrazzanoV1beta1().VerrazzanoBindings(binding.Namespace).Update(context.TODO(), binding, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("Failed adding finalizer %s to binding %s, error %s", bindingFinalizer, binding.Name, err.Error())
		return nil, err
	}
	return binding, nil
}

// Remove finalizer from binding after binding is deleted
func (c *Controller) removeFinalizer(binding *v1beta1v8o.VerrazzanoBinding) error {
	glog.Infof("Removing finalizer %s from binding %s", bindingFinalizer, binding.Name)
	binding.SetFinalizers(remove(binding.GetFinalizers(), bindingFinalizer))

	// Update binding
	_, err := c.verrazzanoOperatorClientSet.VerrazzanoV1beta1().VerrazzanoBindings(binding.Namespace).Update(context.TODO(), binding, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("Failed removing finalizer %s from binding %s, error %s", bindingFinalizer, binding.Name, err.Error())
		return err
	}
	return nil
}
