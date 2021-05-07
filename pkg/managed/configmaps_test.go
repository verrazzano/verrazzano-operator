// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateConfigMaps(t *testing.T) {
	assert := assert.New(t)

	SyntheticModelBinding := testutil.GetSyntheticModelBinding()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	err := CreateConfigMaps(SyntheticModelBinding, clusterConnections, clusterInfoDocker)
	if err != nil {
		t.Fatalf("can't create config maps: %v", err)
	}

	configmap, err := clusterConnection.KubeClient.CoreV1().ConfigMaps("test").Get(context.TODO(), "test-configmap", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal("test-configmap", configmap.Name)
	assert.Equal(2, len(configmap.Data))
	assert.Equal("aaa", configmap.Data["foo"])
	assert.Equal("bbb", configmap.Data["bar"])
}

func TestCreateConfigMapsUpdateMap(t *testing.T) {
	assert := assert.New(t)

	SyntheticModelBinding := testutil.GetSyntheticModelBinding()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	err := CreateConfigMaps(SyntheticModelBinding, clusterConnections, clusterInfoDocker)
	if err != nil {
		t.Fatalf("can't create config maps: %v", err)
	}

	configmap, err := clusterConnection.KubeClient.CoreV1().ConfigMaps("test").Get(context.TODO(), "test-configmap", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal("test-configmap", configmap.Name)
	assert.Equal(2, len(configmap.Data))
	assert.Equal("aaa", configmap.Data["foo"])
	assert.Equal("bbb", configmap.Data["bar"])
	assert.Equal("", configmap.Data["biz"])

	// update the config map in the binding
	cm := SyntheticModelBinding.ManagedClusters["cluster1"].ConfigMaps[0].Data
	cm["bar"] = "ddd"
	cm["biz"] = "ccc"

	err = CreateConfigMaps(SyntheticModelBinding, clusterConnections, clusterInfoDocker)
	if err != nil {
		t.Fatalf("can't create config maps: %v", err)
	}

	configmap, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("test").Get(context.TODO(), "test-configmap", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal("test-configmap", configmap.Name)
	assert.Equal(3, len(configmap.Data))
	assert.Equal("aaa", configmap.Data["foo"])
	assert.Equal("ddd", configmap.Data["bar"])
	assert.Equal("ccc", configmap.Data["biz"])
}
