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

const GenericComponentSelectorLabel = "verrazzano.name"

// NewDeployment constructs a deployment for a generic component.
func NewDeployment(generic v1beta1v8o.VerrazzanoGenericComponent, namespace string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
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
}

// NewService constructs a service for a generic component.
func NewService(generic v1beta1v8o.VerrazzanoGenericComponent, namespace string, labels map[string]string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generic.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				GenericComponentSelectorLabel: generic.Name,
			},
			Ports: []corev1.ServicePort{},
			Type:  corev1.ServiceTypeClusterIP,
		},
	}

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

// AddEnvVars adds environment variables to a generic components deployment container.
func AddEnvVars(mc *types.ManagedCluster, component string, envs *[]corev1.EnvVar) {
	if *envs != nil && len(*envs) != 0 {
		for _, generic := range mc.GenericComponents {
			if generic.Name == component {
				for _, deployment := range mc.Deployments {
					if deployment.Name == component {
						for index, container := range deployment.Spec.Template.Spec.Containers {
							for _, env := range *envs {
								found := false
								for _, cenv := range container.Env {
									if cenv.Name == env.Name {
										found = true
									}
								}
								// Only add env variable to container if the env variable does not already exist.
								if !found {
									container.Env = append(container.Env, env)
								}
							}
							deployment.Spec.Template.Spec.Containers[index] = *container.DeepCopy()
						}
						return
					}
				}
			}
		}
	}
}
