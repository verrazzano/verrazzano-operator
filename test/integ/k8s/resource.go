// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	. "github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano-operator/test/integ/util"
	"io/ioutil"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

func IsPodRunning(name string, namespace string) bool {
	GinkgoWriter.Write([]byte("[DEBUG] checking if there is a running pod named " + name + "* in namespace " + namespace + "\n"))
	clientset := GetClientSet()
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		Fail("Could not get list of pods")
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			conditions := pods.Items[i].Status.Conditions
			for j := range conditions {
				if conditions[j].Type == "Ready" {
					if conditions[j].Status == "True" {
						return true
					}
				}
			}
		}
	}
	return false
}

func DoesCRDExist(crdName string) bool {
	kubeconfig := util.GetKubeconfig()

	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		Fail("Could not get current context from kubeconfig " + kubeconfig)
	}

	apixClient, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		Fail("Could not get apix client")
	}

	crds, err := apixClient.CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
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

func GetClientSet() *kubernetes.Clientset {
	kubeconfig := util.GetKubeconfig()

	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		Fail("Could not get current context from kubeconfig " + kubeconfig)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		Fail("Could not get clientset from config")
	}

	return clientset
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
