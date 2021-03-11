// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	clientsetfake "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned/fake"
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

	clusterConnection, err := BuildManagedClusterConnection("", make(chan struct{}))
	assert.NoError(err, "got error from BuildManagedClusterConnection")

	// assert that all client sets have been initialized
	assert.NotNil(clusterConnection.KubeClient, "expected client set to be initialized")
	assert.NotNil(clusterConnection.KubeExtClientSet, "expected client set to be initialized")
	assert.NotNil(clusterConnection.VerrazzanoOperatorClientSet, "expected client set to be initialized")

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
	assert.NotNil(clusterConnection.ConfigMapLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ConfigMapInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.DaemonSetLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.DaemonSetInformer, "expected informer to be initialized")
	assert.NotNil(clusterConnection.ServiceLister, "expected lister to be initialized")
	assert.NotNil(clusterConnection.ServiceInformer, "expected informer to be initialized")
}

// Test_createTempKubeconfigFile test creating temp file for storing kubeconfig.
//  WHEN I call createTempKubeconfigFile
//  THEN a temp file should be created and the name should be returned
func Test_createTempKubeconfigFile(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "postive_test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createTempKubeconfigFile()
			if (err != nil) != tt.wantErr {
				t.Errorf("createTempKubeconfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if _, err := os.Stat(got); os.IsNotExist(err) {
				t.Errorf("createTempKubeconfigFile() got %v, but it doesn't exist", got)
			}
		})
	}
}
