// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package genericcomp

import (
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/testutil"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
)

var generic = v1beta1.VerrazzanoGenericComponent{
	Name: "test-generic",
	Deployment: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "test-container-name",
				Image: "test-image",
				Ports: []corev1.ContainerPort{
					{
						Name:          "test-port",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: 8095,
					},
				},
				Env: []corev1.EnvVar{
					{
						Name:  "test-env-name1",
						Value: "test-env-value",
					},
					{
						Name: "test-env-name2",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								Key: "password",
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret-ref1",
								},
							},
						},
					},
				},
			},
		},
		InitContainers: []corev1.Container{
			{
				Name: "test-container-name",
				Command: []string{
					"test-command",
					"--test-arg",
				},
				Env: []corev1.EnvVar{
					{
						Name: "test-env-name2",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								Key: "password",
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret-ref2",
								},
							},
						},
					},
				},
			},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test-secret1",
			},
		},
	},
}

// TestNewDeployment tests that a deployment is created
// GIVEN a generic component definition
//  WHEN I call NewDeployment
//  THEN the expected generic component deployment should be created
func TestNewDeployment(t *testing.T) {
	assert := assert.New(t)

	labels := map[string]string{
		constants.K8SAppLabel:       constants.VerrazzanoGroup,
		constants.VerrazzanoBinding: "test-binding",
		constants.VerrazzanoCluster: "cluster1",
	}
	matchLabels := map[string]string{
		GenericComponentSelectorLabel: generic.Name,
	}

	generic.Replicas = util.NewVal(2)

	deploy := NewDeployment(generic, "test-namespace", labels)
	assert.Equal("test-generic", deploy.Name, "deployment name not equal to expected value")
	assert.Equal("test-namespace", deploy.Namespace, "deployment namespace not equal to expected value")
	assert.Equal(labels, deploy.Labels, "deployment labels not equal to expected value")
	assert.Equal(int32(2), *deploy.Spec.Replicas, "deployment replicas not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Selector.MatchLabels, "deployment selector match labels not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Template.Labels, "deployment template labels not equal to expected value")
	assert.Equal(deploy.Spec.Template.Spec, generic.Deployment, "deployment temp spec not equal to expected value")

	// Unset the replica value to make sure default value of 1 is used.
	generic.Replicas = nil

	deploy = NewDeployment(generic, "test-namespace", labels)
	assert.Equal("test-generic", deploy.Name, "deployment name not equal to expected value")
	assert.Equal("test-namespace", deploy.Namespace, "deployment namespace not equal to expected value")
	assert.Equal(labels, deploy.Labels, "deployment labels not equal to expected value")
	assert.Equal(int32(1), *deploy.Spec.Replicas, "deployment replicas not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Selector.MatchLabels, "deployment selector match labels not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Template.Labels, "deployment template labels not equal to expected value")
	assert.Equal(deploy.Spec.Template.Spec, generic.Deployment, "deployment temp spec not equal to expected value")
}

// TestNewService tests that a service is created
// GIVEN a generic component definition
//  WHEN I call NewService
//  THEN the expected generic component service should be created
func TestNewService(t *testing.T) {
	assert := assert.New(t)

	labels := map[string]string{
		constants.K8SAppLabel:       constants.VerrazzanoGroup,
		constants.VerrazzanoBinding: "test-binding",
		constants.VerrazzanoCluster: "cluster1",
	}

	selector := map[string]string{
		GenericComponentSelectorLabel: generic.Name,
	}

	service := NewService(generic, "test-namespace", labels)
	assert.Equal("test-generic", service.Name, "service name not equal to expected value")
	assert.Equal("test-namespace", service.Namespace, "service namespace not equal to expected value")
	assert.Equal(labels, service.Labels, "service labels not equal to expected value")
	assert.Equal(selector, service.Spec.Selector, "service selector not equal to expected value")
	assert.Equal(corev1.ServiceTypeClusterIP, service.Spec.Type, "service type not equal to expected value")
	assert.Equal(1, len(service.Spec.Ports), "service port list length not equal to expected value")
	assert.Equal("test-port", service.Spec.Ports[0].Name, "service port name not equal to expected value")
	assert.Equal(int32(8095), service.Spec.Ports[0].Port, "service port name not equal to expected value")
	assert.Equal(corev1.ProtocolTCP, service.Spec.Ports[0].Protocol, "service port name not equal to expected value")

	// No container ports specified then no new service is needed.
	generic.Deployment.Containers[0].Ports = []corev1.ContainerPort{}
	service = NewService(generic, "test-namespace", labels)
	assert.Nil(service, "service should be nil")
}

// TestGetSecrets tests that a list of secrets is returned
// GIVEN a deployment specification
//  WHEN I call GetSecrets
//  THEN the secrets referenced by the deployment are returned
func TestGetSecrets(t *testing.T) {
	assert := assert.New(t)

	labels := map[string]string{
		constants.K8SAppLabel:       constants.VerrazzanoGroup,
		constants.VerrazzanoBinding: "test-binding",
		constants.VerrazzanoCluster: "cluster1",
	}

	deploy := NewDeployment(generic, "test-namespace", labels)

	secrets := GetSecrets(*deploy)
	assert.Equal(3, len(secrets), "secrets list length not equal to expected value")
	assert.Equal("test-secret1", secrets[0], "secret not equal to expected value")
	assert.Equal("test-secret-ref1", secrets[1], "secret not equal to expected value")
	assert.Equal("test-secret-ref2", secrets[2], "secret not equal to expected value")
}

// TestAddEnvVars tests env variables are added to a generic component deployment
// GIVEN a managed cluster struct, a generic component name, and a list of environment variables to add
//  WHEN I call AddEnvVars
//  THEN the environment variables are added to a generic component deployment
func TestAddEnvVars(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	mc := modelBindingPair.ManagedClusters["cluster1"]

	envs := []corev1.EnvVar{
		{
			Name:  "test_env1",
			Value: "test_value1",
		},
		{
			Name:  "test_env2",
			Value: "test_value2",
		},
		// environment variable that already exist - will be replaced
		{
			Name:  "test_env1",
			Value: "test_value3",
		},
	}

	AddEnvVars(mc, "test-generic", &envs)
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.Containers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal("test_env1", mc.Deployments[0].Spec.Template.Spec.Containers[0].Env[0].Name, "environment variable name not equal to expected value")
	assert.Equal("test_value3", mc.Deployments[0].Spec.Template.Spec.Containers[0].Env[0].Value, "environment variable value not equal to expected value")
	assert.Equal("test_env2", mc.Deployments[0].Spec.Template.Spec.Containers[0].Env[1].Name, "environment variable name not equal to expected value")
	assert.Equal("test_value2", mc.Deployments[0].Spec.Template.Spec.Containers[0].Env[1].Value, "environment variable value not equal to expected value")

	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal("test_env1", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[0].Name, "environment variable name not equal to expected value")
	assert.Equal("test_value3", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[0].Value, "environment variable value not equal to expected value")
	assert.Equal("test_env2", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[1].Name, "environment variable name not equal to expected value")
	assert.Equal("test_value2", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[1].Value, "environment variable value not equal to expected value")

	// Specify nil for env vars - should be no change
	AddEnvVars(mc, "test-generic", nil)
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.Containers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env), "environment variable list length not equal to expected value")

	// Specify empty env vars array - should be no change
	AddEnvVars(mc, "test-generic", &[]corev1.EnvVar{})
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.Containers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env), "environment variable list length not equal to expected value")

}
