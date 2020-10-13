// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package genericcomp

import (
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenericComponentSelectorLabel defines the selector label for generic component resources.
const GenericComponentSelectorLabel = "verrazzano.name"

// NewDeployment constructs a deployment for a generic component.
func NewDeployment(generic v1beta1v8o.VerrazzanoGenericComponent, bindingName string, namespace string, labels map[string]string) *appsv1.Deployment {
	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generic.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 {
				if generic.Replicas == nil {
					return util.NewVal(1)
				}
				return generic.Replicas
			}(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					GenericComponentSelectorLabel: generic.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						GenericComponentSelectorLabel: generic.Name,
					},
				},
				Spec: generic.Deployment,
			},
		},
	}

	// Include Fluentd needed resource if Fluentd integration is enabled
	if IsFluentdEnabled(&generic) {
		// Add Fluentd container
		deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, createFluentdContainer(bindingName, generic.Name))

		// Add Fluentd volumes
		volumes := createFluentdVolHostPaths()
		for _, volume := range *volumes {
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)
		}
		deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, createFluentdVolConfigMap(generic.Name))
	}

	return &deploy
}

// NewService constructs a service for a generic component.
func NewService(generic v1beta1v8o.VerrazzanoGenericComponent, namespace string, labels map[string]string) *corev1.Service {
	service := &corev1.Service{}

	// Add all container ports to the k8s service
	for _, container := range generic.Deployment.Containers {
		for _, port := range container.Ports {
			sp := corev1.ServicePort{
				Name:     port.Name,
				Protocol: port.Protocol,
				Port:     port.ContainerPort,
			}
			service.Spec.Ports = append(service.Spec.Ports, sp)
		}
	}

	// No ports then no need for a service
	if len(service.Spec.Ports) == 0 {
		return nil
	}

	service.Name = generic.Name
	service.Namespace = namespace
	service.Labels = labels

	service.Spec.Selector = map[string]string{GenericComponentSelectorLabel: generic.Name}
	service.Spec.Type = corev1.ServiceTypeClusterIP

	return service
}

// GetSecrets returns the secrets required by a generic component deployment.
func GetSecrets(deploy appsv1.Deployment) []string {
	var secrets []string

	// Capture any image pull secrets
	for _, secret := range deploy.Spec.Template.Spec.ImagePullSecrets {
		secrets = append(secrets, secret.Name)
	}

	// Capture any secrets referenced in container env variables
	for _, container := range deploy.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secrets = append(secrets, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}

	// Capture any secrets referenced in init container env variables
	for _, container := range deploy.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secrets = append(secrets, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}

	return secrets
}

// UpdateEnvVars adds environment variables to a generic components deployment container.
func UpdateEnvVars(mc *types.ManagedCluster, component string, envs *[]corev1.EnvVar) {
	if envs == nil || len(*envs) == 0 {
		return
	}

	for _, generic := range mc.GenericComponents {
		if generic.Name == component {
			for _, deployment := range mc.Deployments {
				if deployment.Name == component {
					addContainerEnvs(envs, deployment.Spec.Template.Spec.Containers)
					addContainerEnvs(envs, deployment.Spec.Template.Spec.InitContainers)
					return
				}
			}
		}
	}
}

// Add environment variables to containers
func addContainerEnvs(envs *[]corev1.EnvVar, containers []corev1.Container) {
	for containerIndex := range containers {
		for _, env := range *envs {
			found := false
			for envIndex, cenv := range containers[containerIndex].Env {
				if cenv.Name == env.Name {
					containers[containerIndex].Env[envIndex].Value = env.Value
					found = true
				}
			}
			// Only add env variable to container if the env variable does not already exist.
			if !found {
				containers[containerIndex].Env = append(containers[containerIndex].Env, env)
			}
		}
	}
}

// IsFluentdEnabled checks if Fluentd integration is enabled for this generic component.
func IsFluentdEnabled(generic *v1beta1v8o.VerrazzanoGenericComponent) bool {
	if generic.FluentdEnabled == nil {
		return true
	}

	return *generic.FluentdEnabled
}
