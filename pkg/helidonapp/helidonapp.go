// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonapp

import (
	"fmt"

	"github.com/verrazzano/verrazzano-operator/pkg/types"

	"github.com/golang/glog"
	v1helidonapp "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const microOperatorName = "verrazzano-helidon-app-operator"
const DefaultPort = int32(8080)
const DefaultTargetPort = int32(8080)

func CreateHelidonAppCR(mcName string, namespace string, app *v1beta1v8o.VerrazzanoHelidon, mbPair *types.ModelBindingPair, labels map[string]string) *v1helidonapp.HelidonApp {
	helidonApp := v1helidonapp.HelidonApp{}

	helidonApp.TypeMeta.Kind = "HelidonApp"
	helidonApp.TypeMeta.APIVersion = "verrazzano.io/v1beta1"

	helidonApp.ObjectMeta.Name = app.Name
	helidonApp.ObjectMeta.Namespace = namespace
	helidonApp.ObjectMeta.Labels = labels

	helidonApp.Spec.Description = fmt.Sprintf("Helidon application for managed cluster %s", mcName)
	helidonApp.Spec.Name = app.Name
	helidonApp.Spec.Namespace = namespace
	helidonApp.Spec.Image = app.Image

	if len(app.ImagePullSecrets) != 0 {
		helidonApp.Spec.ImagePullSecrets = app.ImagePullSecrets
	}
	helidonApp.Spec.Port = DefaultPort
	helidonApp.Spec.TargetPort = DefaultTargetPort
	if app.Port > 0 {
		helidonApp.Spec.Port = int32(app.Port)
	}
	if app.TargetPort > 0 {
		helidonApp.Spec.TargetPort = int32(app.TargetPort)
	}

	// Get the Helidon binding and set replicas
	if mbPair.Binding.Spec.HelidonBindings != nil {
		for _, binding := range mbPair.Binding.Spec.HelidonBindings {
			if binding.Name == app.Name {
				helidonApp.Spec.Replicas = binding.Replicas
				break
			}
		}
	}

	// Add the ENV vars specified in the model file
	var envs []corev1.EnvVar
	var envSet = make(map[string]bool)
	for _, v := range app.Envs{
		e := corev1.EnvVar{
			Name:      v.Name,
			Value:     v.Value,
		}
		envSet[e.Name] = true
		envs = append(envs, e)
	}

	// Set the default Coherence related ENV vars, only if they are not
	// explicitly set in the model file
	for _, connection := range app.Connections {
		if connection.Coherence != nil {
			if len(connection.Coherence) > 1 {
				glog.Errorf("Only one Coherence binding allowed for a Helidon application '%s'. Found %d", app.Name, len(connection.Coherence))
			}
			for _, cohConnection := range connection.Coherence {
				// Get the Coherence cluster
				var cohCluster *v1beta1v8o.VerrazzanoCoherenceCluster
				for _, cluster := range mbPair.Model.Spec.CoherenceClusters {
					if cluster.Name == cohConnection.Target {
						cohCluster = &cluster
						break
					}
				}
				// Add the environment variables for the Coherence connection
				// Only override if the ENV var is not explicitly defined in the model
				if cohCluster != nil {
					var env corev1.EnvVar
					const COH_CLUSTER = "COH_CLUSTER"
					const COH_CACHE_CONFIG = "COH_CACHE_CONFIG"
					const COH_POF_CONFIG = "COH_POF_CONFIG"

					if _, ok := envSet[COH_CLUSTER]; !ok {
						env.Name = COH_CLUSTER
						env.Value = cohConnection.Target
						envs = append(envs, env)
					}
					if _, ok := envSet[COH_CACHE_CONFIG]; !ok {
						env.Name = COH_CACHE_CONFIG
						env.Value = cohCluster.CacheConfig
						envs = append(envs, env)

					}
					if _, ok := envSet[COH_POF_CONFIG]; !ok {
						env.Name = COH_POF_CONFIG
						env.Value = cohCluster.PofConfig
						envs = append(envs, env)
					}
				} else {
					glog.Errorf("Coherence binding '%s' not found in binding file", cohConnection.Target)
				}
			}
		}
	}

	if len(envs) != 0 {
		helidonApp.Spec.Env = envs
	}

	// Include fluentd needed resource if fluentd integration is enabled
	if IsFluentdEnabled(app) {
		// Add fluentd container
		helidonApp.Spec.Containers = append(helidonApp.Spec.Containers, createFluentdContainer(mbPair.Binding, app, mbPair.VerrazzanoUri, mbPair.SslVerify))

		// Add fluentd volumes
		volumes := createFluentdVolHostPaths()
		for _, volume := range *volumes {
			helidonApp.Spec.Volumes = append(helidonApp.Spec.Volumes, volume)
		}
		helidonApp.Spec.Volumes = append(helidonApp.Spec.Volumes, createFluentdVolConfigMap(app))
	}

	return &helidonApp
}

// Create a deployment for the verrazzano-helidon-app-operator
func CreateDeployment(namespace string, labels map[string]string, image string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      microOperatorName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: util.NewVal(1),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            microOperatorName,
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{microOperatorName},
							Env: []corev1.EnvVar{
								{
									Name:  "WATCH_NAMESPACE",
									Value: "",
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "OPERATOR_NAME",
									Value: microOperatorName,
								},
							},
						},
					},
					TerminationGracePeriodSeconds: util.New64Val(1),
					ServiceAccountName:            util.GetServiceAccountNameForSystem(),
				},
			},
		},
	}

	return deployment
}

// Update env variables for a given component
func UpdateEnvVars(mc *types.ManagedCluster, component string, envs *[]corev1.EnvVar) {
	if *envs != nil && len(*envs) != 0 {
		for _, app := range mc.HelidonApps {
			if app.Name == component {
				for _, env := range *envs {
					app.Spec.Env = append(app.Spec.Env, env)
				}
				return
			}
		}
	}
}

// Check if Fluentd integration is enabled for this application
func IsFluentdEnabled(app *v1beta1v8o.VerrazzanoHelidon) bool {
	if app.FluentdEnabled == nil {
		return true
	}

	return *app.FluentdEnabled
}
