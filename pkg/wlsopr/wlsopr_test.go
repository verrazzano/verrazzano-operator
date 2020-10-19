// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsopr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vz "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	wlsv1beta "github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var origGetEnvFunc = util.GetEnvFunc

var (
	emptyBinding = &vz.VerrazzanoBinding{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       vz.VerrazzanoBindingSpec{},
		Status:     vz.VerrazzanoBindingStatus{},
	}
)

// TestNewWlsOperatorCRMissingManagedClusterName tests the creation of a WebLogic micro operator custom resource
// GIVEN a configuration with no managed cluster name specified
//  WHEN I call NewWlsOperatorCR
//  THEN there should be an error returned that the managed cluster name is required
func TestNewWlsOperatorCRMissingManagedClusterName(t *testing.T) {
	crConfig := WlsOperatorCRConfig{
		BindingName:        "mytestbinding",
		BindingNamespace:   "verazzano-testbinding",
		DomainNamespace:    "default",
		Labels:             make(map[string]string),
		ManagedClusterName: "",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}
	_, err := NewWlsOperatorCR(crConfig)
	if assert.Error(t, err) {
		assert.Equal(t, errors.New("ManagedClusterName is required"), err)
	}
}

// TestNewWlsOperatorCRMissingDomainNamespace tests the creation of a WebLogic micro operator custom resource
// GIVEN a configuration with no domain namespace specified
//  WHEN I call NewWlsOperatorCR
//  THEN there should be an error returned that the domain namespace is required
func TestNewWlsOperatorCRMissingDomainNamespace(t *testing.T) {
	crConfig := WlsOperatorCRConfig{
		BindingName:        "mytestbinding",
		BindingNamespace:   "verazzano-testbinding",
		DomainNamespace:    "",
		Labels:             make(map[string]string),
		ManagedClusterName: "local",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}
	_, err := NewWlsOperatorCR(crConfig)
	if assert.Error(t, err) {
		assert.Equal(t, errors.New("DomainNamespace is required"), err)
	}
}

// TestNewWlsOperatorCRMissingBindingNamespace tests the creation of a WebLogic micro operator custom resource
// GIVEN a configuration with no binding namespace specified
//  WHEN I call NewWlsOperatorCR
//  THEN there should be an error returned that the binding namespace is required
func TestNewWlsOperatorCRMissingBindingNamespace(t *testing.T) {
	crConfig := WlsOperatorCRConfig{
		BindingName:        "mytestbinding",
		BindingNamespace:   "",
		DomainNamespace:    "default",
		Labels:             make(map[string]string),
		ManagedClusterName: "local",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}
	_, err := NewWlsOperatorCR(crConfig)
	if assert.Error(t, err) {
		assert.Equal(t, errors.New("BindingNamespace is required"), err)
	}
}

// TestNewWlsOperatorCR tests the creation of a WebLogic micro operator custom resource
// GIVEN a valid configuration
//  WHEN I call NewWlsOperatorCR
//  THEN there should be an expected WebLogic micro operator custom resource created
func TestNewWlsOperatorCR(t *testing.T) {
	assert := assert.New(t)
	crConfig := WlsOperatorCRConfig{
		BindingName:      "mytestbinding",
		BindingNamespace: "verazzano-testbinding",
		DomainNamespace:  "default",
		Labels: map[string]string{
			"mylabel": "mylabelvalue",
		},
		ManagedClusterName: "local",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}

	wlsOperatorSpec := wlsv1beta.WlsOperatorSpec{
		Description:      fmt.Sprintf("WebLogic Operator for managed cluster %s using binding %s", crConfig.ManagedClusterName, crConfig.BindingName),
		Name:             fmt.Sprintf("%s-%s", operatorName, crConfig.BindingName),
		Namespace:        crConfig.BindingNamespace,
		ServiceAccount:   operatorName,
		Image:            crConfig.WlsOperatorImage,
		ImagePullPolicy:  "IfNotPresent",
		DomainNamespaces: []string{crConfig.DomainNamespace},
	}
	wlsopr, err := NewWlsOperatorCR(crConfig)
	if assert.NoError(err) {
		assert.Equal("WlsOperator", wlsopr.TypeMeta.Kind)
		assert.Equal("verrazzano.io/v1beta1", wlsopr.TypeMeta.APIVersion)
		assert.Equal(operatorName, wlsopr.ObjectMeta.Name)
		assert.Equal(crConfig.BindingNamespace, wlsopr.ObjectMeta.Namespace)
		assert.Equal(crConfig.Labels, wlsopr.ObjectMeta.Labels)
		assert.Equal(wlsv1beta.WlsOperatorStatus{}, wlsopr.Status)
		assert.Equal(wlsOperatorSpec, wlsopr.Spec)
	}
}

// TestCreateDeployment tests the creation of a WebLogic micro operator
// GIVEN a namespace name, binding name, and image name
//  WHEN I call CreateDeployment
//  THEN there should be an expected WebLogic micro operator deployment created
func TestCreateDeployment(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	labels := util.GetManagedBindingLabels(modelBindingPair.Binding, "cluster1")

	// temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	deployment := CreateDeployment("test-namespace", "test-binding", labels, "wko-image")
	assert.Equal(microOperatorName, deployment.Name, "deployment name not equal to expected value")
	assert.Equal("test-namespace", deployment.Namespace, "deployment namespace not equal to expected value")
	assert.Equal(labels, deployment.Labels, "deployment labels not equal to expected value")
	assert.Equal(int32(1), *deployment.Spec.Replicas, "deployment namespace not equal to expected value")
	assert.Equal(appsv1.RecreateDeploymentStrategyType, deployment.Spec.Strategy.Type, "deployment strategy not equal to expected value")
	assert.Equal(labels, deployment.Spec.Selector.MatchLabels, "deployment selector match labels not equal to expected value")
	assert.Equal(labels, deployment.Spec.Template.Labels, "deployment template labels not equal to expected value")
	assert.Equal(0, len(deployment.Spec.Template.Spec.InitContainers), "deployment init container count not equal to expected value")
	assert.Equal(1, len(deployment.Spec.Template.Spec.Containers), "deployment container count not equal to expected value")
	container := deployment.Spec.Template.Spec.Containers[0]
	assert.Equal(microOperatorName, container.Name, "deployment container name not equal to expected value")
	assert.Equal("wko-image", container.Image, "deployment container image not equal to expected value")
	assert.Equal(corev1.PullIfNotPresent, container.ImagePullPolicy, "deployment container image pull policy not equal to expected value")
	assert.Equal("verrazzano-wko-operator", container.Command[0], "deployment container command not equal to expected value")
	assert.Equal(resource.MustParse("2.5Gi"), container.Resources.Requests["memory"], "deployment container memory request not equal to expected value")
	assert.Equal(3, len(container.Env), "deployment container env count not equal to expected value")
	assert.Equal("WATCH_NAMESPACE", container.Env[0].Name, "deployment container env name not equal to expected value")
	assert.Equal("", container.Env[0].Value, "deployment container env value not equal to expected value")
	assert.Equal("POD_NAME", container.Env[1].Name, "deployment container env name not equal to expected value")
	assert.Equal("metadata.name", container.Env[1].ValueFrom.FieldRef.FieldPath, "deployment container env value not equal to expected value")
	assert.Equal("OPERATOR_NAME", container.Env[2].Name, "deployment container env name not equal to expected value")
	assert.Equal(microOperatorName, container.Env[2].Value, "deployment container env value not equal to expected value")
	assert.Equal(util.New64Val(1), deployment.Spec.Template.Spec.TerminationGracePeriodSeconds, "deployment termination grace period not equal to expected value")
	assert.Equal(util.GetServiceAccountNameForSystem(), deployment.Spec.Template.Spec.ServiceAccountName, "deployment service account name not equal to expected value")
}

// getenv returns a mocked response for keys used by these tests
func getenv(key string) string {
	if key == "WLS_MICRO_REQUEST_MEMORY" {
		return "2.5Gi"
	}
	return origGetEnvFunc(key)
}
