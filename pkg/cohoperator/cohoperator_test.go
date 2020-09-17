// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohoperator

import (
	v1betav8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCR(t *testing.T) {
	assert := assert.New(t)
	vzCluster := &v1betav8o.VerrazzanoCoherenceCluster{
		ImagePullSecrets: []corev1.LocalObjectReference{ {Name: "secret1"}},
	}
	labels := map[string]string{"label1":"val1", "label2":"val2"}
	cohCluster := CreateCR("cohOp", "testNS", vzCluster, labels)
	assert.NotNil(cohCluster, "CreateCR returned nil")

	assert.Equal("CohCluster", cohCluster.TypeMeta.Kind)
	assert.Equal("verrazzano.io/v1beta1", cohCluster.TypeMeta.APIVersion)

	assert.Equal("testNS-coherence-operator", cohCluster.ObjectMeta.Name)
	assert.Equal("testNS", cohCluster.ObjectMeta.Namespace)
	assert.Equal(2, len(cohCluster.ObjectMeta.Labels))
	assert.Equal("val1", cohCluster.ObjectMeta.Labels["label1"])
	assert.Equal("val2", cohCluster.ObjectMeta.Labels["label2"])

	assert.Equal("Coherence operator for managed cluster cohOp", cohCluster.Spec.Description, )
	assert.Equal("testNS-coherence-operator", cohCluster.Spec.Name, )
	assert.Equal("testNS", cohCluster.Spec.Namespace, )
	assert.Equal("testNS-coherence-operator", cohCluster.Spec.ServiceAccount, )
	assert.Equal(1, len(cohCluster.Spec.ImagePullSecrets))
	assert.Equal("secret1", cohCluster.Spec.ImagePullSecrets[0].Name)
}

func TestCreateDeployment(t *testing.T) {
	assert := assert.New(t)

	// needed by CreateDeployment
	os.Setenv("COH_MICRO_REQUEST_MEMORY", "50Mi")

	labels := map[string]string{"label1":"val1", "label2":"val2"}
	dep := CreateDeployment("testNs","testBinding",labels, "testImage")
	assert.NotNil(dep, "CreateDeployment returned nil")

	assert.Equal(microOperatorName, dep.ObjectMeta.Name)
	assert.Equal("testNs", dep.ObjectMeta.Namespace)
	assert.Equal(2, len(dep.ObjectMeta.Labels))
	assert.Equal("val1", dep.ObjectMeta.Labels["label1"])
	assert.Equal("val2", dep.ObjectMeta.Labels["label2"])

	assert.Equal(int32(1), *dep.Spec.Replicas)
	assert.Equal(appsv1.RecreateDeploymentStrategyType, dep.Spec.Strategy.Type)
	assert.Equal(2, len(dep.Spec.Selector.MatchLabels))
	assert.Equal("val1", dep.Spec.Selector.MatchLabels["label1"])
	assert.Equal("val2", dep.Spec.Selector.MatchLabels["label2"])

	assert.Equal(2, len(dep.Spec.Template.ObjectMeta.Labels))
	assert.Equal("val1", dep.Spec.Template.ObjectMeta.Labels["label1"])
	assert.Equal("val2", dep.Spec.Template.ObjectMeta.Labels["label2"])
	assert.Equal(microOperatorName, dep.Spec.Template.Spec.Containers[0].Name)
	assert.Equal("testImage", dep.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(corev1.PullIfNotPresent, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	assert.Equal(microOperatorName, dep.Spec.Template.Spec.Containers[0].Command[0])
	assert.Equal(resource.MustParse("50Mi"), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory])
	assert.Equal(3, len(dep.Spec.Template.Spec.Containers[0].Env))
	assert.Equal("WATCH_NAMESPACE", dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal("", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal("POD_NAME", dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal("metadata.name", dep.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.FieldRef.FieldPath)
	assert.Equal("OPERATOR_NAME", dep.Spec.Template.Spec.Containers[0].Env[2].Name)
	assert.Equal(microOperatorName, dep.Spec.Template.Spec.Containers[0].Env[2].Value)
	assert.Equal(int64(1), *dep.Spec.Template.Spec.TerminationGracePeriodSeconds)
	assert.Equal(constants.VerrazzanoSystem, dep.Spec.Template.Spec.ServiceAccountName)
}
