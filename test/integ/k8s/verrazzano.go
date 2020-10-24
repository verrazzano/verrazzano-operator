// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	vzclient "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerrazzanoK8sClient struct {
	client *vzclient.Clientset
}

func (c K8sClient) DoesModelExist(name string) bool {
	_, err := c.vzClient.VerrazzanoV1beta1().VerrazzanoModels("default").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoModels")
}

func (c K8sClient) DoesBindingExist(name string) bool {
	_, err := c.vzClient.VerrazzanoV1beta1().VerrazzanoBindings("default").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoBindings")
}

func (c K8sClient) DoesVmiExist(name string) bool {
	_, err := c.vmiClient.VerrazzanoV1().VerrazzanoMonitoringInstances("verrazzano-system").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoMonitoringInstances")
}
