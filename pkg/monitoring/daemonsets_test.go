// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"testing"
)

func TestCreateFilebeatDaemonSet(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "cluster1"
	const uri = "vzURI"
	nameMap := map[string]string{
			constants.FilebeatName: constants.LoggingNamespace,
			constants.JournalbeatName: constants.LoggingNamespace,
			constants.NodeExporterName: constants.MonitoringNamespace}
	dsetSet := make(map[string]bool)
	dsets := SystemDaemonSets(clusterName, uri)
	for _,v := range dsets {
		dsetSet[v.Name] = true
		assert.Equal(nameMap[v.Name], v.Namespace)
		assert.NotNil(v.Labels, "Daemonset labels are nil")
	}
	// Make sure there is a daemonset for each expected deamon
	for k,_ := range nameMap {
		_, ok := dsetSet[k]
		if !assert.True(ok, "Daemonset missing entry for %v", k) {
			t.FailNow()
		}
	}
}
