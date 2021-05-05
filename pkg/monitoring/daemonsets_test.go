// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateSystemDaemonSet(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "cluster1"
	const uri = "vzURI"
	nameMap := map[string]bool{
		constants.NodeExporterName: true}
	dsets := SystemDaemonSets(clusterName, uri, ClusterInfo{ContainerRuntime: "containerd://1.4.0"})
	for _, v := range dsets {
		switch v.Name {
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
