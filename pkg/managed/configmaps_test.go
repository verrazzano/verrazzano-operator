// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateConfigMaps(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	err := CreateConfigMaps(modelBindingPair, clusterConnections)
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

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]

	err := CreateConfigMaps(modelBindingPair, clusterConnections)
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
	cm := modelBindingPair.ManagedClusters["cluster1"].ConfigMaps[0].Data
	cm["bar"] = "ddd"
	cm["biz"] = "ccc"

	err = CreateConfigMaps(modelBindingPair, clusterConnections)
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

func TestCreateConfigMapsVmiSystem(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	const clusterName = "cluster1"
	clusterConnection := clusterConnections[clusterName]

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName
	err := CreateConfigMaps(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("can't create config maps: %v", err)
	}
	list, err := clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("can't list config maps: %v", err)
	}
	assert.Equal(6, len(list.Items))

	cm, err := clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").Get(context.TODO(), "index-config", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal("vmo-demo-filebeat", cm.Data["index_name"])
	filebeatLabels := monitoring.GetFilebeatLabels(clusterName)
	assert.Equal(filebeatLabels, cm.Labels)

	cm, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").Get(context.TODO(), "filebeat-index-config", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal("vmo-cluster1-filebeat-%{+yyyy.MM.dd}", cm.Data["filebeat-index-name"])
	assert.Equal(filebeatLabels, cm.Labels)

	cm, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").Get(context.TODO(), "filebeat-config", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal(monitoring.FilebeatConfigData, cm.Data["filebeat.yml"])
	assert.Equal(filebeatLabels, cm.Labels)

	cm, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").Get(context.TODO(), "filebeat-inputs", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal(monitoring.FilebeatInputData, cm.Data["kubernetes.yml"])
	assert.Equal(filebeatLabels, cm.Labels)

	cm, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").Get(context.TODO(), "journalbeat-index-config", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal("vmo-cluster1-journalbeat-%{+yyyy.MM.dd}", cm.Data["journalbeat-index-name"])
	journalbeatLabels := monitoring.GetJournalbeatLabels(clusterName)
	assert.Equal(journalbeatLabels, cm.Labels)

	cm, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("logging").Get(context.TODO(), "journalbeat-config", metav1.GetOptions{})
	if err != nil {
		t.Errorf("can't get configmap: %v", err)
	}
	assert.Equal(monitoring.JournalbeatConfigData, cm.Data["journalbeat.yml"])
	assert.Equal(journalbeatLabels, cm.Labels)
}

func TestCleanupOrphanedConfigMaps(t *testing.T) {
	modelBindingPair := testutil.GetModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster3"]

	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphaned",
			Namespace: "test",
			Labels: map[string]string{
				"verrazzano.binding": "testBinding",
				"verrazzano.cluster": "cluster3",
			},
		},
	}
	_, err := clusterConnection.KubeClient.CoreV1().ConfigMaps("test").Create(context.TODO(), &cm, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("can't create config map: %v", err)
	}
	err = CleanupOrphanedConfigMaps(modelBindingPair, clusterConnections)
	if err != nil {
		t.Fatalf("can't cleanup config maps: %v", err)
	}
	_, err = clusterConnection.KubeClient.CoreV1().ConfigMaps("test").Get(context.TODO(), "orphaned", metav1.GetOptions{})
	if err == nil {
		t.Error("expected orphaned configmap to be cleaned up")
	}
}
