// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsopr

import (
	"fmt"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const microOperatorName = "verrazzano-wko-operator"
const operatorName = "wls-operator"

// Create a WlsOperator CR
func CreateWlsOperatorCR(binding *v1beta1v8o.VerrazzanoBinding, managedClusterName string, domainNamespace string, labels map[string]string) *v1beta1.WlsOperator {

	return &v1beta1.WlsOperator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WlsOperator",
			APIVersion: "verrazzano.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: util.GetManagedNamespaceForBinding(binding),
			Labels:    labels,
		},
		Spec: v1beta1.WlsOperatorSpec{
			Description:      fmt.Sprintf("WebLogic Operator for managed cluster %s using binding %s", managedClusterName, binding.Name),
			Name:             fmt.Sprintf("%s-%s", operatorName, binding.Name),
			Namespace:        util.GetManagedNamespaceForBinding(binding),
			ServiceAccount:   operatorName,
			Image:            util.GetWeblogicOperatorImage(),
			ImagePullPolicy:  "IfNotPresent",
			DomainNamespaces: []string{domainNamespace},
		},
		Status: v1beta1.WlsOperatorStatus{},
	}
}

// Create a deployment for the verrazzano-wko-operator
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
							Command:         []string{"verrazzano-wko-operator"},
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
