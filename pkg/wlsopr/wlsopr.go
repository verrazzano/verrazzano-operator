// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsopr

import (
	"errors"
	"fmt"

	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MicroOperatorName = "verrazzano-wko-operator"
const operatorName = "wls-operator"

// WlsOperatorCRConfig provides the parameters to construct the new WlsOperatorCR
type WlsOperatorCRConfig struct {
	BindingName        string
	ManagedClusterName string
	BindingNamespace   string
	DomainNamespace    string
	Labels             map[string]string
	WlsOperatorImage   string
}

// NewWlsOperatorCR creates a new WlsOperator CR
func NewWlsOperatorCR(cr WlsOperatorCRConfig) (*v1beta1.WlsOperator, error) {

	if cr.ManagedClusterName == "" {
		return nil, errors.New("ManagedClusterName is required")
	}
	if cr.DomainNamespace == "" {
		return nil, errors.New("DomainNamespace is required")
	}
	if cr.BindingNamespace == "" {
		return nil, errors.New("BindingNamespace is required")
	}

	wlsopr := &v1beta1.WlsOperator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WlsOperator",
			APIVersion: "verrazzano.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: cr.BindingNamespace,
			Labels:    cr.Labels,
		},
		Spec: v1beta1.WlsOperatorSpec{
			Description:      fmt.Sprintf("WebLogic Operator for managed cluster %s using binding %s", cr.ManagedClusterName, cr.BindingName),
			Name:             fmt.Sprintf("%s-%s", operatorName, cr.BindingName),
			Namespace:        cr.BindingNamespace,
			ServiceAccount:   operatorName,
			Image:            cr.WlsOperatorImage,
			ImagePullPolicy:  "IfNotPresent",
			DomainNamespaces: []string{cr.DomainNamespace},
		},
		Status: v1beta1.WlsOperatorStatus{},
	}
	return wlsopr, nil
}

// CreateDeployment creates a deployment for the verrazzano-wko-operator
func CreateDeployment(namespace string, bindingName string, labels map[string]string, image string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MicroOperatorName,
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
							Name:            MicroOperatorName,
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
							Command:         []string{"verrazzano-wko-operator"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(util.GetWlsMicroRequestMemory()),
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
									Value: MicroOperatorName,
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
