// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohcluster

import (
	v1coh "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kind resource kind for CoherenceCluster
const Kind = "CoherenceCluster"

// APIVersion resource API version for CoherenceCluster
const APIVersion = "coherenceclusters.coherence.oracle.com/v1"

// CreateCR constructs the CoherenceCluster resource
func CreateCR(namespace string, cluster *v1beta1v8o.VerrazzanoCoherenceCluster, cohBinding *v1beta1v8o.VerrazzanoCoherenceBinding, labels map[string]string) *v1coh.CoherenceCluster {
	applicationImage := cluster.Image
	cacheConfig := cluster.CacheConfig
	coherenceImage := "ghcr.io/oracle/coherence-ce:14.1.1-0-3"
	coherenceCluster := v1coh.CoherenceCluster{
		TypeMeta: v1meta.TypeMeta{
			Kind:       Kind,
			APIVersion: APIVersion,
		},
		ObjectMeta: v1meta.ObjectMeta{
			Name:      cluster.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v1coh.CoherenceClusterSpec{
			CoherenceRoleSpec: v1coh.CoherenceRoleSpec{
				Replicas: func() *int32 {
					// Set replicas if specified in the binding or the default of 3 if not specified
					if cohBinding.Replicas != nil {
						return cohBinding.Replicas
					}
					return util.NewVal(3)
				}(),
				// Make sure Coherence pods do not inject istio since Coherence does not work with istio
				Annotations: map[string]string{
					"sidecar.istio.io/inject": "false",
					"prometheus.io/path":      "/metrics",
					"prometheus.io/port":      "9612",
					"prometheus.io/scrape":    "true",
				},
				Coherence: &v1coh.CoherenceSpec{
					// Set the cacheConfig
					CacheConfig: &cacheConfig,
					// Set the coherence image to coherence CE
					ImageSpec: v1coh.ImageSpec{
						Image: &coherenceImage,
					},
					// Enable metrics for Prometheus
					Metrics: &v1coh.PortSpecWithSSL{
						Enabled: func() *bool { b := true; return &b }(),
						Port:    util.NewVal(9612),
					},
				},
				// Set the pofConfig
				JVM: &v1coh.JVMSpec{
					Args: []string{
						"-Dcoherence.pof.config=" + cluster.PofConfig,
					},
				},
				// Set the Coherence application image
				Application: &v1coh.ApplicationSpec{
					ImageSpec: v1coh.ImageSpec{
						Image: &applicationImage,
					},
				},
				// Set the optional ports
				Ports: func() []v1coh.NamedPortSpec {
					var portSpecs []v1coh.NamedPortSpec
					for _, v := range cluster.Ports {
						ps := v1coh.NamedPortSpec{
							Name: v.Name,
							PortSpec: v1coh.PortSpec{
								Port:     v.PortSpec.Port,
								Protocol: v.PortSpec.Protocol,
								Service:  v.PortSpec.Service,
							},
						}
						portSpecs = append(portSpecs, ps)
					}
					return portSpecs
				}(),
			},
			// Add any imagePullSecrets that were specified
			ImagePullSecrets: func() []v1coh.LocalObjectReference {
				var imagePullSecrets []v1coh.LocalObjectReference
				if len(cluster.ImagePullSecrets) != 0 {
					for _, secret := range cluster.ImagePullSecrets {
						var imagePullSecret v1coh.LocalObjectReference
						imagePullSecret.Name = secret.Name
						imagePullSecrets = append(imagePullSecrets, imagePullSecret)
					}
				}
				return imagePullSecrets
			}(),
		},
	}

	return &coherenceCluster
}

// UpdateEnvVars update env variables for a given component
func UpdateEnvVars(mc *types.ManagedCluster, component string, envs *[]corev1.EnvVar) {
	if *envs != nil && len(*envs) != 0 {
		for _, cluster := range mc.CohClusterCRs {
			if cluster.Name == component {
				for _, env := range *envs {
					cluster.Spec.CoherenceRoleSpec.Env = append(cluster.Spec.CoherenceRoleSpec.Env, env)
				}
				return
			}
		}
	}
}
