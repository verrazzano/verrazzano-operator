// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"errors"
	"fmt"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSystemDeployments constructs the necessary Deployments for the specified ManagedCluster for the System VMI.
func GetSystemDeployments(managedClusterName, verrazzanoURI string, labels map[string]string, sec Secrets) ([]*appsv1.Deployment, error) {
	var deployments []*appsv1.Deployment

	if managedClusterName == "" {
		return nil, errors.New("GetSystemDeployments managedClusterName parameter must not be empty")
	}
	if verrazzanoURI == "" {
		return nil, errors.New("GetSystemDeployments verrazzanoURI parameter must not be empty")
	}
	deployment, err := CreateDeployment(constants.MonitoringNamespace, constants.VmiSystemBindingName, labels, sec)
	if err != nil {
		return nil, err
	}
	deployments = append(deployments, deployment)

	return deployments, nil
}

// DeletePomPusher deletes the Prometheus Pusher deployment.
func DeletePomPusher(binding string, helper util.DeploymentHelper) error {
	return helper.DeleteDeployment(constants.MonitoringNamespace, pomPusherName(binding))
}

func pomPusherName(bindingName string) string {
	return fmt.Sprintf("prom-pusher-%s", bindingName)
}

// CreateDeployment creates prometheus pusher deployment on all clusters, based on a VerrazzanoApplicationBinding.
func CreateDeployment(namespace string, bindingName string, labels map[string]string, sec Secrets) (*appsv1.Deployment, error) {
	payload := "match%5B%5D=%7Bjob%3D~%22" + bindingName + "%2E%2A%22%7D" // URL encoded : match[]={job=~"binding-name.*"}
	password, err := sec.GetVmiPassword()
	if err != nil {
		return nil, err
	}
	image := util.GetPromtheusPusherImage()
	pusherDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      pomPusherName(bindingName),
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: util.NewVal(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "prometheus-pusher",
							Image: image,
							Env: []corev1.EnvVar{
								{
									Name:  "PULL_URL_prometheus_pusher",
									Value: "http://prometheus.istio-system.svc.cluster.local:9090/federate?" + payload,
								},
								{
									Name: "PUSHGATEWAY_URL",

									Value: fmt.Sprintf("http://vmi-%s-prometheus-gw.%s.svc.cluster.local:9091", bindingName, constants.VerrazzanoNamespace),
								},
								{
									Name:  "PUSHGATEWAY_USER",
									Value: constants.VmiUsername,
								},
								{
									Name:  "PUSHGATEWAY_PASSWORD",
									Value: password,
								},
								{
									Name:  "LOGLEVEL",
									Value: "4",
								},
								{
									Name:  "SPLIT_SIZE",
									Value: "1000",
								},
								{
									Name:  "no_proxy",
									Value: "localhost,prometheus.istio-system.svc.cluster.local,127.0.0.1,/var/run/docker.sock",
								},
								{
									Name:  "NO_PROXY",
									Value: "localhost,prometheus.istio-system.svc.cluster.local,127.0.0.1,/var/run/docker.sock",
								},
								{
									Name:  "PROM_CERT",
									Value: "/verrazzano/certs/ca.crt",
								}},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports:           []corev1.ContainerPort{{Name: "master", ContainerPort: int32(9091)}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cert-vol",
									MountPath: "/verrazzano/certs",
								},
							},
						},
					},
					TerminationGracePeriodSeconds: util.New64Val(1),
					Volumes: []corev1.Volume{
						{
							Name: "cert-vol",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  "system-tls",
									DefaultMode: util.NewVal(420),
								},
							},
						},
					},
				},
			},
		},
	}
	return pusherDeployment, nil
}
