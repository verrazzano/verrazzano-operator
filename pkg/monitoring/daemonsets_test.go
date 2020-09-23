// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestCreateFilebeatDaemonSet(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "cluster1"
	const uri = "vzURI"
	nameMap := map[string]bool{
			constants.FilebeatName: true,
			constants.JournalbeatName: true,
			constants.NodeExporterName: true}
	dsets := SystemDaemonSets(clusterName, uri)
	for _,v := range dsets {
		switch v.Name {
		case constants.FilebeatName:
			validateFilebeatDaemonset(assert, *v, clusterName)
			break
		}
		delete(nameMap,v.Name)
	}
	// All of the entries in the testMap should have been removed
	assert.Equalf(0, len(nameMap), "SystemDaemonSets did not return all of the entries")
	for _, v := range nameMap {
		assert.Emptyf(v, "Map entry %v missing")
	}
}

func validateFilebeatDaemonset(assert *assert.Assertions, v appsv1.DaemonSet, clusterName string) {
	filebeatLabels := GetFilebeatLabels(clusterName)

	assert.Equal(constants.LoggingNamespace, v.Namespace)
	assert.Equal(filebeatLabels, v.Labels)
	assert.Equal(filebeatLabels, v.Spec.Selector.MatchLabels)
	assert.Equal("RollingUpdate", string(v.Spec.UpdateStrategy.Type))
	assert.Equal(filebeatLabels, v.Spec.Template.Labels)

	// Volumes
	assert.Lenf(v.Spec.Template.Spec.Volumes,4, "Spec.Template.Spec.Volumes has wrong number of items")

	assert.Equal("config", v.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(v.Name + "-config", v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name)
	assert.Equal(int32(0600), *v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.DefaultMode)

	assert.Equal("varlibdockercontainers", v.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal("/var/lib/docker/containers", v.Spec.Template.Spec.Volumes[1].VolumeSource.HostPath.Path)

	assert.Equal("data", v.Spec.Template.Spec.Volumes[2].Name)
	assert.Equal("/var/lib/filebeat-data", v.Spec.Template.Spec.Volumes[2].VolumeSource.HostPath.Path)
	assert.Nil(v.Spec.Template.Spec.Volumes[2].VolumeSource.HostPath.Type)

	assert.Equal("inputs", v.Spec.Template.Spec.Volumes[3].Name)
	assert.Equal("filebeat-inputs",
		v.Spec.Template.Spec.Volumes[3].VolumeSource.ConfigMap.LocalObjectReference.Name)
	assert.Equal(int32(0600), *v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.DefaultMode)

	// Containers
	assert.Lenf(v.Spec.Template.Spec.Containers,1, "Spec.Template.Spec.Containers has wrong number of items")
	assert.Equal(v.Name, v.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(util.GetFilebeatImage(), v.Spec.Template.Spec.Containers[0].Image)
	assert.Nil(v.Spec.Template.Spec.Containers[0].Command)
	assert.Len( v.Spec.Template.Spec.Containers[0].Args,3,
		"Spec.Template.Spec.Containers[0].Args has the wrong number of args)" )
	assert.Equal("-c", v.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal("/etc/filebeat/filebeat.yml", v.Spec.Template.Spec.Containers[0].Args[1])
	assert.Equal("-e", v.Spec.Template.Spec.Containers[0].Args[2])
	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Limits, "memory", "800Mi",
		"Spec.Template.Spec.Containers[0].Resources.Limits")
	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Requests, "cpu", "200m",
		"Spec.Template.Spec.Containers[0].Resources.Limits")
	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Requests, "memory", "200Mi",
		"Spec.Template.Spec.Containers[0].Resources.Limits")

	assert.Empty(v.Spec.Template.Spec.Containers[0].WorkingDir)
	assert.Nil(v.Spec.Template.Spec.Containers[0].Ports)
	assert.Nil(v.Spec.Template.Spec.Containers[0].EnvFrom)

	// Env Vars
	assert.Equal("NODENAME", v.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal("v1", v.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.FieldRef.APIVersion)
	assert.Equal("spec.nodeName", v.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.FieldRef.FieldPath)

	assert.Equal("ES_URL", v.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(fmt.Sprintf("http://vmi-system-es-ingest.%s.svc.cluster.local", constants.VerrazzanoNamespace),
		 v.Spec.Template.Spec.Containers[0].Env[1].Value)

	assert.Equal("ES_USER", v.Spec.Template.Spec.Containers[0].Env[2].Name)
	assert.Equal(constants.FilebeatName + "-secret",
		v.Spec.Template.Spec.Containers[0].Env[2].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal("username", v.Spec.Template.Spec.Containers[0].Env[2].ValueFrom.SecretKeyRef.Key)


	assert.Equal("ES_PASSWORD", v.Spec.Template.Spec.Containers[0].Env[3].Name)
	assert.Equal(constants.FilebeatName + "-secret",
		v.Spec.Template.Spec.Containers[0].Env[3].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal("password", v.Spec.Template.Spec.Containers[0].Env[3].ValueFrom.SecretKeyRef.Key)

	assert.Equal("ES_PORT", v.Spec.Template.Spec.Containers[0].Env[4].Name)
	assert.Equal("9200", v.Spec.Template.Spec.Containers[0].Env[4].Value)

	assert.Equal("INDEX_NAME", v.Spec.Template.Spec.Containers[0].Env[5].Name)
	assert.Equal("filebeat-index-config",
		v.Spec.Template.Spec.Containers[0].Env[5].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name)
	assert.Equal("filebeat-index-name",
		v.Spec.Template.Spec.Containers[0].Env[5].ValueFrom.ConfigMapKeyRef.Key)

	// Volume mounts
	assert.Equal("config", v.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	assert.Equal("/etc/filebeat", v.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)

	assert.Equal("inputs", v.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[1].ReadOnly)
	assert.Equal("/usr/share/filebeat/inputs.d", v.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)

	assert.Equal("data", v.Spec.Template.Spec.Containers[0].VolumeMounts[2].Name)
	assert.Equal("/usr/share/filebeat/data", v.Spec.Template.Spec.Containers[0].VolumeMounts[2].MountPath)

	assert.Equal("varlibdockercontainers", v.Spec.Template.Spec.Containers[0].VolumeMounts[3].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[3].ReadOnly)
	assert.Equal("/var/lib/docker/containers", v.Spec.Template.Spec.Containers[0].VolumeMounts[3].MountPath)


}

func validateResSize(assert *assert.Assertions, rm corev1.ResourceList, key corev1.ResourceName, val string, field string) {
	q, ok := rm[key]
	assert.Truef(ok, "%v entry is missing for key '%v'", field, string(key))
	assert.Equal(val, q.String(), "%v entry has the wrong value for key '%v'", field, string(key))
}


