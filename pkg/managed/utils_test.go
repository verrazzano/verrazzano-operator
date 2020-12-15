// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	cohoprclientset "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/client/clientset/versioned"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	clientsetfake "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
	cohcluclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned"
	cohcluclientsetfake "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/clientset/versioned/fake"
	domclientset "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned"
	domclientsetfake "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/clientset/versioned/fake"
	helidionclientset "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/client/clientset/versioned"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	wlsoprclientset "github.com/verrazzano/verrazzano-wko-operator/pkg/client/clientset/versioned"
	istioauthclientset "istio.io/client-go/pkg/clientset/versioned"
	istioauthclientsetfake "istio.io/client-go/pkg/clientset/versioned/fake"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extclientsetfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// TestBuildManagedClusterConnection test the creation of a managed cluster connection
// GIVEN a fake kubeconfig contents and test rest.Config
//  WHEN I call BuildManagedClusterConnection
//  THEN the cluster connection client sets should be initialized with the test rest.Config
//   AND the cluster connection listers and informers should be initialized
func TestBuildManagedClusterConnection(t *testing.T) {
	assert := asserts.New(t)
	kubeConfigContents := []byte("test kubecfg contents")
	testConfig := &rest.Config{
		Host:            "testHost",
		APIPath:         "api",
		ContentConfig:   rest.ContentConfig{},
		Impersonate:     rest.ImpersonationConfig{},
		TLSClientConfig: rest.TLSClientConfig{},
	}

	// mock kubernetes client set creation to return fake
	origNewKubernetesClientSet := newKubernetesClientSet
	newKubernetesClientSet = func(c *rest.Config) (kubernetes.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return fake.NewSimpleClientset(), nil
	}
	defer func() { newKubernetesClientSet = origNewKubernetesClientSet }()

	// mock verrazzano operator client set creation to return fake
	origNewVerrazzanoOperatorClientSet := newVerrazzanoOperatorClientSet
	newVerrazzanoOperatorClientSet = func(c *rest.Config) (clientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return clientsetfake.NewSimpleClientset(), nil
	}
	defer func() { newVerrazzanoOperatorClientSet = origNewVerrazzanoOperatorClientSet }()

	// mock wls operator client set creation to return fake
	origNewWLSOperatorClientSet := newWLSOperatorClientSet
	newWLSOperatorClientSet = func(c *rest.Config) (wlsoprclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return testutil.NewWlsOprClientset(), nil
	}
	defer func() { newWLSOperatorClientSet = origNewWLSOperatorClientSet }()

	// mock domain client set creation to return fake
	origNewDomainClientSet := newDomainClientSet
	newDomainClientSet = func(c *rest.Config) (domclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return domclientsetfake.NewSimpleClientset(), nil
	}
	defer func() { newDomainClientSet = origNewDomainClientSet }()

	// mock helidon client set creation to return fake
	origNewHelidonClientSet := newHelidonClientSet
	newHelidonClientSet = func(c *rest.Config) (helidionclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return testutil.NewHelidionClientset(), nil
	}
	defer func() { newHelidonClientSet = origNewHelidonClientSet }()

	// mock Coherence operator client set creation to return fake
	origNewCOHOperatorClientSet := newCOHOperatorClientSet
	newCOHOperatorClientSet = func(c *rest.Config) (cohoprclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return testutil.NewCohOprClientset(), nil
	}
	defer func() { newCOHOperatorClientSet = origNewCOHOperatorClientSet }()

	// mock Coherence cluster client set creation to return fake
	origNewCOHClusterClientSet := newCOHClusterClientSet
	newCOHClusterClientSet = func(c *rest.Config) (cohcluclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return cohcluclientsetfake.NewSimpleClientset(), nil
	}
	defer func() { newCOHClusterClientSet = origNewCOHClusterClientSet }()

	// mock Istio client set creation to return fake
	origNewIstioClientSet := newIstioClientSet
	newIstioClientSet = func(c *rest.Config) (istioauthclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return istioauthclientsetfake.NewSimpleClientset(), nil
	}
	defer func() { newIstioClientSet = origNewIstioClientSet }()

	// mock Istio auth client set creation to return fake
	origNewIstioAuthClientSet := newIstioAuthClientSet
	newIstioAuthClientSet = func(c *rest.Config) (istioauthclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return istioauthclientsetfake.NewSimpleClientset(), nil
	}
	defer func() { newIstioAuthClientSet = origNewIstioAuthClientSet }()

	// mock kubernetes ext client set creation to return fake
	origNewExtClientSet := newExtClientSet
	newExtClientSet = func(c *rest.Config) (extclientset.Interface, error) {
		assert.Equal(testConfig, c, "didn't get the expected config")
		return extclientsetfake.NewSimpleClientset(), nil
	}
	defer func() { newExtClientSet = origNewExtClientSet }()

	// mock clientcmd.BuildConfigFromFlags function
	oldBuildConfigFromFlags := buildConfigFromFlags
	buildConfigFromFlags = func(masterUrl, kubeconfigPath string) (config *rest.Config, err error) {
		return testConfig, nil
	}
	defer func() { buildConfigFromFlags = oldBuildConfigFromFlags }()

	// mock os.remove function
	oldOsRemove := osRemove
	osRemove = func(name string) error {
		return nil
	}
	defer func() { osRemove = oldOsRemove }()

	// mock ioutil functions
	oldIoWriteFile := ioWriteFile
	ioWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		return nil
	}
	defer func() { ioWriteFile = oldIoWriteFile }()

	oldCreateKubeconfig := createKubeconfig
	createKubeconfig = func() (string, error) {
		return "kubeconfig9999", nil
	}
	defer func() { createKubeconfig = oldCreateKubeconfig }()

	clusterConnection, err := BuildManagedClusterConnection(kubeConfigContents, make(chan struct{}))
	assert.NoError(err, "got error from BuildManagedClusterConnection")

	// assert that all client sets have been initialized
	assert.NotNil(clusterConnection.KubeClient, "expected client set to be initialized")
	assert.NotNil(clusterConnection.KubeExtClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.VerrazzanoOperatorClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.WlsOprClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.DomainClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.HelidonClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.CohOprClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.CohClusterClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.IstioClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.IstioAuthClientSet, "expected client set to be initialized")

	// assert that all listers and informers have been initialized
	assert.NotNil(clusterConnection.DeploymentLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.DeploymentInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.PodLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.PodInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.ServiceAccountLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ServiceAccountInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.NamespaceLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.NamespaceInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.SecretLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.SecretInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.ClusterRoleLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ClusterRoleInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.ClusterRoleBindingLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ClusterRoleBindingInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.WlsOperatorLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.WlsOperatorInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.HelidonLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.HelidonInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.CohOperatorLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.CohOperatorInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.IstioGatewayLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.IstioGatewayInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.IstioVirtualServiceLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.IstioVirtualServiceInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.IstioServiceEntryLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.IstioServiceEntryInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.ConfigMapLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ConfigMapInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.DaemonSetLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.DaemonSetInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.ServiceLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ServiceInformer, "expected informer to be initialized")
}

// TestGetFilteredConnections test getting a filtered set of connections based on a given VerrazzanoBinding.
// GIVEN a VerrazzanoBinding and map of managed cluster connections
//  WHEN I call GetFilteredConnections
//  THEN a filtered map of connections (applicable to the given VerrazzanoBinding) should be returned
func TestGetFilteredConnections(t *testing.T) {
	assert := asserts.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()

	connections, err := GetFilteredConnections(modelBindingPair, clusterConnections)
	assert.NoError(err, "got error from BuildManagedClusterConnection")
	expectedClusterNames := map[string]struct{}{"cluster1": {}, "cluster2": {}}
	assert.Len(connections, len(expectedClusterNames), "expected 2 connections")
	for name := range connections {
		assert.Contains(expectedClusterNames, name, "connection %s not expected", name)
	}
}

// Test_setupHTTPResolve test setting nup http resolve for a given restConfig.
// GIVEN a restConfig and env var "rancherURL" and "rancherHost"
//  WHEN I call setupHTTPResolve
//  THEN the restConfig should setup Dial correctly depending on the URL and host
func Test_setupHTTPResolve(t *testing.T) {
	type args struct {
		url  string
		host string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "no_url",
			args:    args{url: "", host: ""},
			wantErr: false,
		},
		{
			name:    "wrong_url",
			args:    args{url: ":wrong_url", host: "172.17.0.2"},
			wantErr: true,
		},
		{
			name:    "no_host",
			args:    args{url: "https://127.0.0.1", host: ""},
			wantErr: false,
		},
		{
			name:    "same_host",
			args:    args{url: "https://127.0.0.1", host: "127.0.0.1"},
			wantErr: false,
		},
		{
			name:    "different_host",
			args:    args{url: "https://127.0.0.1", host: "172.17.0.2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("rancherURL", tt.args.url)
			os.Setenv("rancherHost", tt.args.host)
			config := &rest.Config{}
			if err := setupHTTPResolve(config); (err != nil) != tt.wantErr {
				t.Errorf("setupHTTPResolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
