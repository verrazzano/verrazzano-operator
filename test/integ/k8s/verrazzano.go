// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DoesModelExist returns true if the given VerrazzanoModel exists
func (c Client) DoesModelExist(name string) bool {
	_, err := c.vzClient.VerrazzanoV1beta1().VerrazzanoModels("default").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoModel")
}

// DoesBindingExist returns true if the given VerrazzanoBinding exists
func (c Client) DoesBindingExist(name string) bool {
	_, err := c.vzClient.VerrazzanoV1beta1().VerrazzanoBindings("default").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoBinding")
}

// DoesVmiExist returns true if the given VerrazzanoMonitoringInstance exists
func (c Client) DoesVmiExist(name string) bool {
	_, err := c.vmiClient.VerrazzanoV1().VerrazzanoMonitoringInstances("verrazzano-system").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoMonitoringInstance")
}
