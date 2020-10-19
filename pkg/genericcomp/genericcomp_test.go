// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package genericcomp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

var origGetEnvFunc = util.GetEnvFunc

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
		constants.ServiceAppLabel:   "test-generic",
	}
	matchLabels := map[string]string{
		GenericComponentSelectorLabel: generic.Name,
	}

	// Temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	generic.Replicas = util.NewVal(2)

	deploy := NewDeployment(generic, "test-binding", "test-namespace", labels)
	assert.Equal("test-generic", deploy.Name, "deployment name not equal to expected value")
	assert.Equal("test-namespace", deploy.Namespace, "deployment namespace not equal to expected value")
	assert.Equal(labels, deploy.Labels, "deployment labels not equal to expected value")
	assert.Equal(int32(2), *deploy.Spec.Replicas, "deployment replicas not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Selector.MatchLabels, "deployment selector match labels not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Template.Labels, "deployment template labels not equal to expected value")
	assert.Equal(2, len(deploy.Spec.Template.Spec.Containers), "deployment container count not equal to expected value")
	assert.Equal(generic.Deployment.Containers[0], deploy.Spec.Template.Spec.Containers[0], "deployment container not equal to expected value")
	assert.Equal(generic.Deployment.InitContainers[0], deploy.Spec.Template.Spec.InitContainers[0], "deployment init container not equal to expected value")

	// Check the Fluentd container that is created
	fluentd := deploy.Spec.Template.Spec.Containers[1]
	assert.Equal("fluentd", fluentd.Name, "Fluentd container name not equal to expected value")
	assert.Equal(getenv("FLUENTD_IMAGE"), fluentd.Image, "Fluentd container name not equal to expected value")
	assert.Equal(2, len(fluentd.Args), "Fluentd container args count not equal to expected value")
	assert.Equal("-c", fluentd.Args[0], "Fluentd container arg not equal to expected value")
	assert.Equal("/etc/fluent.conf", fluentd.Args[1], "Fluentd container arg not equal to expected value")
	assert.Equal(7, len(fluentd.Env), "Fluentd container envs count not equal to expected value")
	assert.Equal("APPLICATION_NAME", fluentd.Env[0].Name, "Fluentd container envs name not equal to expected value")
	assert.Equal("test-generic", fluentd.Env[0].Value, "Fluentd container envs value not equal to expected value")
	assert.Equal("FLUENTD_CONF", fluentd.Env[1].Name, "Fluentd container envs name not equal to expected value")
	assert.Equal("fluentd.conf", fluentd.Env[1].Value, "Fluentd container envs value not equal to expected value")
	assert.Equal("FLUENT_ELASTICSEARCH_SED_DISABLE", fluentd.Env[2].Name, "Fluentd container envs name not equal to expected value")
	assert.Equal("true", fluentd.Env[2].Value, "Fluentd container envs value not equal to expected value")
	assert.Equal("ELASTICSEARCH_HOST", fluentd.Env[3].Name, "Fluentd container envs name not equal to expected value")
	assert.Equal("vmi-test-binding-es-ingest.verrazzano-system.svc.cluster.local", fluentd.Env[3].Value, "Fluentd container envs value not equal to expected value")
	assert.Equal("ELASTICSEARCH_PORT", fluentd.Env[4].Name, "Fluentd container envs name not equal to expected value")
	assert.Equal("9200", fluentd.Env[4].Value, "Fluentd container envs value not equal to expected value")
	assert.Equal("ELASTICSEARCH_USER", fluentd.Env[5].Name, "Fluentd container envs name not equal to expected value")
	assert.Equal(constants.VmiSecretName, fluentd.Env[5].ValueFrom.SecretKeyRef.Name, "Fluentd container envs value from secret name not equal to expected value")
	assert.Equal("username", fluentd.Env[5].ValueFrom.SecretKeyRef.Key, "Fluentd container envs value from secret key not equal to expected value")
	assert.Equal(true, *fluentd.Env[5].ValueFrom.SecretKeyRef.Optional, "Fluentd container envs value from secret optional not equal to expected value")
	assert.Equal(constants.VmiSecretName, fluentd.Env[5].ValueFrom.SecretKeyRef.Name, "Fluentd container envs value from secret name not equal to expected value")
	assert.Equal("password", fluentd.Env[6].ValueFrom.SecretKeyRef.Key, "Fluentd container envs value from secret key not equal to expected value")
	assert.Equal(true, *fluentd.Env[6].ValueFrom.SecretKeyRef.Optional, "Fluentd container envs value from secret optional not equal to expected value")
	assert.Equal(3, len(fluentd.VolumeMounts), "Fluentd container volume mounts count not equal to expected value")
	assert.Equal("/fluentd/etc/fluentd.conf", fluentd.VolumeMounts[0].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assert.Equal("fluentd-config-volume", fluentd.VolumeMounts[0].Name, "Fluentd container volume mounts name not equal to expected value")
	assert.Equal("fluentd.conf", fluentd.VolumeMounts[0].SubPath, "Fluentd container volume mounts sub path not equal to expected value")
	assert.Equal(true, fluentd.VolumeMounts[0].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
	assert.Equal("/var/log", fluentd.VolumeMounts[1].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assert.Equal("varlog", fluentd.VolumeMounts[1].Name, "Fluentd container volume mounts name not equal to expected value")
	assert.Equal(true, fluentd.VolumeMounts[1].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
	assert.Equal("/u01/data/docker/containers", fluentd.VolumeMounts[2].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assert.Equal("datadockercontainers", fluentd.VolumeMounts[2].Name, "Fluentd container volume mounts name not equal to expected value")
	assert.Equal(true, fluentd.VolumeMounts[2].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")

	// Check for volumes created for Fluentd
	volumes := deploy.Spec.Template.Spec.Volumes
	assert.Equal(3, len(volumes), "Fluentd container volumes count not equal to expected value")
	assert.Equal("varlog", volumes[0].Name, "Fluentd volume name not equal to expected value")
	assert.Equal("/var/log", volumes[0].VolumeSource.HostPath.Path, "Fluentd volume host path not equal to expected value")
	assert.Equal("datadockercontainers", volumes[1].Name, "Fluentd volume name not equal to expected value")
	assert.Equal("/u01/data/docker/containers", volumes[1].VolumeSource.HostPath.Path, "Fluentd volume host path not equal to expected value")
	assert.Equal("fluentd-config-volume", volumes[2].Name, "Fluentd volume name not equal to expected value")
	assert.Equal("test-generic-fluentd", volumes[2].VolumeSource.ConfigMap.Name, "Fluentd volume config map name not equal to expected value")
	assert.Equal(int32(420), *volumes[2].VolumeSource.ConfigMap.DefaultMode, "Fluentd volume config map default mode not equal to expected value")

	// Unset the replica value to make sure default value of 1 is used.
	generic.Replicas = nil

	// Disable Fluentd and make sure Fluentd container is not included in deployment.
	fluentdEnabled := false
	generic.FluentdEnabled = &fluentdEnabled

	deploy = NewDeployment(generic, "test-binding", "test-namespace", labels)
	assert.Equal("test-generic", deploy.Name, "deployment name not equal to expected value")
	assert.Equal("test-namespace", deploy.Namespace, "deployment namespace not equal to expected value")
	assert.Equal(labels, deploy.Labels, "deployment labels not equal to expected value")
	assert.Equal(int32(1), *deploy.Spec.Replicas, "deployment replicas not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Selector.MatchLabels, "deployment selector match labels not equal to expected value")
	assert.Equal(matchLabels, deploy.Spec.Template.Labels, "deployment template labels not equal to expected value")
	assert.Equal(1, len(deploy.Spec.Template.Spec.Containers), "deployment container count not equal to expected value")
	assert.Equal(generic.Deployment.Containers[0], deploy.Spec.Template.Spec.Containers[0], "deployment container not equal to expected value")
	assert.Equal(generic.Deployment.InitContainers[0], deploy.Spec.Template.Spec.InitContainers[0], "deployment init container not equal to expected value")
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

	// Make sure fluentd is enabled so additional secrets are created.
	fluentdEnabled := true
	generic.FluentdEnabled = &fluentdEnabled
	generic.Deployment.Containers[0].Ports = []corev1.ContainerPort{
		{
			Name:          "test-port",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8095,
		},
	}

	deploy := NewDeployment(generic, "test-binding", "test-namespace", labels)

	secrets := GetSecrets(*deploy)
	assert.Equal(5, len(secrets), "secrets list length not equal to expected value")
	assert.Equal("test-secret1", secrets[0], "secret not equal to expected value")
	assert.Equal("test-secret-ref1", secrets[1], "secret not equal to expected value")
	assert.Equal(constants.VmiSecretName, secrets[2], "secret not equal to expected value")
	assert.Equal(constants.VmiSecretName, secrets[3], "secret not equal to expected value")
	assert.Equal("test-secret-ref2", secrets[4], "secret not equal to expected value")
}

// TestUpdateEnvVars tests env variables are added to a generic component deployment
// GIVEN a managed cluster struct, a generic component name, and a list of environment variables to add
//  WHEN I call UpdateEnvVars
//  THEN the environment variables are added to a generic component deployment
func TestUpdateEnvVars(t *testing.T) {
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
		// environment variable that already exist - will be replaced because it has a different value
		{
			Name:  "test_env1",
			Value: "test_value3",
		},
	}

	UpdateEnvVars(mc, "test-generic", &envs)
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
	UpdateEnvVars(mc, "test-generic", nil)
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.Containers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env), "environment variable list length not equal to expected value")

	// Specify empty env vars array - should be no change
	UpdateEnvVars(mc, "test-generic", &[]corev1.EnvVar{})
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.Containers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env), "environment variable list length not equal to expected value")

	envs = []corev1.EnvVar{
		{
			Name:  "test_env1",
			Value: "test_value1",
		},
		{
			Name:  "test_env2",
			Value: "test_value2",
		},
		// environment variable that already exist - will not be replaced because value is the same
		{
			Name:  "test_env1",
			Value: "test_value1",
		},
	}

	UpdateEnvVars(mc, "test-generic", &envs)
	assert.Equal(2, len(mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env), "environment variable list length not equal to expected value")
	assert.Equal("test_env1", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[0].Name, "environment variable name not equal to expected value")
	assert.Equal("test_value1", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[0].Value, "environment variable value not equal to expected value")
	assert.Equal("test_env2", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[1].Name, "environment variable name not equal to expected value")
	assert.Equal("test_value2", mc.Deployments[0].Spec.Template.Spec.InitContainers[0].Env[1].Value, "environment variable value not equal to expected value")
}

// getenv returns a mocked response for keys used by these tests
func getenv(key string) string {
	if key == "FLUENTD_IMAGE" {
		return "fluentd-kubernetes-daemonset:v1.10.4-7f37ac6-20"
	}
	return origGetEnvFunc(key)
}
