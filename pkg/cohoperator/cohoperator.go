// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohoperator

import (
	"fmt"

	v1cohcluster "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/apis/verrazzano/v1beta1"
	v1betav8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const microOperatorName = "verrazzano-coh-cluster-operator"

func CreateCR(mcName string, mcNamespace string, cluster *v1betav8o.VerrazzanoCoherenceCluster, labels map[string]string) *v1cohcluster.CohCluster {
	var operatorName = fmt.Sprintf("%s-coherence-operator", mcNamespace)

	cohCluster := v1cohcluster.CohCluster{}

	cohCluster.TypeMeta.Kind = "CohCluster"
	cohCluster.TypeMeta.APIVersion = "verrazzano.io/v1beta1"

	cohCluster.ObjectMeta.Name = operatorName
	cohCluster.ObjectMeta.Namespace = mcNamespace
	cohCluster.ObjectMeta.Labels = labels

	cohCluster.Spec.Description = fmt.Sprintf("Coherence operator for managed cluster %s", mcName)
	cohCluster.Spec.Name = operatorName
	cohCluster.Spec.Namespace = mcNamespace
	cohCluster.Spec.ServiceAccount = operatorName

	if len(cluster.ImagePullSecrets) != 0 {
		cohCluster.Spec.ImagePullSecrets = cluster.ImagePullSecrets
	}

	return &cohCluster
}

// Create a deployment for the coh-cluster-operator
func CreateDeployment(namespace string, bindingName string, labels map[string]string, image string) *appsv1.Deployment {
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
							ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
							Command:         []string{microOperatorName},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(util.GetCohMicroRequestMemory()),
								},
							},
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
