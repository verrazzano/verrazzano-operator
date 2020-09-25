// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"

	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

// KubeSecrets type used for CRUD operations related to secrets
type KubeSecrets struct {
	namespace     string
	kubeClientSet kubernetes.Interface
	secretLister  corev1listers.SecretLister
}

// Get returns a secret for a given secret name and namespace.
func (ks *KubeSecrets) Get(name string) (*corev1.Secret, error) {
	return ks.secretLister.Secrets(ks.namespace).Get(name)
}

// Create creates a secret for a given secret name and namespace.
func (ks *KubeSecrets) Create(newSecret *corev1.Secret) (*corev1.Secret, error) {
	return ks.kubeClientSet.CoreV1().Secrets(ks.namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
}

// Update updates a secret for a given secret name and namespace.
func (ks *KubeSecrets) Update(newSecret *corev1.Secret) (*corev1.Secret, error) {
	return ks.kubeClientSet.CoreV1().Secrets(ks.namespace).Update(context.TODO(), newSecret, metav1.UpdateOptions{})
}

// List returns a list of secrets for given namespace and selector.
func (ks *KubeSecrets) List(ns string, selector labels.Selector) (ret []*corev1.Secret, err error) {
	return ks.secretLister.Secrets(ns).List(selector)
}

// Delete deletes a secret for a given secret name and namespace.
func (ks *KubeSecrets) Delete(ns, name string) error {
	return ks.kubeClientSet.CoreV1().Secrets(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// GetVmiPassword returns a Verrazzano Monitoring Instance password for given namespace.
func (ks *KubeSecrets) GetVmiPassword() (string, error) {
	return monitoring.GetVmiPassword(ks)
}
