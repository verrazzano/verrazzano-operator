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

type KubeSecrets struct {
	namespace     string
	kubeClientSet kubernetes.Interface
	secretLister  corev1listers.SecretLister
}

func (ks *KubeSecrets) Get(name string) (*corev1.Secret, error) {
	return ks.secretLister.Secrets(ks.namespace).Get(name)
}

func (ks *KubeSecrets) Create(newSecret *corev1.Secret) (*corev1.Secret, error) {
	return ks.kubeClientSet.CoreV1().Secrets(ks.namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
}

func (ks *KubeSecrets) Update(newSecret *corev1.Secret) (*corev1.Secret, error) {
	return ks.kubeClientSet.CoreV1().Secrets(ks.namespace).Update(context.TODO(), newSecret, metav1.UpdateOptions{})
}

func (ks *KubeSecrets) List(ns string, selector labels.Selector) (ret []*corev1.Secret, err error) {
	return ks.secretLister.Secrets(ns).List(selector)
}

func (ks *KubeSecrets) Delete(ns, name string) error {
	return ks.kubeClientSet.CoreV1().Secrets(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (ks *KubeSecrets) GetVmiPassword() (string, error) {
	return monitoring.GetVmiPassword(ks)
}
