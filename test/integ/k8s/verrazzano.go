// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DoesVmiExist returns true if the given VerrazzanoMonitoringInstance exists
func (c Client) DoesVmiExist(name string) bool {
	_, err := c.vmiClient.VerrazzanoV1().VerrazzanoMonitoringInstances("verrazzano-system").Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "VerrazzanoMonitoringInstance")
}
