// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestLoggingConfigMaps(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "cluster1"
	filebeatLabels := GetFilebeatLabels(clusterName)
	journalbeatLabels := GetJournalbeatLabels(clusterName)

	testMap := map[string]corev1.ConfigMap{
		"index-config": {ObjectMeta: metav1.ObjectMeta{
			Name:      "index-config",
			Namespace: constants.LoggingNamespace,
			Labels:    filebeatLabels},
			Data: map[string]string{"index_name": "vmo-demo-filebeat"},
		},
		"filebeat-index-config": {ObjectMeta: metav1.ObjectMeta{
			Name:      "filebeat-index-config",
			Namespace: constants.LoggingNamespace,
			Labels:    filebeatLabels},
			Data: map[string]string{"filebeat-index-name": "vmo-" + clusterName + "-filebeat-%{+yyyy.MM.dd}"},
		},
		"filebeat-config": {ObjectMeta: metav1.ObjectMeta{
			Name:      "filebeat-config",
			Namespace: constants.LoggingNamespace,
			Labels:    filebeatLabels},
			Data: map[string]string{"filebeat.yml": FilebeatConfigData},
		},
		"filebeat-inputs": {ObjectMeta: metav1.ObjectMeta{
			Name:      "filebeat-config",
			Namespace: constants.LoggingNamespace,
			Labels:    filebeatLabels},
			Data: map[string]string{"kubernetes.yml": FilebeatInputData},
		},
		"journalbeat-index-config": {ObjectMeta: metav1.ObjectMeta{
			Name:      "journalbeat-index-config",
			Namespace: constants.LoggingNamespace,
			Labels:    journalbeatLabels},
			Data: map[string]string{"journalbeat-index-name": "vmo-" + clusterName + "-journalbeat-%{+yyyy.MM.dd}"},
		},
		"journalbeat-config": {ObjectMeta: metav1.ObjectMeta{
			Name:      "journalbeat-index-config",
			Namespace: constants.LoggingNamespace,
			Labels:    journalbeatLabels},
			Data: map[string]string{"journalbeat.yml": JournalbeatConfigData},
		},
	}
	cmaps := LoggingConfigMaps(clusterName)
	assert.NotNilf(cmaps, "LoggingConfigMaps return nil")
	for _, v := range cmaps {
		tv, ok := testMap[v.Name]
		assert.Truef(ok, "LoggingConfigMaps returned unexpected entry &v", v.Name)
		assert.Equal(tv.Namespace, v.Namespace)
		assert.Equal(tv.Labels, v.Labels)
		for dk, dv := range v.Data {
			tdata, ok := tv.Data[dk]
			assert.Truef(ok, "Configmap data is missing for entry &v", v.Name)
			assert.Equal(tdata, dv)
		}
		delete(testMap, v.Name)
	}
	// All of the entries in the testMap should have been removed
	assert.Equalf(0, len(testMap), "LoggingConfigMaps did not return all of the entries")
	for _, v := range testMap {
		assert.Emptyf(v, "Configmap entry %v missing")
	}
}
