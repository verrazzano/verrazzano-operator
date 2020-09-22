// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohcluster

import (
	"testing"

	v1coh "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	v1betav8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestCreateCR(t *testing.T) {
	assert := assert.New(t)

	serviceSpec := &v1coh.ServiceSpec{}
	protocol := "tcp"
	serviceSpec2 := &v1coh.ServiceSpec{}
	protocol2 := "udp"

	vzCluster := &v1betav8o.VerrazzanoCoherenceCluster{
		Name:             "clusterName",
		Image:            "testImage",
		CacheConfig:      "cacheConfig",
		PofConfig:        "pofConfig",
		ImagePullSecrets: []corev1.LocalObjectReference{{Name: "secret1"}},
		Ports: []v1coh.NamedPortSpec{
			{Name: "port1",
				PortSpec: v1coh.PortSpec{
					Port:     5000,
					Protocol: &protocol,
					Service:  serviceSpec,
				}},
			{Name: "port2",
				PortSpec: v1coh.PortSpec{
					Port:     5001,
					Protocol: &protocol2,
					Service:  serviceSpec2,
				}},
		}}

	binding := &v1betav8o.VerrazzanoCoherenceBinding{
		Name:     "testBinding",
		Replicas: util.NewVal(5),
	}
	labels := map[string]string{"label1": "val1", "label2": "val2"}
	cluster := CreateCR("testNS", vzCluster, binding, labels)
	assert.NotNil(cluster, "CreateCR returned nil")

	assert.Equal("CoherenceCluster", cluster.TypeMeta.Kind)
	assert.Equal("coherenceclusters.coherence.oracle.com/v1", cluster.TypeMeta.APIVersion)

	assert.Equal("clusterName", cluster.ObjectMeta.Name)
	assert.Equal("testNS", cluster.ObjectMeta.Namespace)
	assert.Equal(2, len(cluster.ObjectMeta.Labels))
	assert.Equal("val1", cluster.ObjectMeta.Labels["label1"])
	assert.Equal("val2", cluster.ObjectMeta.Labels["label2"])

	assert.Equal(int32(5), *cluster.Spec.CoherenceRoleSpec.Replicas)

	assert.Equal("false", cluster.Spec.Annotations["sidecar.istio.io/inject"])
	assert.Equal("/metrics", cluster.Spec.Annotations["prometheus.io/path"])
	assert.Equal("9612", cluster.Spec.Annotations["prometheus.io/port"])
	assert.Equal("true", cluster.Spec.Annotations["prometheus.io/scrape"])

	assert.Equal("cacheConfig", *cluster.Spec.Coherence.CacheConfig)
	assert.True(*cluster.Spec.Coherence.Metrics.Enabled)
	assert.Equal(int32(9612), *cluster.Spec.Coherence.Metrics.Port)

	assert.Equal(1, len(cluster.Spec.JVM.Args))
	assert.Equal("-Dcoherence.pof.config=pofConfig", cluster.Spec.JVM.Args[0])

	assert.Equal("testImage", *cluster.Spec.Application.ImageSpec.Image)

	assert.Equal(int32(5000), cluster.Spec.Ports[0].Port)
	assert.Equal("port1", cluster.Spec.Ports[0].Name)
	assert.Equal(int32(5000), cluster.Spec.Ports[0].PortSpec.Port)
	assert.Equal(serviceSpec, cluster.Spec.Ports[0].PortSpec.Service)
	assert.Equal(&protocol, cluster.Spec.Ports[0].PortSpec.Protocol)
	assert.Equal(int32(5001), cluster.Spec.Ports[1].Port)
	assert.Equal("port2", cluster.Spec.Ports[1].Name)
	assert.Equal(int32(5001), cluster.Spec.Ports[1].PortSpec.Port)
	assert.Equal(serviceSpec2, cluster.Spec.Ports[1].PortSpec.Service)
	assert.Equal(&protocol2, cluster.Spec.Ports[1].PortSpec.Protocol)

	assert.Equal("secret1", cluster.Spec.ImagePullSecrets[0].Name)

	// Check default replica
	binding.Replicas = nil
	cluster2 := CreateCR("testNS", vzCluster, binding, labels)
	assert.Equal(int32(3), *cluster2.Spec.CoherenceRoleSpec.Replicas)
}

func TestUpdateEnvVars(t *testing.T) {
	assert := assert.New(t)

	const cohComp = "coh"
	const wlsComp = "wls"

	mc := types.ManagedCluster{
		CohClusterCRs: []*v1coh.CoherenceCluster{
			{ObjectMeta: v1.ObjectMeta{Name: cohComp}},
			{ObjectMeta: v1.ObjectMeta{Name: wlsComp}},
		},
	}

	envSrc := &corev1.EnvVarSource{}
	envs := []corev1.EnvVar{
		{Name: "key1", Value: "val1"},
		{Name: "key2", ValueFrom: envSrc},
	}
	UpdateEnvVars(&mc, "coh", &envs)

	assert.Equal(2, len(mc.CohClusterCRs[0].Spec.Env))
	assert.Equal("key1", mc.CohClusterCRs[0].Spec.Env[0].Name)
	assert.Equal("val1", mc.CohClusterCRs[0].Spec.Env[0].Value)
	assert.Equal("key2", mc.CohClusterCRs[0].Spec.Env[1].Name)
	assert.Equal(envSrc, mc.CohClusterCRs[0].Spec.Env[1].ValueFrom)
	assert.Equal(0, len(mc.CohClusterCRs[1].Spec.Env))
}
