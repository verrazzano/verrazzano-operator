// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
)

var testClusterName = "test-cluster"
var testVerrazzanoURI = "/verrazzano/uri"

var testPodInformer = &controllertest.FakeInformer{}
var testDeploymentInformer = &controllertest.FakeInformer{}
var testNamespaceInformer = &controllertest.FakeInformer{}
var testSecretInformer = &controllertest.FakeInformer{}

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
	ObjectMeta:       metav1.ObjectMeta{Name: "verrazzano-operator", Namespace: "verrazzano-system"}},
	&testNode)

var testNode = v1.Node{Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{ContainerRuntimeVersion: "docker://19.3.11"}}}
var testNodes = []v1.Node{testNode}
var testNodeList = v1.NodeList{Items: testNodes}

var testClusterInfoDockerStandalone = monitoring.ClusterInfo{ContainerRuntime: "docker://19.3.11"}
var testClusterInfoDockerManaged = monitoring.ClusterInfo{ContainerRuntime: "docker://19.3.11",
	ManagedClusterName: "cluster1", ElasticsearchUsername: "testUsername", ElasticsearchPassword: "testPassword",
	ElasticsearchURL: "testURL", ElasticsearchCABundle: []byte("testCABundle")}

// TestNewController tests creation of a Controller from kubeconfig.
// This test mocks reading of the kubeconfig file
// GIVEN a kubeconfig
// WHEN I create a controller
// THEN the PUBLIC state of the controller is in the expected state for a non-running controller
func TestNewController(t *testing.T) {
	binding := &types.SyntheticBinding{
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
	controller := createController(t, localMockSetupFunc, monitoringMockSetupFunc, false)

	// assert initial lister state
	listers := controller.ListerSet()
	assert.Equal(t, 0, len(*listers.SyntheticModelBindings))
	assert.NotNil(t, listers.KubeClientSet)
}

// Create Controller instances for the tests.
// Replaces several external functions used by the Controller to allow for unit testing
func createController(t *testing.T, localMockSetup func(*testLocalPackage), monitoringMockSetup func(*testMonitoringPackage), managedCluster bool) *Controller {

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
	if managedCluster {
		secret = v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: MCRegistrationSecret},
			Data: map[string][]byte{ClusterNameData: []byte("cluster1"),
				ElasticsearchURLData:      []byte("testURL"),
				ElasticsearchUsernameData: []byte("testUsername"),
				ElasticsearchPasswordData: []byte("testPassword"),
				ElasticsearchCABundleData: []byte("testCABundle")}}
		controller.kubeClientSet.CoreV1().Secrets(constants.VerrazzanoSystem).Create(context.TODO(), &secret, metav1.CreateOptions{})
	}

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

func (t *testManagedPackage) BuildManagedClusterConnection(kubeconfigPath string, stopCh <-chan struct{}) (*util.ManagedClusterConnection, error) {
	t.Record("BuildManagedClusterConnection", map[string]interface{}{
		"kubeconfigPath": kubeconfigPath,
		"stopCh":         stopCh})
	return &testManagedClusterConnection, nil
}

func (t *testManagedPackage) CreateNamespaces(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateNamespaces", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateSecrets(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, kubeClientSet kubernetes.Interface, sec monitoring.Secrets, clusterInfo monitoring.ClusterInfo) error {
	t.Record("CreateSecrets", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
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

func (t *testManagedPackage) CreateConfigMaps(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection, clusterInfo monitoring.ClusterInfo) error {
	t.Record("CreateConfigMaps", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections,
		"clusterInfo":         clusterInfo})
	return nil
}

func (t *testManagedPackage) CreateClusterRoles(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateClusterRoles", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateClusterRoleBindings", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}
func (t *testManagedPackage) CreateIngresses(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateIngresses", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateServiceEntries(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateServiceEntries", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"filteredConnections":                filteredConnections,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CreateServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateServices", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, verrazzanoURI string, sec monitoring.Secrets) error {
	t.Record("CreateDeployments", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"sec":                                sec})
	return nil
}

func (t *testManagedPackage) CreateCustomResources(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, stopCh <-chan struct{}) error {
	t.Record("CreateCustomResources", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"stopCh":                             stopCh})
	return nil
}

func (t *testManagedPackage) UpdateIstioPrometheusConfigMaps(vzSynMB *types.SyntheticModelBinding, secretLister corev1listers.SecretLister, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("UpdateIstioPrometheusConfigMaps", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"secretLister":                       secretLister,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CreateDaemonSets(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, verrazzanoURI string, clusterInfo monitoring.ClusterInfo) error {
	t.Record("CreateDaemonSets", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"verrazzanoURI":                      verrazzanoURI,
		"clusterInfo":                        clusterInfo})
	return nil
}

func (t *testManagedPackage) CreateDestinationRules(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateDestinationRules", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CreateAuthorizationPolicies(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CreateAuthorizationPolicies", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedCustomResources(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, stopCh <-chan struct{}) error {
	t.Record("CleanupOrphanedCustomResources", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"stopCh":                             stopCh})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedServiceEntries(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedServiceEntries", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedIngresses(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedIngresses", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedClusterRoleBindings", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedClusterRoles(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedClusterRoles", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedConfigMaps(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedConfigMaps", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, allvzSynMBs map[string]*types.SyntheticModelBinding) error {
	t.Record("CleanupOrphanedNamespaces", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"allvzSynMBs":                        allvzSynMBs})
	return nil
}

func (t *testManagedPackage) DeleteCustomResources(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("DeleteCustomResources", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) DeleteClusterRoleBindings(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	t.Record("DeleteClusterRoleBindings", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"bindingLabel":                       bindingLabel})
	return nil
}

func (t *testManagedPackage) DeleteClusterRoles(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	t.Record("DeleteClusterRoles", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"bindingLabel":                       bindingLabel})
	return nil
}

func (t *testManagedPackage) DeleteNamespaces(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, bindingLabel bool) error {
	t.Record("DeleteNamespaces", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections,
		"bindingLabel":                       bindingLabel})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedServices(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedServices", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) CleanupOrphanedDeployments(vzSynMB *types.SyntheticModelBinding, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("CleanupOrphanedDeployments", map[string]interface{}{
		"vzSynMB":                            vzSynMB,
		"availableManagedClusterConnections": availableManagedClusterConnections})
	return nil
}

func (t *testManagedPackage) DeleteServices(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("DeleteServices", map[string]interface{}{
		"vzSynMB":             vzSynMB,
		"filteredConnections": filteredConnections})
	return nil
}

func (t *testManagedPackage) DeleteDeployments(vzSynMB *types.SyntheticModelBinding, filteredConnections map[string]*util.ManagedClusterConnection) error {
	t.Record("DeleteDeployments", map[string]interface{}{
		"vzSynMB":             vzSynMB,
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

func (l *testLocalPackage) DeleteVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {
	l.Record("DeleteVmi", map[string]interface{}{
		"binding":      binding,
		"vmoClientSet": vmoClientSet,
		"vmiLister":    vmiLister})

	return nil
}

func (l *testLocalPackage) DeleteSecrets(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister) error {
	l.Record("DeleteSecrets", map[string]interface{}{
		"binding":       binding,
		"kubeClientSet": kubeClientSet,
		"secretLister":  secretLister})

	return nil
}

func (l *testLocalPackage) DeleteConfigMaps(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	l.Record("DeleteConfigMaps", map[string]interface{}{
		"binding":         binding,
		"kubeClientSet":   kubeClientSet,
		"configMapLister": configMapLister})

	return nil
}

func (l *testLocalPackage) CreateUpdateVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error {
	l.Record("CreateUpdateVmi", map[string]interface{}{
		"binding":                 binding,
		"vmoClientSet":            vmoClientSet,
		"vmiLister":               vmiLister,
		"verrazzanoURI":           verrazzanoURI,
		"enableMonitoringStorage": enableMonitoringStorage})
	return nil
}

func (l *testLocalPackage) UpdateConfigMaps(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	l.Record("UpdateConfigMaps", map[string]interface{}{
		"binding":         binding,
		"kubeClientSet":   kubeClientSet,
		"configMapLister": configMapLister})
	return nil
}

func (l *testLocalPackage) UpdateAcmeDNSSecret(binding *types.SyntheticBinding, kubeClientSet kubernetes.Interface, secretLister corev1listers.SecretLister, name string, verrazzanoURI string) error {
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

func (m *testMonitoringPackage) CreateVmiSecrets(binding *types.SyntheticBinding, secrets monitoring.Secrets) error {
	m.Record("CreateVmiSecrets", map[string]interface{}{
		"binding": binding,
		"secrets": secrets})

	return nil
}

func testExecuteCreateUpdateGlobaEntitiesGoroutine(binding *types.SyntheticBinding, c *Controller) {
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
