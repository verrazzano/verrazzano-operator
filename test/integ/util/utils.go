// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"io/ioutil"
	"path/filepath"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/test/integ/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// Creates a Managed Cluster with the given name, based on the local test kubeconfig
func CreateManagedCluster(f *framework.Framework, clusterName string) (*v1beta1v8o.VerrazzanoManagedCluster, error) {
	secret :=
		&corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: f.Namespace,
			},
			Data: map[string][]byte{
				"kubeconfig": []byte(f.KubeConfigContents),
			},
		}
	_, err := f.KubeClient.CoreV1().Secrets(f.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	managedCluster := &v1beta1v8o.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: f.Namespace,
		},
		Spec: v1beta1v8o.VerrazzanoManagedClusterSpec{
			KubeconfigSecret: clusterName,
		},
	}
	_, err = f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoManagedClusters(f.Namespace).Create(context.TODO(), managedCluster, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return managedCluster, nil
}

// Deletes the given ManagedCluster
func DeleteManagedCluster(f *framework.Framework, clusterName string) error {
	var returnErr error
	err := f.KubeClient.CoreV1().Secrets(f.Namespace).Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	if err != nil {
		returnErr = err
	}
	err = f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoManagedClusters(f.Namespace).Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	if err != nil {
		returnErr = err
	}
	return returnErr
}

// Creates an Application Model that will match with the Application Binding
func CreateAppModel(f *framework.Framework, modelName string) (*v1beta1v8o.VerrazzanoModel, error) {
	appModel := &v1beta1v8o.VerrazzanoModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelName,
			Namespace: f.Namespace,
		},
		Spec: v1beta1v8o.VerrazzanoModelSpec{
			Description: "test model",
		},
	}

	_, err := f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoModels(f.Namespace).Create(context.TODO(), appModel, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return appModel, nil
}

// Creates an Application Binding with the given name, with the given "components" and the clusters they attach to
func CreateAppBinding(f *framework.Framework, bindingName string, modelName string, componentNameToCluster map[string]string) (*v1beta1v8o.VerrazzanoBinding, error) {
	appBinding := &v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bindingName,
			Namespace: f.Namespace,
		},
		Spec: v1beta1v8o.VerrazzanoBindingSpec{
			ModelName: modelName,
		},
	}

	for componentName, clusterName := range componentNameToCluster {
		placement := v1beta1v8o.VerrazzanoPlacement{
			Name: clusterName,
			Namespaces: []v1beta1v8o.KubernetesNamespace{
				{
					Name: f.Namespace,
					Components: []v1beta1v8o.BindingComponent{
						{
							Name: componentName,
						},
					},
				},
			},
		}
		appBinding.Spec.Placement = append(appBinding.Spec.Placement, placement)
	}

	_, err := f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoBindings(f.Namespace).Create(context.TODO(), appBinding, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return appBinding, nil
}

// Deletes the given ApplicationModel
func DeleteAppModel(f *framework.Framework, modelName string) error {
	err := f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoModels(f.Namespace).Delete(context.TODO(), modelName, metav1.DeleteOptions{})
	return err
}

// Deletes the given ApplicationBinding
func DeleteAppBinding(f *framework.Framework, bindingName string) error {
	err := f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoBindings(f.Namespace).Delete(context.TODO(), bindingName, metav1.DeleteOptions{})
	return err
}

// ReadModel reads/unmarshal's a model yaml file into a VerrazzanoModel
func ReadModel(path string) (*v1beta1v8o.VerrazzanoModel, error) {
	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	var vmodel v1beta1v8o.VerrazzanoModel
	err = yaml.Unmarshal(yamlFile, &vmodel)
	return &vmodel, err
}

// ReadBinding reads/unmarshal's VerrazzanoBinding yaml file into a VerrazzanoBinding
func ReadBinding(path string) (*v1beta1v8o.VerrazzanoBinding, error) {
	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	var vbnd v1beta1v8o.VerrazzanoBinding
	err = yaml.Unmarshal(yamlFile, &vbnd)
	return &vbnd, err
}

// Write writes/marshalls the obj to a yaml file
func WriteYmal(path string, obj interface{}) (string, error) {
	fileout, _ := filepath.Abs(path)
	bytes, err := ToYmal(obj)
	err = ioutil.WriteFile(fileout, bytes, 0644)
	return fileout, err
}

// ToYmal arshalls the obj to a yaml
func ToYmal(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}
