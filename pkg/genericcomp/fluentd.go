// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package genericcomp

import (
	"fmt"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fluentdConf = "fluentd.conf"

// CreateFluentdConfigMap creates the Fluentd configmap for a given generic application
func CreateFluentdConfigMap(componentName string, namespace string, labels map[string]string) *corev1.ConfigMap {
	// fluentd parsing rules
	parsingRules := `<label @FLUENT_LOG>
  <match fluent.*>
    @type stdout
  </match>
</label>
<source>
  @type tail
  path "/var/log/containers/#{ENV['APPLICATION_NAME']}*#{ENV['APPLICATION_NAME']}*.log"
  pos_file "/tmp/#{ENV['APPLICATION_NAME']}.log.pos"
  read_from_head true
  tag "#{ENV['APPLICATION_NAME']}"
  format none
</source>
<filter **>
  @type parser
  key_name log
  <parse>
    @type grok
    <grok>
      name any-message
      pattern %{GREEDYDATA:message}
    </grok>
	time_key timestamp
	keep_time_key true
  </parse>
</filter>
<filter **>
  @type record_transformer
  <record>
    applicationName "#{ENV['APPLICATION_NAME']}"
  </record>
</filter>
<match **>
  @type elasticsearch
  host "#{ENV['ELASTICSEARCH_HOST']}"
  port "#{ENV['ELASTICSEARCH_PORT']}"
  user "#{ENV['ELASTICSEARCH_USER']}"
  password "#{ENV['ELASTICSEARCH_PASSWORD']}"
  index_name "#{ENV['APPLICATION_NAME']}"
  scheme http
  include_timestamp true
  flush_interval 10s
</match>
`
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFluentdConfigMapName(componentName),
			Namespace: namespace,
			Labels:    labels,
		},
		Data: func() map[string]string {
			var data map[string]string
			data = make(map[string]string)
			data[fluentdConf] = parsingRules
			return data
		}(),
	}
}

// Create the Fluentd sidecar container
func createFluentdContainer(binding *v1beta1v8o.VerrazzanoBinding, componentName string) corev1.Container {
	container := corev1.Container{
		Name:            "fluentd",
		Args:            []string{"-c", "/etc/fluent.conf"},
		Image:           util.GetFluentdImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "APPLICATION_NAME",
				Value: componentName,
			},
			{
				Name:  "FLUENTD_CONF",
				Value: "fluentd.conf",
			},
			{
				Name:  "FLUENT_ELASTICSEARCH_SED_DISABLE",
				Value: "true",
			},
			{
				Name:  "ELASTICSEARCH_HOST",
				Value: fmt.Sprintf("vmi-%s-es-ingest.%s.svc.cluster.local", binding.Name, constants.VerrazzanoNamespace),
			},
			{
				Name:  "ELASTICSEARCH_PORT",
				Value: "9200",
			},
			{
				Name: "ELASTICSEARCH_USER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: constants.VmiSecretName,
						},
						Key: "username",
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
			{
				Name: "ELASTICSEARCH_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: constants.VmiSecretName,
						},
						Key: "password",
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/fluentd/etc/fluentd.conf",
				Name:      "fluentd-config-volume",
				SubPath:   "fluentd.conf",
				ReadOnly:  true,
			},
			{
				MountPath: "/var/log",
				Name:      "varlog",
				ReadOnly:  true,
			},
			{
				MountPath: "/u01/data/docker/containers",
				Name:      "datadockercontainers",
				ReadOnly:  true,
			},
		},
	}

	return container
}

// Create hostPath volumes for logs
func createFluentdVolHostPaths() *[]corev1.Volume {
	return &[]corev1.Volume{
		{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "datadockercontainers",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/u01/data/docker/containers",
				},
			},
		},
	}
}

// Create volume for fluentd config map
func createFluentdVolConfigMap(componentName string) corev1.Volume {
	return corev1.Volume{
		Name: "fluentd-config-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getFluentdConfigMapName(componentName),
				},
				DefaultMode: func(mode int32) *int32 {
					return &mode
				}(420),
			},
		},
	}
}

// Get Fluentd configmap name
func getFluentdConfigMapName(componentName string) string {
	return fmt.Sprintf("%s-fluentd", componentName)
}
