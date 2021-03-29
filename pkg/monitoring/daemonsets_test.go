// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateFilebeatDaemonSet(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "cluster1"
	const uri = "vzURI"
	nameMap := map[string]bool{
		constants.FilebeatName:     true,
		constants.JournalbeatName:  true,
		constants.NodeExporterName: true}
	dsets := SystemDaemonSets(clusterName, uri, clusterInfoContainerd)
	for _, v := range dsets {
		switch v.Name {
		case constants.FilebeatName:
			validateFilebeatDaemonset(assert, *v, clusterName)
			break
		case constants.JournalbeatName:
			validateJournalbeatDaemonset(assert, *v, clusterName)
			break
		case constants.NodeExporterName:
			validateNodeExporterDaemonSet(assert, *v, clusterName)
			break
		}
		delete(nameMap, v.Name)
	}
	// All of the entries in the testMap should have been removed
	assert.Equalf(0, len(nameMap), "SystemDaemonSets did not return all of the entries")
	for _, v := range nameMap {
		assert.Emptyf(v, "Map entry %v missing")
	}
}

func validateFilebeatDaemonset(assert *assert.Assertions, v appsv1.DaemonSet, clusterName string) {
	labels := GetFilebeatLabels(clusterName)

	assert.Equal(constants.LoggingNamespace, v.Namespace)
	assert.Equal(labels, v.Labels)
	assert.Equal(labels, v.Spec.Selector.MatchLabels)
	assert.Equal("RollingUpdate", string(v.Spec.UpdateStrategy.Type))
	assert.Equal(labels, v.Spec.Template.Labels)

	// Volumes
	assert.Lenf(v.Spec.Template.Spec.Volumes, 5, "Spec.Template.Spec.Volumes has wrong number of items")

	assert.Equal("config", v.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(v.Name+"-config", v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name)
	assert.Equal(int32(0600), *v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.DefaultMode)

	assert.Equal("varlibdockercontainers", v.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal(FilebeatLogHostPathContainerd, v.Spec.Template.Spec.Volumes[1].VolumeSource.HostPath.Path)

	assert.Equal("data", v.Spec.Template.Spec.Volumes[2].Name)
	assert.Equal("/var/lib/filebeat-data", v.Spec.Template.Spec.Volumes[2].VolumeSource.HostPath.Path)
	assert.Nil(v.Spec.Template.Spec.Volumes[2].VolumeSource.HostPath.Type)

	assert.Equal("inputs", v.Spec.Template.Spec.Volumes[3].Name)
	assert.Equal("filebeat-inputs",
		v.Spec.Template.Spec.Volumes[3].VolumeSource.ConfigMap.LocalObjectReference.Name)
	assert.Equal(int32(0600), *v.Spec.Template.Spec.Volumes[3].VolumeSource.ConfigMap.DefaultMode)

	assert.Equal("secret", v.Spec.Template.Spec.Volumes[4].Name)
	assert.Equal("filebeat-secret",
		v.Spec.Template.Spec.Volumes[4].VolumeSource.Secret.SecretName)

	// Containers
	assert.Lenf(v.Spec.Template.Spec.Containers, 1, "Spec.Template.Spec.Containers has wrong number of items")
	assert.Equal(v.Name, v.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(util.GetFilebeatImage(), v.Spec.Template.Spec.Containers[0].Image)
	assert.Nil(v.Spec.Template.Spec.Containers[0].Command)
	assert.Len(v.Spec.Template.Spec.Containers[0].Args, 3,
		"Spec.Template.Spec.Containers[0].Args has the wrong number of args)")
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
	assert.Equal(fmt.Sprintf("http://vmi-system-es-ingest.%s.svc.cluster.local:9200", constants.VerrazzanoNamespace),
		v.Spec.Template.Spec.Containers[0].Env[1].Value)

	assert.Equal("ES_USER", v.Spec.Template.Spec.Containers[0].Env[2].Name)
	assert.Equal(constants.FilebeatName+"-secret",
		v.Spec.Template.Spec.Containers[0].Env[2].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal("username", v.Spec.Template.Spec.Containers[0].Env[2].ValueFrom.SecretKeyRef.Key)

	assert.Equal("ES_PASSWORD", v.Spec.Template.Spec.Containers[0].Env[3].Name)
	assert.Equal(constants.FilebeatName+"-secret",
		v.Spec.Template.Spec.Containers[0].Env[3].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal("password", v.Spec.Template.Spec.Containers[0].Env[3].ValueFrom.SecretKeyRef.Key)

	assert.Equal("INDEX_NAME", v.Spec.Template.Spec.Containers[0].Env[4].Name)
	assert.Equal("filebeat-index-config",
		v.Spec.Template.Spec.Containers[0].Env[4].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name)
	assert.Equal("filebeat-index-name",
		v.Spec.Template.Spec.Containers[0].Env[4].ValueFrom.ConfigMapKeyRef.Key)

	assert.Equal("CLUSTER_NAME", v.Spec.Template.Spec.Containers[0].Env[5].Name)
	assert.Equal("", v.Spec.Template.Spec.Containers[0].Env[5].Value)

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

	assert.Equal("secret", v.Spec.Template.Spec.Containers[0].VolumeMounts[4].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[4].ReadOnly)
	assert.Equal("/etc/filebeat/secret", v.Spec.Template.Spec.Containers[0].VolumeMounts[4].MountPath)

	assert.Equal(corev1.PullIfNotPresent, v.Spec.Template.Spec.Containers[0].ImagePullPolicy)

	assert.Nil(v.Spec.Template.Spec.Containers[0].SecurityContext.Privileged)
	assert.Zero(*v.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
}

func validateJournalbeatDaemonset(assert *assert.Assertions, v appsv1.DaemonSet, clusterName string) {
	labels := GetJournalbeatLabels(clusterName)

	assert.Equal(constants.LoggingNamespace, v.Namespace)
	assert.Equal(labels, v.Labels)
	assert.Equal(labels, v.Spec.Selector.MatchLabels)
	assert.Equal("RollingUpdate", string(v.Spec.UpdateStrategy.Type))
	assert.Equal(labels, v.Spec.Template.Labels)

	// Volumes
	assert.Lenf(v.Spec.Template.Spec.Volumes, 6, "Spec.Template.Spec.Volumes has wrong number of items")

	assert.Equal("config", v.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(v.Name+"-config", v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name)
	assert.Equal(int32(0600), *v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.DefaultMode)
	assert.Equal("journalbeat.yml", v.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Key)

	assert.Equal("var-log-journal", v.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal("/var/log/journal", v.Spec.Template.Spec.Volumes[1].VolumeSource.HostPath.Path)

	assert.Equal("run-log-journal", v.Spec.Template.Spec.Volumes[2].Name)
	assert.Equal("/run/log/journal", v.Spec.Template.Spec.Volumes[2].VolumeSource.HostPath.Path)

	assert.Equal("etc-machine-id", v.Spec.Template.Spec.Volumes[3].Name)
	assert.Equal("/etc/machine-id", v.Spec.Template.Spec.Volumes[3].VolumeSource.HostPath.Path)

	assert.Equal("data", v.Spec.Template.Spec.Volumes[4].Name)
	assert.Equal("/var/lib/journalbeat-data", v.Spec.Template.Spec.Volumes[4].VolumeSource.HostPath.Path)
	assert.Nil(v.Spec.Template.Spec.Volumes[4].VolumeSource.HostPath.Type)

	assert.Equal("secret", v.Spec.Template.Spec.Volumes[5].Name)
	assert.Equal("journalbeat-secret",
		v.Spec.Template.Spec.Volumes[5].VolumeSource.Secret.SecretName)

	// Containers
	assert.Lenf(v.Spec.Template.Spec.Containers, 1, "Spec.Template.Spec.Containers has wrong number of items")
	assert.Equal(v.Name, v.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(util.GetFilebeatImage(), v.Spec.Template.Spec.Containers[0].Image)
	assert.Nil(v.Spec.Template.Spec.Containers[0].Command)
	assert.Len(v.Spec.Template.Spec.Containers[0].Args, 3,
		"Spec.Template.Spec.Containers[0].Args has the wrong number of args)")
	assert.Equal("-c", v.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal("/etc/journalbeat/journalbeat.yml", v.Spec.Template.Spec.Containers[0].Args[1])
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
	assert.Equal(fmt.Sprintf("http://vmi-system-es-ingest.%s.svc.cluster.local:9200", constants.VerrazzanoNamespace),
		v.Spec.Template.Spec.Containers[0].Env[1].Value)

	assert.Equal("ES_USER", v.Spec.Template.Spec.Containers[0].Env[2].Name)
	assert.Equal(constants.JournalbeatName+"-secret",
		v.Spec.Template.Spec.Containers[0].Env[2].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal("username", v.Spec.Template.Spec.Containers[0].Env[2].ValueFrom.SecretKeyRef.Key)

	assert.Equal("ES_PASSWORD", v.Spec.Template.Spec.Containers[0].Env[3].Name)
	assert.Equal(constants.JournalbeatName+"-secret",
		v.Spec.Template.Spec.Containers[0].Env[3].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal("password", v.Spec.Template.Spec.Containers[0].Env[3].ValueFrom.SecretKeyRef.Key)

	assert.Equal("INDEX_NAME", v.Spec.Template.Spec.Containers[0].Env[4].Name)
	assert.Equal(constants.JournalbeatName+"-index-config",
		v.Spec.Template.Spec.Containers[0].Env[4].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name)
	assert.Equal(constants.JournalbeatName+"-index-name",
		v.Spec.Template.Spec.Containers[0].Env[4].ValueFrom.ConfigMapKeyRef.Key)

	assert.Equal("CLUSTER_NAME", v.Spec.Template.Spec.Containers[0].Env[5].Name)
	assert.Equal("", v.Spec.Template.Spec.Containers[0].Env[5].Value)

	// Volume mounts
	assert.Equal("config", v.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	assert.Equal("/etc/journalbeat", v.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)

	assert.Equal("run-log-journal", v.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[1].ReadOnly)
	assert.Equal("/run/log/journal", v.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)

	assert.Equal("var-log-journal", v.Spec.Template.Spec.Containers[0].VolumeMounts[2].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[2].ReadOnly)
	assert.Equal("/var/log/journal", v.Spec.Template.Spec.Containers[0].VolumeMounts[2].MountPath)

	assert.Equal("data", v.Spec.Template.Spec.Containers[0].VolumeMounts[3].Name)
	assert.Equal("/usr/share/journalbeat/data", v.Spec.Template.Spec.Containers[0].VolumeMounts[3].MountPath)

	assert.Equal("etc-machine-id", v.Spec.Template.Spec.Containers[0].VolumeMounts[4].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[4].ReadOnly)
	assert.Equal("/etc/machine-id", v.Spec.Template.Spec.Containers[0].VolumeMounts[4].MountPath)

	assert.Equal("secret", v.Spec.Template.Spec.Containers[0].VolumeMounts[5].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[5].ReadOnly)
	assert.Equal("/etc/journalbeat/secret", v.Spec.Template.Spec.Containers[0].VolumeMounts[5].MountPath)

	assert.Equal(corev1.PullIfNotPresent, v.Spec.Template.Spec.Containers[0].ImagePullPolicy)

	assert.Nil(v.Spec.Template.Spec.Containers[0].SecurityContext.Privileged)
	assert.Zero(*v.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
	assert.Nil(v.Spec.Template.Spec.Containers[0].SecurityContext.ProcMount)

	assert.Equal(int64(30), *v.Spec.Template.Spec.TerminationGracePeriodSeconds)
	assert.Equal(v.Name, v.Spec.Template.Spec.ServiceAccountName)
}

func validateNodeExporterDaemonSet(assert *assert.Assertions, v appsv1.DaemonSet, clusterName string) {
	labels := GetNodeExporterLabels(clusterName)

	applabel := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}
	assert.Equal(constants.MonitoringNamespace, v.Namespace)
	assert.Equal(labels, v.Labels)
	assert.Equal(applabel, v.Spec.Selector.MatchLabels)
	assert.Equal("RollingUpdate", string(v.Spec.UpdateStrategy.Type))
	assert.Equal(labels, v.Spec.Template.Labels)

	// Annotations
	assert.Equal("true", v.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal("9100", v.Spec.Template.Annotations["prometheus.io/port"])

	// Volumes
	assert.Lenf(v.Spec.Template.Spec.Volumes, 3, "Spec.Template.Spec.Volumes has wrong number of items")

	assert.Equal("proc", v.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal("/proc", v.Spec.Template.Spec.Volumes[0].VolumeSource.HostPath.Path)

	assert.Equal("sys", v.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal("/sys", v.Spec.Template.Spec.Volumes[1].VolumeSource.HostPath.Path)

	assert.Equal("root", v.Spec.Template.Spec.Volumes[2].Name)
	assert.Equal("/", v.Spec.Template.Spec.Volumes[2].VolumeSource.HostPath.Path)

	// Containers
	assert.Lenf(v.Spec.Template.Spec.Containers, 1, "Spec.Template.Spec.Containers has wrong number of items")
	assert.Equal(v.Name, v.Spec.Template.Spec.Containers[0].Name)
	assert.Equal(util.GetNodeExporterImage(), v.Spec.Template.Spec.Containers[0].Image)
	assert.Len(v.Spec.Template.Spec.Containers[0].Args, 6,
		"Spec.Template.Spec.Containers[0].Args has the wrong number of args)")
	assert.Equal("--web.listen-address=0.0.0.0:9100", v.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal("--path.procfs=/host/proc", v.Spec.Template.Spec.Containers[0].Args[1])
	assert.Equal("--path.sysfs=/host/sys", v.Spec.Template.Spec.Containers[0].Args[2])
	assert.Equal("--path.rootfs=/host/root", v.Spec.Template.Spec.Containers[0].Args[3])
	assert.Equal("--collector.filesystem.ignored-mount-points=^/(dev|proc|sys|var/lib/docker/.+)($|/)",
		v.Spec.Template.Spec.Containers[0].Args[4])
	assert.Equal("--collector.filesystem.ignored-fs-types=^(autofs|binfmt_misc|cgroup|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|mqueue|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|sysfs|tracefs)$",
		v.Spec.Template.Spec.Containers[0].Args[5])

	// Ports
	assert.Equal("metrics", v.Spec.Template.Spec.Containers[0].Ports[0].Name)
	assert.Equal(int32(9100), v.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(int32(9100), v.Spec.Template.Spec.Containers[0].Ports[0].HostPort)
	assert.Equal("TCP", string(v.Spec.Template.Spec.Containers[0].Ports[0].Protocol))

	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Limits, "cpu", "250m",
		"Spec.Template.Spec.Containers[0].Resources.Limits")
	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Limits, "memory", "180Mi",
		"Spec.Template.Spec.Containers[0].Resources.Limits")
	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Requests, "cpu", "102m",
		"Spec.Template.Spec.Containers[0].Resources.Limits")
	validateResSize(assert, v.Spec.Template.Spec.Containers[0].Resources.Requests, "memory", "180Mi",
		"Spec.Template.Spec.Containers[0].Resources.Limits")

	// Volume mounts
	assert.Equal("proc", v.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	assert.Equal("/host/proc", v.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)

	assert.Equal("sys", v.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[1].ReadOnly)
	assert.Equal("/host/sys", v.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)

	assert.Equal("root", v.Spec.Template.Spec.Containers[0].VolumeMounts[2].Name)
	assert.True(v.Spec.Template.Spec.Containers[0].VolumeMounts[2].ReadOnly)
	assert.Equal("/host/root", v.Spec.Template.Spec.Containers[0].VolumeMounts[2].MountPath)

	assert.Equal(corev1.PullIfNotPresent, v.Spec.Template.Spec.Containers[0].ImagePullPolicy)

	assert.Equal("Always", string(v.Spec.Template.Spec.RestartPolicy))
	assert.Equal("ClusterFirst", string(v.Spec.Template.Spec.DNSPolicy))
	assert.Equal(v.Name, v.Spec.Template.Spec.ServiceAccountName)
	assert.True(v.Spec.Template.Spec.HostNetwork)
	assert.True(v.Spec.Template.Spec.HostPID)
	assert.Equal("linux", v.Spec.Template.Spec.NodeSelector["beta.kubernetes.io/os"])
	assert.Equal(int64(65534), *v.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal("NoSchedule", string(v.Spec.Template.Spec.Tolerations[0].Effect))
	assert.Equal("Exists", string(v.Spec.Template.Spec.Tolerations[0].Operator))
}

func validateResSize(assert *assert.Assertions, rm corev1.ResourceList, key corev1.ResourceName, val string, field string) {
	q, ok := rm[key]
	assert.Truef(ok, "%v entry is missing for key '%v'", field, string(key))
	assert.Equal(val, q.String(), "%v entry has the wrong value for key '%v'", field, string(key))
}
