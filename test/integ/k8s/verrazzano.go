// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"github.com/verrazzano/verrazzano-operator/test/integ/util"
	"k8s.io/client-go/tools/clientcmd"

	vzclient "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerrazzanoK8sClient struct {
	client *vzclient.Clientset
}

func (c VerrazzanoK8sClient) DoesModelExist(name string) bool {
	_, err := c.client.VerrazzanoV1beta1().VerrazzanoModels("default").Get(context.TODO(), name, metav1.GetOptions{})
	return err == nil
}

func (c VerrazzanoK8sClient) DoesBindingExist(name string) bool {
	_, err := c.client.VerrazzanoV1beta1().VerrazzanoBindings("default").Get(context.TODO(), name, metav1.GetOptions{})
	return err == nil
}

// Get a new client that calls the Kubernetes API server to access the Verrazzano API Objects
func NewVzK8sClient() (VerrazzanoK8sClient, error) {
	kubeconfig := util.GetKubeconfig()

	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return VerrazzanoK8sClient{}, err
	}

	cli, err := vzclient.NewForConfig(config)
	if err != nil {
		return VerrazzanoK8sClient{}, err
	}
	return VerrazzanoK8sClient{client: cli}, err
}
