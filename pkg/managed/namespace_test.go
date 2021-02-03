// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func assertNamespace(t *testing.T, clusterConnection *util.ManagedClusterConnection, name string, wantExist bool) {
	namespace, err := clusterConnection.KubeClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if wantExist {
		if err != nil {
			t.Errorf("can't get namespace: %v", err)
		}
		assert.Equal(t, name, namespace.Name)
	} else {
		if err == nil || namespace != nil {
			t.Errorf("namespace %s exists", name)
		}
	}
}

