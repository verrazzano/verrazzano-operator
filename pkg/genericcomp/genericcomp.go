// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package genericcomp

import (
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

// UpdateEnvVars updates env variables for a given generic component.
func UpdateEnvVars(mc *types.ManagedCluster, component string, envs *[]corev1.EnvVar) {
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
