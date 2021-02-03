// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateConfigMapsVmiSystem(t *testing.T) {
	assert := assert.New(t)

	modelBindingPair := types.NewModelBindingPair()
	clusterConnections := testutil.GetManagedClusterConnections()
	const clusterName = "cluster1"
	clusterConnection := clusterConnections[clusterName]

	modelBindingPair.Binding.Name = constants.VmiSystemBindingName
	err := CreateConfigMaps(&modelBindingPair, clusterConnections)
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
