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
	filebeatLabels := GetFilebeatLabels(managedClusterName)
	journalbeatLabels := GetJournalbeatLabels(managedClusterName)
	nodeExporterLabels := GetNodeExporterLabels(managedClusterName)
	var daemonSets []*appsv1.DaemonSet

	fileabeatDS, err := createFilebeatDaemonSet(constants.LoggingNamespace, constants.FilebeatName, filebeatLabels, clusterInfo)
	if err != nil {
		zap.S().Debugf("New Daemonset %s is giving error %s", constants.FilebeatName, err)
	}
	journalbeatDS, err := createJournalbeatDaemonSet(constants.LoggingNamespace, constants.JournalbeatName, journalbeatLabels, clusterInfo)
	if err != nil {
		zap.S().Debugf("New Daemonset %s is giving error %s", constants.JournalbeatName, err)
	}
	nodeExporterDS, err := createNodeExporterDaemonSet(constants.MonitoringNamespace, constants.NodeExporterName, nodeExporterLabels)
	if err != nil {
		zap.S().Debugf("New Daemonset %s is giving error %s", constants.NodeExporterName, err)
	}

	daemonSets = append(daemonSets, fileabeatDS, journalbeatDS, nodeExporterDS)
	return daemonSets
}

func createFilebeatDaemonSet(namespace string, name string, labels map[string]string, clusterInfo ClusterInfo) (*appsv1.DaemonSet, error) {

	loggingDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: "RollingUpdate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name + "-config",
									},
									DefaultMode: resources.NewVal(0600),
								},
							},
						},
						{
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: getFilebeatLogHostPath(clusterInfo),
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/filebeat-data",
									Type: nil,
								},
							},
						},
						{
							Name: "inputs",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "filebeat-inputs",
									},
									DefaultMode: resources.NewVal(0600),
								},
							},
						},
						{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: constants.FilebeatName + "-secret"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    name,
							Image:   util.GetFilebeatImage(),
							Command: nil,
							Args: []string{
								"-c", "/etc/filebeat/filebeat.yml",
								"-e",
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"memory": resource.MustParse("800Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("200Mi"),
								},
							},
							WorkingDir: "",
							Ports:      nil,
							EnvFrom:    nil,
							Env: []corev1.EnvVar{
								{
									Name: "NODENAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
								{
									Name:  "ES_URL",
									Value: getElasticsearchURL(clusterInfo),
								},
								{
									Name: "ES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name + "-secret",
											},
											Key: "username",
										},
									},
								},
								{
									Name: "ES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name + "-secret",
											},
											Key: "password",
										},
									},
								},
								{
									Name: "INDEX_NAME",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "filebeat-index-config",
											},
											Key: "filebeat-index-name",
										},
									},
								},
								{
									Name:  "CLUSTER_NAME",
									Value: clusterInfo.ManagedClusterName,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/etc/filebeat",
								},
								{
									Name:      "inputs",
									ReadOnly:  true,
									MountPath: "/usr/share/filebeat/inputs.d",
								},
								{
									Name:      "data",
									MountPath: "/usr/share/filebeat/data",
								},
								{
									Name:      "varlibdockercontainers",
									ReadOnly:  true,
									MountPath: "/var/lib/docker/containers",
								},
								{
									Name:      "secret",
									ReadOnly:  true,
									MountPath: "/etc/filebeat/secret",
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								Privileged: nil,
								RunAsUser:  resources.New64Val(0),
								ProcMount:  nil,
							},
						},
					},
					TerminationGracePeriodSeconds: resources.New64Val(30),
					ServiceAccountName:            name,
				},
			},
		},
	}
	return loggingDaemonSet, nil
}

func createJournalbeatDaemonSet(namespace string, name string, labels map[string]string, clusterInfo ClusterInfo) (*appsv1.DaemonSet, error) {

	loggingDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: "RollingUpdate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name + "-config",
									},
									DefaultMode: resources.NewVal(0600),
									Items: []corev1.KeyToPath{
										{
											Key:  "journalbeat.yml",
											Path: "journalbeat.yml",
										},
									},
								},
							},
						},
						{
							Name: "var-log-journal",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log/journal",
								},
							},
						},
						{
							Name: "run-log-journal",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/log/journal",
								},
							},
						},
						{
							Name: "etc-machine-id",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/machine-id",
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/journalbeat-data",
									Type: nil,
								},
							},
						},
						{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: constants.JournalbeatName + "-secret"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    name,
							Image:   util.GetJournalbeatImage(),
							Command: nil,
							Args: []string{
								"-c", "/etc/journalbeat/journalbeat.yml",
								"-e",
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{

									"memory": resource.MustParse("800Mi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("200Mi"),
								},
							},
							WorkingDir: "",
							Ports:      nil,
							EnvFrom:    nil,
							Env: []corev1.EnvVar{
								{
									Name: "NODENAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
								{
									Name:  "ES_URL",
									Value: getElasticsearchURL(clusterInfo),
								},
								{
									Name: "ES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "journalbeat-secret",
											},
											Key: "username",
										},
									},
								},
								{
									Name: "ES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "journalbeat-secret",
											},
											Key: "password",
										},
									},
								},
								{
									Name: "INDEX_NAME",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "journalbeat-index-config",
											},
											Key: "journalbeat-index-name",
										},
									},
								},
								{
									Name:  "CLUSTER_NAME",
									Value: clusterInfo.ManagedClusterName,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/etc/journalbeat",
								},
								{
									Name:      "run-log-journal",
									ReadOnly:  true,
									MountPath: "/run/log/journal",
								},
								{
									Name:      "var-log-journal",
									ReadOnly:  true,
									MountPath: "/var/log/journal",
								},
								{
									Name:      "data",
									MountPath: "/usr/share/journalbeat/data",
								},
								{
									Name:      "etc-machine-id",
									ReadOnly:  true,
									MountPath: "/etc/machine-id",
								},
								{
									Name:      "secret",
									ReadOnly:  true,
									MountPath: "/etc/journalbeat/secret",
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								Privileged: nil,
								RunAsUser:  resources.New64Val(0),
								ProcMount:  nil,
							},
						},
					},
					TerminationGracePeriodSeconds: resources.New64Val(30),
					ServiceAccountName:            name,
				},
			},
		},
	}
	return loggingDaemonSet, nil
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
