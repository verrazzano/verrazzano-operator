// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsdom

import (
	"fmt"

	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v7weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v7"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateWlsDomainCR(namespace string, domainModel v1beta1v8o.VerrazzanoWebLogicDomain, mbPair *types.ModelBindingPair, labels map[string]string) *v7weblogic.Domain {
	domainCRValues := domainModel.DomainCRValues

	labels["weblogic.resourceVersion"] = "domain-v7"
	labels["weblogic.domainUID"] = domainModel.Name

	domainCR := &v7weblogic.Domain{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      domainModel.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v7weblogic.DomainSpec{
			DomainHome: domainCRValues.DomainHome,
			DomainUID: func() string {
				if len(domainCRValues.DomainUID) > 0 {
					return domainCRValues.DomainUID
				} else {
					return domainModel.Name
				}
			}(),
			Image: domainCRValues.Image,
			ImagePullPolicy: func() string {
				// ImagePullPolicy
				if len(domainCRValues.ImagePullPolicy) > 0 {
					return domainCRValues.ImagePullPolicy
				} else {
					return "IfNotPresent"
				}
			}(),
			ImagePullSecrets: domainCRValues.ImagePullSecrets,
			WebLogicCredentialsSecret: v7weblogic.WebLogicSecret{
				Name: domainCRValues.WebLogicCredentialsSecret.Name,
			},
			LogHome:                  fmt.Sprintf("/scratch/logs/%s", domainModel.Name),
			LogHomeEnabled:           true,
			Clusters:                 domainCRValues.Clusters,
			IncludeServerOutInPodLog: domainCRValues.IncludeServerOutInPodLog,
			DomainHomeInImage:        true,
			AdminServer: v7weblogic.AdminServer{
				Server: v7weblogic.Server{
					BaseConfiguration: v7weblogic.BaseConfiguration{
						ServerStartState: func() string {
							if len(domainCRValues.AdminServer.ServerStartState) > 0 {
								return domainCRValues.AdminServer.ServerStartState
							} else {
								return "RUNNING"
							}
						}(),
						RestartVersion: domainCRValues.AdminServer.RestartVersion,
						ServerService:  domainCRValues.AdminServer.ServerService,
					},
				},
				AdminService: v7weblogic.AdminService{
					Channels: func() []v7weblogic.Channel {
						var channels []v7weblogic.Channel

						if domainCRValues.AdminServer.AdminService.Channels != nil {
							channels = domainCRValues.AdminServer.AdminService.Channels
						} else {
							// Default Channel
							port := domainModel.AdminPort
							if port > 0 {
								channels = append(channels, v7weblogic.Channel{
									ChannelName: "istio-default",
									NodePort:    port,
								})
							}

							// T3 Channel?
							if domainModel.T3Port > 0 {
								channels = append(channels, v7weblogic.Channel{
									ChannelName: "T3Channel",
									NodePort:    domainModel.T3Port,
								})
							}
						}
						return channels
					}(),
				},
			},
			Replicas: func() *int32 {
				// Set replicas if specified in the model or the default of 1 if not specified
				if domainCRValues.Replicas != nil {
					return domainCRValues.Replicas
				} else {
					return util.NewVal(1)
				}
			}(),
			Configuration: v7weblogic.Configuration{
				Istio: v7weblogic.Istio{
					// Istio is always enabled for WebLogic domains in Verrazzano
					Enabled: true,
					ReadinessPort: 8888,
				},
			},
			ServerStartPolicy: func() string {
				if len(domainCRValues.ServerStartPolicy) > 0 {
					return domainCRValues.ServerStartPolicy
				} else {
					return "IF_NEEDED"
				}
			}(),
			BaseConfiguration: v7weblogic.BaseConfiguration{
				ServerPod: func() v7weblogic.ServerPod {
					serverPod := domainCRValues.ServerPod

					// Add fluentd config
					addFluentdConfig(&serverPod, domainModel, mbPair)

					// Provide default values for some env values
					javaOptionsFound := false
					userMemArgsFound := false
					for _, env := range serverPod.Env {
						if env.Name == "JAVA_OPTIONS" {
							javaOptionsFound = true
						} else if env.Name == "USER_MEM_ARGS" {
							userMemArgsFound = true
						}
					}
					if !javaOptionsFound {
						serverPod.Env = append(serverPod.Env, corev1.EnvVar{
							Name:  "JAVA_OPTIONS",
							Value: "-Dweblogic.StdoutDebugEnabled=false",
						})
					}
					if !userMemArgsFound {
						serverPod.Env = append(serverPod.Env, corev1.EnvVar{
							Name:  "USER_MEM_ARGS",
							Value: "-Djava.security.egd=file:/dev/./urandom -Xms64m -Xmx256m ",
						})
					}
					return serverPod
				}(),
				ServerService: domainCRValues.ServerService,
				ServerStartState: func() string {
					if len(domainCRValues.ServerStartState) > 0 {
						return domainCRValues.ServerStartState
					} else {
						return "RUNNING"
					}
				}(),
				RestartVersion: domainCRValues.RestartVersion,
			},
		},
		Status: v7weblogic.DomainStatus{},
	}

	// ConfigOverrides
	if len(domainCRValues.ConfigOverrides) > 0 {
		domainCR.Spec.ConfigOverrides = domainCRValues.ConfigOverrides
	}

	// ConfigOverrideSecrets
	if domainCRValues.ConfigOverrideSecrets != nil {
		domainCR.Spec.ConfigOverrideSecrets = domainCRValues.ConfigOverrideSecrets
	}

	// Process any overrides specified in the binding file
	binding := mbPair.Binding
	for _, wlsBinding := range binding.Spec.WeblogicBindings {
		if wlsBinding.Name == domainModel.Name {
			if wlsBinding.Replicas != nil {
				domainCR.Spec.Replicas = wlsBinding.Replicas
			}
			break
		}
	}

	return domainCR
}

// Update env variables for a given component
func UpdateEnvVars(mc *types.ManagedCluster, component string, envs *[]corev1.EnvVar) {
	if *envs != nil && len(*envs) != 0 {
		for _, domain := range mc.WlsDomainCRs {
			if domain.Name == component {
				for _, env := range *envs {
					domain.Spec.ServerPod.Env = append(domain.Spec.ServerPod.Env, env)
				}
				return
			}
		}
	}
}

// Add fluentd to the server pod
func addFluentdConfig(serverPod *v7weblogic.ServerPod, domainModel v1beta1v8o.VerrazzanoWebLogicDomain, mbPair *types.ModelBindingPair) {
	// Add fluentd container
	serverPod.Containers = append(serverPod.Containers, createFluentdContainer(domainModel, mbPair))

	// Add fluentd volumes
	serverPod.Volumes = append(serverPod.Volumes, createFluentdVolEmptyDir("weblogic-domain-storage-volume"))
	serverPod.Volumes = append(serverPod.Volumes, createFluentdVolConfigMap())

	// Add fluentd volume mount
	serverPod.VolumeMounts = append(serverPod.VolumeMounts, createFluentdVolumeMount("weblogic-domain-storage-volume"))
}
