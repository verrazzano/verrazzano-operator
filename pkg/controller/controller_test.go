// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
	vmoclientset "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"testing"
)

var testClusterName = "test-cluster"
var testVerrazzanoURI = "/verrazzano/uri"

var testPodInformer = &controllertest.FakeInformer{}
var testDeploymentInformer = &controllertest.FakeInformer{}
var testNamespaceInformer = &controllertest.FakeInformer{}
var testSecretInformer = &controllertest.FakeInformer{}

var testManagedCluster = &v1beta1.VerrazzanoManagedCluster{
	ObjectMeta: metav1.ObjectMeta{Name: testClusterName, Namespace: "ns1"},
	Spec:       v1beta1.VerrazzanoManagedClusterSpec{KubeconfigSecret: "test-secret"}}

var testManagedClusterConnection = util.ManagedClusterConnection{
	PodInformer:        testPodInformer,
	DeploymentInformer: testDeploymentInformer,
	NamespaceInformer:  testNamespaceInformer,
	SecretInformer:     testSecretInformer}

var testFilteredConnections = map[string]*util.ManagedClusterConnection{testClusterName: &testManagedClusterConnection}

var testImagePullSecrets = []v1.LocalObjectReference{{Name: "test-image-pull-secret"}}
var testSecretData = []byte("test-secret-data")
var testSecretLister = &testutil.SimpleSecretLister{KubeClient: testClientset}

var testKubeSecrets = &KubeSecrets{
	namespace: constants.VerrazzanoNamespace, kubeClientSet: testClientset, secretLister: testSecretLister}

var testClientset = testclient.NewSimpleClientset(&v1.ServiceAccount{
	ImagePullSecrets: testImagePullSecrets,
	ObjectMeta:       metav1.ObjectMeta{Name: "verrazzano-operator", Namespace: "verrazzano-system"}})

// TestNewController tests creation of a Controller from kubeconfig.
// This test mocks reading of the kubeconfig file
// GIVEN a kubeconfig
// WHEN I create a controller
// THEN the PUBLIC state of the controller is in the expected state for a non-running controller
func TestNewController(t *testing.T) {
	binding := &types.LocationInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	// pass in funcs to provide expectations for local and managed packages from NewController invocation
	localMockSetupFunc := func(localMock *testLocalPackage) {
		localMock.CreateUpdateVmi(binding, AnyVmoClientset{}, AnyVmoLister{}, testVerrazzanoURI, "false")
		localMock.UpdateConfigMaps(binding, testClientset, AnyConfigMapLister{})
	}
	monitoringMockSetupFunc := func(monitoringMock *testMonitoringPackage) {
		monitoringMock.CreateVmiSecrets(binding, AnySecrets{})
	}

	// createController invokes NewController which is the function that is being tested
	controller := createController(t,localMockSetupFunc, monitoringMockSetupFunc)

	// assert initial lister state
	listers := controller.ListerSet()
	assert.Equal(t, controller.verrazzanoManagedClusterLister, *listers.ManagedClusterLister)
	assert.Equal(t, 0, len(*listers.ModelBindingPairs))
	assert.NotNil(t, listers.KubeClientSet)
}

// TestProcessManagedCluster test setting up of a managed cluster
// GIVEN a test managed cluster
// WHEN I invoke processManagedCluster
// THEN the managed cluster is added to the controller
// AND all expected external invocations are made for processing a managed cluster
func TestProcessManagedCluster(t *testing.T) {
	controller := createController(t, nil, nil)

	model := &types.ClusterModel{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	binding := &types.LocationInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VmiSystemBindingName,
		},
	}

	mbPair := CreateModelBindingPair(model, binding, controller.verrazzanoURI, testImagePullSecrets)

	mc := &types.ManagedCluster{
		Name:        testManagedCluster.Name,
		Secrets:     map[string][]string{},
		Ingresses:   map[string][]*types.Ingress{},
		RemoteRests: map[string][]*types.RemoteRestConnection{},
	}
	mbPair.ManagedClusters[testManagedCluster.Name] = mc
	mc.Namespaces = append(mc.Namespaces, constants.MonitoringNamespace, constants.LoggingNamespace)

	// set expectations for 'managed' package interactions
	managedMock := controller.managed.(*testManagedPackage)
	managedMock.BuildManagedClusterConnection(testSecretData, controller.stopCh)
	managedMock.CreateNamespaces(mbPair, testFilteredConnections)
	managedMock.CreateSecrets(mbPair, controller.managedClusterConnections, controller.kubeClientSet, controller.secrets)
	managedMock.CreateServiceAccounts(mbPair.Binding.Name, mbPair.ImagePullSecrets, mbPair.ManagedClusters, testFilteredConnections)
	managedMock.CreateConfigMaps(mbPair, testFilteredConnections)
	managedMock.CreateClusterRoles(mbPair, testFilteredConnections)
	managedMock.CreateClusterRoleBindings(mbPair, testFilteredConnections)
	managedMock.CreateServices(mbPair, testFilteredConnections)
	managedMock.CreateDeployments(mbPair, controller.managedClusterConnections,  controller.verrazzanoURI, controller.secrets)
	managedMock.CreateDaemonSets(mbPair, controller.managedClusterConnections, controller.verrazzanoURI)
	managedMock.SetupComplete()

	// record expected 'util' interactions
	utilMock := controller.util.(*testUtilPackage)
	utilMock.GetManagedClustersForVerrazzanoBinding(mbPair, controller.managedClusterConnections)
	utilMock.SetupComplete()

	// invoke method that is being tested
	controller.processManagedCluster(testManagedCluster)

	assert.Equal(t, &testManagedClusterConnection, controller.managedClusterConnections[testClusterName])

	managedMock.Verify(t)
	utilMock.Verify(t)
}

// TestProcessApplicationModelAdded tests adding an application model to the Controller
// GIVEN a new application model tht is unknown to the Controller
// WHEN I invoke processApplicationModelAdded
// THEN the Controller is updated with the new application model and a new model binding pair
func TestProcessApplicationModelAdded(t *testing.T) {
	controller := createController(t, nil, nil)

	binding := &types.LocationInfo{ObjectMeta: metav1.ObjectMeta{Name: "test-binding"}, Spec: types.LocationInfoSpec{ModelName: "test-model"}}
	controller.applicationBindings["test-binding"] = binding
	controller.imagePullSecrets = testImagePullSecrets

	model := &types.ClusterModel{ObjectMeta: metav1.ObjectMeta{Name: "test-model"}}

	// invoke the method being tested
	controller.processApplicationModelAdded(model)

	assert.Same(t, model, controller.applicationModels["test-model"])
	assert.Len(t, controller.modelBindingPairs, 1)
	assert.NotNil(t, controller.modelBindingPairs["test-binding"])
	assert.Same(t, binding, controller.modelBindingPairs["test-binding"].Binding)
	assert.Equal(t, model, controller.modelBindingPairs["test-binding"].Model)
	assert.Equal(t, testVerrazzanoURI, controller.modelBindingPairs["test-binding"].VerrazzanoURI)
	assert.Equal(t, testImagePullSecrets, controller.modelBindingPairs["test-binding"].ImagePullSecrets)
}

// TestProcessApplicationModelAddedVersionExists tests adding an application model that is already known to the Controller
// GIVEN an existing application model
// WHEN I add the previously existing model to the Controller
// THEN the invocation returns without error and no new model is added
func TestProcessApplicationModelAddedVersionExists(t *testing.T) {
	controller := createController(t,nil, nil)

	model := &types.ClusterModel{ObjectMeta: metav1.ObjectMeta{Name: "test-model", ResourceVersion: "test1"}}
	model2 := &types.ClusterModel{ObjectMeta: metav1.ObjectMeta{Name: "test-model", ResourceVersion: "test1"}}
	controller.applicationModels["test-model"] = model
	controller.processApplicationModelAdded(model2)

	assert.Same(t, model, controller.applicationModels["test-model"])
	assert.Len(t, controller.modelBindingPairs, 0)
}

// TestProcessApplicationModelDeleted tests deleting of an application model
// GIVEN an existing application model
// WHEN the existing model is deleted by invoking processApplicationModelDeleted on the Controller
// THEN the deleted model is deleted from the Controller
func TestProcessApplicationModelDeleted(t *testing.T) {
	controller := createController(t, nil, nil)

	model := &types.ClusterModel{ObjectMeta: metav1.ObjectMeta{Name: "test-model", ResourceVersion: "test1"}}
	controller.applicationModels[model.Name] = model

	// sanity check
	assert.Equal(t, model, controller.applicationModels[model.Name])

	controller.processApplicationModelDeleted(model)
	assert.Len(t, controller.applicationModels, 0)
}

// TestProcessApplicationBindingAdded tests adding an application binding to the Controller
// GIVEN a new application binding
// WHEN I add the new binding to the Controller by invoking processApplicationBindingAdded
// THEN the new binding and a new model binding pair are added to the controller
// AND all expected external invocations are made for processing a new application binding
func TestProcessApplicationBindingAdded(t *testing.T) {
	// get controller
	controller := createController(t, nil, nil)

	binding := &types.LocationInfo{ObjectMeta: metav1.ObjectMeta{Name: "test-binding", Namespace: "test-namespace", Finalizers: []string{bindingFinalizer}},
		Spec: types.LocationInfoSpec{
			ModelName: "test-model",
			Placement: []types.ClusterPlacement{{Name: "test-placement", Namespaces: []types.KubernetesNamespace{{Name: "test-namespace"}}}}}}

	// add the corresponding model
	model := &types.ClusterModel{ObjectMeta: metav1.ObjectMeta{Name: "test-model", ResourceVersion: "test1"}}
	controller.applicationModels["test-model"] = model

	modelBindingPair := CreateModelBindingPair(model, binding, testVerrazzanoURI, nil)

	// record expected 'util' interactions
	utilMock := controller.util.(*testUtilPackage)
	utilMock.GetManagedClustersForVerrazzanoBinding(modelBindingPair, controller.managedClusterConnections)
	utilMock.SetupComplete()

	// record expected 'managed' interactions
	managedMock := controller.managed.(*testManagedPackage)
	managedMock.CreateNamespaces(modelBindingPair, testFilteredConnections)
	managedMock.CreateSecrets(modelBindingPair, controller.managedClusterConnections, controller.kubeClientSet, controller.secrets)
	managedMock.CreateServiceAccounts(modelBindingPair.Binding.Name, modelBindingPair.ImagePullSecrets, modelBindingPair.ManagedClusters, testFilteredConnections)
	managedMock.CreateConfigMaps(modelBindingPair, testFilteredConnections)
	managedMock.CreateClusterRoles(modelBindingPair, testFilteredConnections)
	managedMock.CreateClusterRoleBindings(modelBindingPair, testFilteredConnections)
	managedMock.CreateServices(modelBindingPair, testFilteredConnections)
	managedMock.CreateDeployments(modelBindingPair, testFilteredConnections, controller.verrazzanoURI, controller.secrets)
	managedMock.CreateDaemonSets(modelBindingPair, testFilteredConnections, controller.verrazzanoURI)
	managedMock.SetupComplete()

	localMock := controller.local.(*testLocalPackage)
	localMock.CreateUpdateVmi(binding, controller.vmoClientSet, controller.vmiLister, testVerrazzanoURI, controller.enableMonitoringStorage)
	localMock.UpdateConfigMaps(binding, controller.kubeClientSet, controller.configMapLister)
	localMock.UpdateAcmeDNSSecret(binding, controller.kubeClientSet, controller.secretLister, constants.AcmeDNSSecret, testVerrazzanoURI)
	localMock.SetupComplete()

	monitoringMock := controller.monitoring.(*testMonitoringPackage)
	monitoringMock.CreateVmiSecrets(binding, controller.secrets)
	monitoringMock.SetupComplete()

	// call function being tested
	controller.processApplicationBindingAdded(binding)

	assert.Same(t, binding, controller.applicationBindings["test-binding"])
	mbp := controller.modelBindingPairs["test-binding"]
	assert.NotNil(t, mbp)
	assert.Same(t, binding, mbp.Binding)
	assert.Same(t, model, mbp.Model)
	assert.Equal(t, controller.verrazzanoURI, mbp.VerrazzanoURI)
	assert.Equal(t, controller.imagePullSecrets, mbp.ImagePullSecrets)

	// assert managed interaction
	managedMock.Verify(t)
	utilMock.Verify(t)
	localMock.Verify(t)
	monitoringMock.Verify(t)
}

// TestProcessApplicationBindingDeleted tests deleting of an existing application binding from the Controller
// GIVEN an existinig application binding
// WHEN I delete the binding by invoking processApplicationBindingDeleted() on the Controller
// THEN the binding and all associated state is deleted
func TestProcessApplicationBindingDeleted(t *testing.T) {
	controller := createController(t,nil, nil)

	binding := &types.LocationInfo{ObjectMeta: metav1.ObjectMeta{Name: "test-binding", Namespace: "test-namespace", Finalizers: []string{bindingFinalizer}},
		Spec: types.LocationInfoSpec{
			ModelName: "test-model"}}

	model := &types.ClusterModel{ObjectMeta: metav1.ObjectMeta{Name: "test-model", ResourceVersion: "test1"}}
	mbPair := CreateModelBindingPair(model, binding, testVerrazzanoURI, nil)

	controller.applicationModels["test-model"] = model
	controller.applicationBindings["test-binding"] = binding

	controller.modelBindingPairs["test-binding"] = mbPair

	// set expectations of local package interactions
	localMock := controller.local.(*testLocalPackage)
	localMock.DeleteVmi(binding, controller.vmoClientSet, controller.vmiLister)
	localMock.DeleteSecrets(binding, controller.kubeClientSet, controller.secretLister)
	localMock.DeleteConfigMaps(binding, controller.kubeClientSet, controller.configMapLister)
	localMock.SetupComplete()

	// set expectations of managed package interaction
	managedMock := controller.managed.(*testManagedPackage)
	managedMock.DeleteCustomResources(mbPair, controller.managedClusterConnections)
	managedMock.DeleteClusterRoleBindings(mbPair, controller.managedClusterConnections, true)
	managedMock.DeleteClusterRoles(mbPair, controller.managedClusterConnections, true)
	managedMock.DeleteNamespaces(mbPair, controller.managedClusterConnections, true)
	managedMock.DeleteClusterRoleBindings(mbPair, controller.managedClusterConnections, false)
	managedMock.DeleteClusterRoles(mbPair, controller.managedClusterConnections, false)
	managedMock.DeleteNamespaces(mbPair, controller.managedClusterConnections, false)
	managedMock.DeleteServices(mbPair, testFilteredConnections)
	managedMock.DeleteDeployments(mbPair, testFilteredConnections)
	managedMock.SetupComplete()

	monitoringMock := controller.monitoring.(*testMonitoringPackage)
	monitoringMock.DeletePomPusher(binding.Name, &kubeDeployment{kubeClientSet: controller.kubeClientSet})
	monitoringMock.SetupComplete()

	utilMock := controller.util.(*testUtilPackage)
	utilMock.GetManagedClustersForVerrazzanoBinding(mbPair, controller.managedClusterConnections)
	utilMock.SetupComplete()

	// invoke method being tested
	controller.processApplicationBindingDeleted(binding)

	assert.Len(t, controller.applicationBindings, 0)
	assert.Len(t, controller.modelBindingPairs, 0)

	managedMock.Verify(t)
	localMock.Verify(t)
	monitoringMock.Verify(t)
	utilMock.Verify(t)
}

// Create Controller instances for the tests.
// Replaces several external functions used by the Controller to allow for unit testing
func createController(t *testing.T,localMockSetup func(*testLocalPackage), monitoringMockSetup func(*testMonitoringPackage)) *Controller {

	// rewrite the function that is used to get the managed package implementation during creation of a new controller
	originalGetManagedFunc := newManagedPackage
	newManagedPackage = func() managedInterface {
		return newTestManaged()
	}
	defer func() { newManagedPackage = originalGetManagedFunc }()

	// rewrite the function that is used to get the cache package implementation during creation of a new controller
	originalGetCacheFunc := newCachePackage
	newCachePackage = func() cacheInterface {
		return newTestCache()
	}
	defer func() { newCachePackage = originalGetCacheFunc }()

	// rewrite the function that is used to get the util package implementation during creation of a new controller
	originalGetUtilFunc := newUtilPackage
	newUtilPackage = func() utilInterface {
		return newTestUtil(nil)
	}
	defer func() { newUtilPackage = originalGetUtilFunc }()

	// rewrite the function that is used to get the local package implementation during creation of a new controller
	originalGetLocalFunc := newLocalPackage
	newLocalPackage = func() localInterface {
		return newTestLocal(localMockSetup)
	}
	defer func() { newLocalPackage = originalGetLocalFunc }()

	// rewrite the function that is used to get the monitoring package implementation during creation of a new controller
	originalGetMonitoringFunc := newMonitoringPackage
	newMonitoringPackage = func() monitoringInterface {
		return newTestMonitoring(monitoringMockSetup)
	}
	defer func() { newMonitoringPackage = originalGetMonitoringFunc }()

	// rewrite the function that is used to setup the signal handler during creation of a new controller
	originalSetupSignalHandler := setupSignalHandler
	setupSignalHandler = func() (stopCh <-chan struct{}) {
		return make(chan struct{})
	}
	defer func() { setupSignalHandler = originalSetupSignalHandler }()

	// rewrite the function that is used to build a Kube clientset from config during creation of new controller
	var buildKubeClientSetArg *rest.Config
	buildKubeClientSet = func(config *rest.Config) (kubernetes.Interface, error) {
		buildKubeClientSetArg = config
		return testClientset, nil
	}

	// Rewrite the function that is used to create and update VMI and ConfigMap. This code is normally
	// executed as a goroutine. The test implementation doesn't run in a goroutine.
	originalExecuteCreateUpdateGlobaEntities := executeCreateUpdateGlobalEntities
	executeCreateUpdateGlobalEntities = testExecuteCreateUpdateGlobaEntitiesGoroutine
	defer func() { executeCreateUpdateGlobalEntities = originalExecuteCreateUpdateGlobaEntities }()

	config := &rest.Config{}

	// Create a new Controller instance
	controller, err := NewController(config, "", testVerrazzanoURI, "false")

	// set secretLister
	controller.secretLister = testSecretLister
	secretData := map[string][]byte{"kubeconfig": testSecretData}
	secret := v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-secret"}, Data: secretData}
	controller.kubeClientSet.CoreV1().Secrets("ns1").Create(context.TODO(), &secret, metav1.CreateOptions{})

	controller.verrazzanoOperatorClientSet = fake.NewSimpleClientset()
	controller.managed = newTestManaged()
	controller.cache = newTestCache()

	assert.Nil(t, err)
	assert.NotNil(t, controller)
	assert.Equal(t, config, buildKubeClientSetArg)

	// reset all mocks after creation so that tests may set expectations after creation of controller
	controller.managed.(*testManagedPackage).Verify(t)
	controller.cache.(*testCachePackage).Verify(t)
	controller.util.(*testUtilPackage).Verify(t)
	controller.local.(*testLocalPackage).Verify(t)
	controller.monitoring.(*testMonitoringPackage).Verify(t)

	return controller
}

// test implementation of managed package interface
type testManagedPackage struct {
	managedInterface
	testutil.Mock
}

// used to create new test managed package instance
func newTestManaged() *testManagedPackage {
	t := testManagedPackage{
		Mock: *testutil.NewMock(),
	}
	// go right into recording as the controller calls this directly and no expectations are set in test code
	t.SetupComplete()
	return &t
}

func (t *testManagedPackage) BuildManagedClusterConnection(kubeConfigContents []byte, stopCh <-chan struct{}) (*util.ManagedClusterConnection, error) {
	t.Record("BuildManagedClusterConnection", map[string]interface{}{
		"kubeConfigContents": kubeConfigContents,
		"stopCh":             stopCh})
	return &testManagedClusterConnection, nil
}

func (t *testManagedPackage) CreateCrdDefinitions(managedClusterConnection *util.ManagedClusterConnection, managedCluster *v1beta1.VerrazzanoManagedCluster) error {
	t.Record("CreateCrdDefinitions", map[string]interface{}{
		"managedClusterConnection": managedClusterConnection,
		"managedCluster":           managedCluster})
	return nil
}

func (t *testManagedPackage) CreateNamespaces(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateNamespaces", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateSecrets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec monitoring.Secrets) error {
	t.Record("CreateSecrets", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"kubeClientSet":                      kubeClientSet,
		"sec":                                sec})
	return nil
}

func (t *testManagedPackage) CreateServiceAccounts(bindingName string, imagePullSecrets []v1.LocalObjectReference, managedClusters map[string]*types.ManagedCluster, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateServiceAccounts", map[string]interface{}{
		"bindingName":         bindingName,
		"imagePullSecrets":    imagePullSecrets,
		"managedClusters":     managedClusters,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateConfigMaps(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateConfigMaps", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateClusterRoles(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateClusterRoles", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateClusterRoleBindings(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateClusterRoleBindings", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}
func (t *testManagedPackage) CreateIngresses(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateIngresses", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateServiceEntries(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateServiceEntries", map[string]interface{}{
		"mbPair":                             mbPair,
		"filteredConnections":                filteredConnections,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CreateServices(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateServices", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, verrazzanoURI string, sec monitoring.Secrets) error {
	t.Record("CreateDeployments", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"sec":                                sec})
	return nil
}

func (t *testManagedPackage) CreateCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, stopCh <-chan struct{}) error {
	t.Record("CreateCustomResources", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"stopCh":                             stopCh})
	return nil
}

func (t *testManagedPackage) UpdateIstioPrometheusConfigMaps(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("UpdateIstioPrometheusConfigMaps", map[string]interface{}{
		"mbPair":                             mbPair,
		"secretLister":                       secretLister,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CreateDaemonSets(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, verrazzanoURI string) error {
	t.Record("CreateDaemonSets", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"verrazzanoURI":                      verrazzanoURI})
	return nil
}

func (t *testManagedPackage) CreateDestinationRules(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateDestinationRules", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateAuthorizationPolicies(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateAuthorizationPolicies", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, stopCh <-chan struct{}) error {
	t.Record("CleanupOrphanedCustomResources", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"stopCh":                             stopCh})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedServiceEntries(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedServiceEntries", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedIngresses(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedIngresses", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedClusterRoleBindings", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedClusterRoles", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedConfigMaps(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedConfigMaps", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allMbPairs map[string]*types.ModelBindingPair) error {
	t.Record("CleanupOrphanedNamespaces", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"allMbPairs":                         allMbPairs})
	return nil
}

func (t *testManagedPackage) DeleteCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("DeleteCustomResources", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) DeleteClusterRoleBindings(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	t.Record("DeleteClusterRoleBindings", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"bindingLabel":                       bindingLabel})
	return nil
}

func (t *testManagedPackage) DeleteClusterRoles(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	t.Record("DeleteClusterRoles", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"bindingLabel":                       bindingLabel})
	return nil
}

func (t *testManagedPackage) DeleteNamespaces(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	t.Record("DeleteNamespaces", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"bindingLabel":                       bindingLabel})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedServices(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedServices", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedDeployments(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedDeployments", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) DeleteServices(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("DeleteServices", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) DeleteDeployments(mbPair *types.ModelBindingPair, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("DeleteDeployments", map[string]interface{}{
		"mbPair":              mbPair,
		"filteredConnections": filteredConnections})
	return nil
}

// test implementation of cache package interface
type testCachePackage struct {
	cacheInterface
	testutil.Mock
}

// used to create new test cache package instance
func newTestCache() *testCachePackage {
	t := testCachePackage{
		Mock: *testutil.NewMock(),
	}

	// go right into recording as the controller calls this directly from NewController
	t.SetupComplete()
	return &t
}

func (c *testCachePackage) WaitForCacheSync(stopCh <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool {
	c.Record("WaitForCacheSync", map[string]interface{}{
		"stopCh":     stopCh,
		"cacheSyncs": cacheSyncs})
	return true
}

// test implementation of util package interface
type testUtilPackage struct {
	utilInterface
	testutil.Mock
}

// used to create new test util package instance
func newTestUtil(expectations func(*testUtilPackage)) *testUtilPackage {
	t := &testUtilPackage{
		Mock: *testutil.NewMock(),
	}
	// if expectations func is provided, invoke to set expectations on mock
	if expectations != nil {
		expectations(t)
	}

	// go right into recording as the controller calls this directly in NewController
	t.SetupComplete()
	return t
}

func (u *testUtilPackage) GetManagedClustersForVerrazzanoBinding(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) (map[string]*util.ManagedClusterConnection, error) {
	u.Record("GetManagedClustersForVerrazzanoBinding", map[string]interface{}{
		"mbPair":                             mbPair,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return map[string]*util.ManagedClusterConnection{testManagedCluster.Name: &testManagedClusterConnection}, nil
}

// test implementation of local package interface
type testLocalPackage struct {
	localInterface
	testutil.Mock
}

// used to create new test local package instance
func newTestLocal(expectations func(*testLocalPackage)) *testLocalPackage {
	t := &testLocalPackage{
		Mock: *testutil.NewMock(),
	}
	// if expectations function is provided, invoke to set expectations on mock
	if expectations != nil {
		expectations(t)
	}

	// go right into recording as the controller calls this directly from NewController
	t.SetupComplete()
	return t
}

func (l *testLocalPackage) DeleteVmi(binding *types.LocationInfo, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {
	l.Record("DeleteVmi", map[string]interface{}{
		"binding":      binding,
		"vmoClientSet": vmoClientSet,
		"vmiLister":    vmiLister})

	return nil
}

func (l *testLocalPackage) DeleteSecrets(binding *types.LocationInfo, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error {
	l.Record("DeleteSecrets", map[string]interface{}{
		"binding":       binding,
		"kubeClientSet": kubeClientSet,
		"secretLister":  secretLister})

	return nil
}

func (l *testLocalPackage) DeleteConfigMaps(binding *types.LocationInfo, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	l.Record("DeleteConfigMaps", map[string]interface{}{
		"binding":         binding,
		"kubeClientSet":   kubeClientSet,
		"configMapLister": configMapLister})

	return nil
}

func (l *testLocalPackage) CreateUpdateVmi(binding *types.LocationInfo, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error {
	l.Record("CreateUpdateVmi", map[string]interface{}{
		"binding":                 binding,
		"vmoClientSet":            vmoClientSet,
		"vmiLister":               vmiLister,
		"verrazzanoURI":           verrazzanoURI,
		"enableMonitoringStorage": enableMonitoringStorage})
	return nil
}

func (l *testLocalPackage) UpdateConfigMaps(binding *types.LocationInfo, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	l.Record("UpdateConfigMaps", map[string]interface{}{
		"binding":         binding,
		"kubeClientSet":   kubeClientSet,
		"configMapLister": configMapLister})
	return nil
}

func (l *testLocalPackage) UpdateAcmeDNSSecret(binding *types.LocationInfo, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error {
	l.Record("UpdateAcmeDNSSecret", map[string]interface{}{
		"binding":       binding,
		"kubeClientSet": kubeClientSet,
		"secretLister":  secretLister,
		"verrazzanoURI": verrazzanoURI})
	return nil
}

// test implementation of monitoring package interface
type testMonitoringPackage struct {
	monitoringInterface
	testutil.Mock
}

// used to create new test monitoring package instance
func newTestMonitoring(expectations func(*testMonitoringPackage)) *testMonitoringPackage {
	t := &testMonitoringPackage{
		Mock: *testutil.NewMock(),
	}
	// if expectations function is provided, invoke to set expectations on mock
	if expectations != nil {
		expectations(t)
	}

	// go right into recording as the controller calls this directly from NewController
	t.SetupComplete()
	return t
}

func (m *testMonitoringPackage) CreateVmiSecrets(binding *types.LocationInfo, secrets monitoring.Secrets) error {
	m.Record("CreateVmiSecrets", map[string]interface{}{
		"binding": binding,
		"secrets": secrets})

	return nil
}
func (m *testMonitoringPackage) DeletePomPusher(binding string, helper util.DeploymentHelper) error {
	m.Record("DeletePomPusher", map[string]interface{}{
		"binding": binding,
		"helper":  helper})

	return nil
}

func testExecuteCreateUpdateGlobaEntitiesGoroutine(binding *types.LocationInfo, c *Controller) {
	createUpdateGlobalEntities(binding, c)
}

// used to match any vmo clientset in mock expectations
type AnyVmoClientset struct {
	vmoclientset.Interface
	testutil.Any
}

// used to match any configMap lister in mock expression
type AnyConfigMapLister struct {
	corev1listers.ConfigMapLister
	testutil.Any
}

// used to match any vmo lister in mock expectations
type AnyVmoLister struct {
	vmolisters.VerrazzanoMonitoringInstanceLister
	testutil.Any
}

// used to match any secret in mock expectations
type AnySecrets struct {
	monitoring.Secrets
	testutil.Any
}
