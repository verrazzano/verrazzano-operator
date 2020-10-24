// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	. "github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func (c K8sClient) DoesCRDExist(crdName string) bool {
	crds, err := c.apixClient.CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		Fail("Failed to get list of CustomResourceDefinitions")
	}
	for i := range crds.Items {
		if strings.Compare(crds.Items[i].ObjectMeta.Name, crdName) == 0 {
			return true
		}
	}
	return false
}

func (c K8sClient) DoesClusterRoleExist(name string) bool {
	_, err := c.clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "ClusterRole")
}

func (c K8sClient) DoesClusterRoleBindingExist(name string) bool {
	_, err := c.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "ClusterRoleBinding")
}

func (c K8sClient) DoesNamespaceExist(name string) bool {
	_, err := c.clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Namespaces")
}

func (c K8sClient) DoesSecretExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Secrets")
}

func (c K8sClient) DoesDaemonsetExist(name string, namespace string) bool {
	_, err := c.clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "DaemonSets")
}

func (c K8sClient) DoesDeploymentExist(name string, namespace string) bool {
	_, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Deployments")
}

func (c K8sClient) DoesPodExist(name string, namespace string) bool {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Fail("Could not get list of pods" + err.Error())
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			return true
		}
	}
	return false
}

func (c K8sClient) DoesServiceExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Services")
}

func (c K8sClient) DoesServiceAccountExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "ServiceAccounts")
}

func procExistsStatus(err error, msg string) bool {
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get " + msg)
	}
	return false
}
