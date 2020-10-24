// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	. "github.com/onsi/ginkgo"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"path/filepath"
	"sigs.k8s.io/yaml"
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
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get ClusterRole")
	}
	return false
}

func (c K8sClient) DoesClusterRoleBindingExist(name string) bool {
	_, err := c.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get ClusterRoleBinding")
	}
	return false
}

func (c K8sClient) DoesNamespaceExist(name string) bool {
	_, err := c.clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get Namespaces")
	}
	return false
}

func (c K8sClient) DoesSecretExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get Secrets")
	}
	return false
}

func (c K8sClient) DoesDeployementExist(name string, namespace string) bool {
	_, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get Deployments")
	}
	return false
}

func (c K8sClient) DoesServiceExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get Services")
	}
	return false
}

func (c K8sClient) DoesServiceAccountExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get ServiceAccounts")
	}
	return false
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

// WriteYmal writes/marshalls the obj to a yaml file.
func WriteYmal(path string, obj interface{}) (string, error) {
	fileout, _ := filepath.Abs(path)
	bytes, err := ToYmal(obj)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(fileout, bytes, 0644)
	return fileout, err
}

// ToYmal marshalls the obj to a yaml
func ToYmal(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}
