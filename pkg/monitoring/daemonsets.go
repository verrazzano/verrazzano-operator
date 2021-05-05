// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"github.com/verrazzano/verrazzano-monitoring-operator/pkg/resources"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SystemDaemonSets create all the Daemon sets needed by Filebeats, Journalbeats, and NodeExporters
// in all the managed clusters.
func SystemDaemonSets(managedClusterName string, verrazzanoURI string, clusterInfo ClusterInfo) []*appsv1.DaemonSet {
	nodeExporterLabels := GetNodeExporterLabels(managedClusterName)
	var daemonSets []*appsv1.DaemonSet
	nodeExporterDS, err := createNodeExporterDaemonSet(constants.MonitoringNamespace, constants.NodeExporterName, nodeExporterLabels)
	if err != nil {
		zap.S().Debugf("New Daemonset %s is giving error %s", constants.NodeExporterName, err)
	}

	daemonSets = append(daemonSets, nodeExporterDS)
	return daemonSets
}

func createNodeExporterDaemonSet(namespace string, name string, labels map[string]string) (*appsv1.DaemonSet, error) {

	applabel := map[string]string{
		constants.ServiceAppLabel: constants.NodeExporterName,
	}

	NodeExporterDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: applabel,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: "RollingUpdate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9100",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "proc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/proc",
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "root",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: util.GetNodeExporterImage(),
							Args: []string{
								"--web.listen-address=0.0.0.0:9100",
								"--path.procfs=/host/proc",
								"--path.sysfs=/host/sys",
								"--path.rootfs=/host/root",
								"--collector.filesystem.ignored-mount-points=^/(dev|proc|sys|var/lib/docker/.+)($|/)",
								"--collector.filesystem.ignored-fs-types=^(autofs|binfmt_misc|cgroup|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|mqueue|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|sysfs|tracefs)$",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 9100,
									HostPort:      9100,
									Protocol:      "TCP",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("180Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("102m"),
									"memory": resource.MustParse("180Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "proc",
									ReadOnly:  true,
									MountPath: "/host/proc",
								},
								{
									Name:      "sys",
									ReadOnly:  true,
									MountPath: "/host/sys",
								},
								{
									Name:      "root",
									ReadOnly:  true,
									MountPath: "/host/root",
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					RestartPolicy:      "Always",
					DNSPolicy:          "ClusterFirst",
					ServiceAccountName: name,
					HostNetwork:        true,
					HostPID:            true,
					NodeSelector: map[string]string{
						"beta.kubernetes.io/os": "linux",
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: resources.New64Val(65534),
					},
					Tolerations: []corev1.Toleration{
						{
							Effect:   "NoSchedule",
							Operator: "Exists",
						},
					},
				},
			},
		},
	}
	return NodeExporterDaemonSet, nil
}
