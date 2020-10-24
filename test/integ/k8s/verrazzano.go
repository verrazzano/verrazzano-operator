// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	. "github.com/onsi/ginkgo"

	vzclient "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerrazzanoK8sClient struct {
	client *vzclient.Clientset
}

func (c K8sClient) DoesModelExist(name string) bool {
	_, err := c.vzClient.VerrazzanoV1beta1().VerrazzanoModels("default").Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get VerrazzanoModels")
	}
	return false
}

func (c K8sClient) DoesBindingExist(name string) bool {
	_, err := c.vzClient.VerrazzanoV1beta1().VerrazzanoBindings("default").Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get VerrazzanoBindings")
	}
	return false
}

func (c K8sClient) DoesVmiExist(name string) bool {
	_, err := c.vmiClient.VerrazzanoV1().VerrazzanoMonitoringInstances("verrazzano-system").Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		Fail("Failed calling API to get VerrazzanoMonitoringInstances")
	}
	return false
}
