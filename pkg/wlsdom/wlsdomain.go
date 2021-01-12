// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsdom

import (
	"fmt"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateWlsDomainCR creates a WebLogic domain custom resource.
func CreateWlsDomainCR(namespace string, domainModel v1beta1v8o.VerrazzanoWebLogicDomain, mbPair *types.ModelBindingPair,
	labels map[string]string, datasourceModelConfigMap string, dbSecrets []string) *v8weblogic.Domain {
	domainCRValues := domainModel.DomainCRValues

	var domainUID string
	if len(domainCRValues.DomainUID) > 0 {
		domainUID = domainCRValues.DomainUID
	} else {
		domainUID = domainModel.Name
	}

	labels["weblogic.resourceVersion"] = "domain-v8"
	labels["weblogic.domainUID"] = domainUID

	domainCR := &v8weblogic.Domain{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      domainModel.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v8weblogic.DomainSpec{
			DomainHome: domainCRValues.DomainHome,
			DomainUID:  domainUID,
			Image:      domainCRValues.Image,
			ImagePullPolicy: func() string {
				// ImagePullPolicy
				if len(domainCRValues.ImagePullPolicy) > 0 {
					return domainCRValues.ImagePullPolicy
				}
				return "IfNotPresent"
			}(),
			ImagePullSecrets: domainCRValues.ImagePullSecrets,
			WebLogicCredentialsSecret: corev1.SecretReference{
				Name: domainCRValues.WebLogicCredentialsSecret.Name,
			},
			LogHome:                  fmt.Sprintf("/scratch/logs/%s", domainUID),
			LogHomeEnabled:           true,
			Clusters:                 domainCRValues.Clusters,
			IncludeServerOutInPodLog: domainCRValues.IncludeServerOutInPodLog,
			DomainHomeSourceType:     "FromModel",
			AdminServer: v8weblogic.AdminServer{
				ServerStartState: func() string {
					if len(domainCRValues.AdminServer.ServerStartState) > 0 {
						return domainCRValues.AdminServer.ServerStartState
					}
					return "RUNNING"
				}(),
				RestartVersion: domainCRValues.AdminServer.RestartVersion,
				ServerService:  domainCRValues.AdminServer.ServerService,
				AdminService: v8weblogic.AdminService{
					Channels: func() []v8weblogic.Channel {
						var channels []v8weblogic.Channel

						if domainCRValues.AdminServer.AdminService.Channels != nil {
							channels = domainCRValues.AdminServer.AdminService.Channels
						} else {
							// Default Channel
							port := domainModel.AdminPort
							if port > 0 {
								channels = append(channels, v8weblogic.Channel{
									ChannelName: "istio-default",
									NodePort:    port,
								})
							}

							// T3 Channel?
							if domainModel.T3Port > 0 {
								channels = append(channels, v8weblogic.Channel{
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
				}
				return util.NewVal(1)
			}(),
			Configuration: v8weblogic.Configuration{
				IntrospectorJobActiveDeadlineSeconds: func() int {
					if domainCRValues.Configuration.IntrospectorJobActiveDeadlineSeconds > 0 {
						return domainCRValues.Configuration.IntrospectorJobActiveDeadlineSeconds
					}
					return 60 * 10 // Default to 10 minutes if not specified in domainCRValues
				}(),
				Istio: v8weblogic.Istio{
					// Istio is always enabled for WebLogic domains in Verrazzano
					Enabled:       true,
					ReadinessPort: 8888,
				},
				Model: v8weblogic.Model{
					ConfigMap:               datasourceModelConfigMap,
					RuntimeEncryptionSecret: fmt.Sprintf("%s-runtime-encrypt-secret", domainUID),
				},
				Secrets: SliceUniqMap(append(domainCRValues.Configuration.Secrets, dbSecrets...)),
			},
			ServerStartPolicy: func() string {
				if len(domainCRValues.ServerStartPolicy) > 0 {
					return domainCRValues.ServerStartPolicy
				}
				return "IF_NEEDED"
			}(),
			ServerPod: func() v8weblogic.ServerPod {
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
				}
				return "RUNNING"
			}(),
			RestartVersion: domainCRValues.RestartVersion,
		},
		Status: v8weblogic.DomainStatus{},
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

// UpdateEnvVars given a WebLogic component update environment variables.
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
func addFluentdConfig(serverPod *v8weblogic.ServerPod, domainModel v1beta1v8o.VerrazzanoWebLogicDomain, mbPair *types.ModelBindingPair) {
	// Add fluentd container
	serverPod.Containers = append(serverPod.Containers, createFluentdContainer(domainModel, mbPair))

	// Add fluentd volumes
	serverPod.Volumes = append(serverPod.Volumes, createFluentdVolEmptyDir("weblogic-domain-storage-volume"))
	serverPod.Volumes = append(serverPod.Volumes, createFluentdVolConfigMap())

	// Add fluentd volume mount
	serverPod.VolumeMounts = append(serverPod.VolumeMounts, createFluentdVolumeMount("weblogic-domain-storage-volume"))
}

// SliceUniqMap will remove duplicate entries from the input
// slice - essentially turning a "list" into a "set"
// a "list" can have duplicate entries, a "set" cannot
func SliceUniqMap(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
