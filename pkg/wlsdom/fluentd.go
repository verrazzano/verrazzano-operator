// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsdom

import (
	"fmt"
	"strconv"

	"github.com/verrazzano/verrazzano-operator/pkg/types"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create the fluentd container
func createFluentdContainer(domainModel v1beta1v8o.VerrazzanoWebLogicDomain, mbPair *types.ModelBindingPair) corev1.Container {
	container := corev1.Container{
		Name:            "fluentd",
		Args:            []string{"-c", "/etc/fluent.conf"},
		Image:           constants.FluentdImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name: "DOMAIN_UID",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['weblogic.domainUID']",
					},
				},
			},
			{
				Name: "SERVER_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['weblogic.serverName']",
					},
				},
			},
			{
				Name:  "LOG_PATH",
				Value: fmt.Sprintf("/scratch/logs/%s/$(SERVER_NAME).log", domainModel.Name),
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
				Value: fmt.Sprintf("elasticsearch.vmi.%s.%s", mbPair.Binding.Name, mbPair.VerrazzanoUri),
			},
			{
				Name:  "ELASTICSEARCH_PORT",
				Value: "443",
			},
			{
				Name:  "ELASTICSEARCH_SSL_VERIFY",
				Value: strconv.FormatBool(mbPair.SslVerify),
			},
			{
				Name: "ELASTICSEARCH_USER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: mbPair.Binding.Name,
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
							Name: mbPair.Binding.Name,
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
				MountPath: "/scratch",
				Name:      "weblogic-domain-storage-volume",
				ReadOnly:  true,
			},
		},
	}

	return container
}

func createFluentdVolEmptyDir(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func createFluentdVolConfigMap() corev1.Volume {
	return corev1.Volume{
		Name: "fluentd-config-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "fluentd-config",
				},
				DefaultMode: func(mode int32) *int32 {
					return &mode
				}(420),
			},
		},
	}
}

func createFluentdVolumeMount(name string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: "/scratch",
	}
}

// Create the fluentd configmap
func CreateFluentdConfigMap(namespace string, labels map[string]string) *corev1.ConfigMap {

	// fluentd parsing rules
	parsingRules := `<match fluent.**>
  @type null
</match>
<source>
  @type tail
  path "#{ENV['LOG_PATH']}"
  pos_file /tmp/server.log.pos
  read_from_head true
  tag "#{ENV['DOMAIN_UID']}"
  # messages look like this:
  #   firstline:  ####
  #   format1:    <Mar 17, 2020 2:41:55,029 PM EDT> 
  #   format2:    <Info> 
  #   format3:    <WorkManager>
  #   format4:    <meerkat>
  #   format5:    <AdminServer>
  #   format6:    <Timer-2> 
  #   format7:    <<WLS Kernel>> 
  #   format8:    <> 
  #   format9:    <00ccb822-8beb-4ce0-905d-2039c4fd676f-00000010> 
  #   format10:   <1584470515029> 
  #   format11:   <[severity-value: 64] [rid: 0] [partition-id: 0] [partition-name: DOMAIN] > 
  #   format12:   <BEA-002959> 
  #   formart13:  <Self-tuning thread pool contains 0 running threads, 1 idle threads, and 12 standby threads> 
  <parse>
	@type multiline
	format_firstline /^####/
	format1 /^####<(?<timestamp>(.*?))>/
	format2 / <(?<level>(.*?))>/
	format3 / <(?<subSystem>(.*?))>/
	format4 / <(?<serverName>(.*?))>/
	format5 / <(?<serverName2>(.*?))>/
	format6 / <(?<threadName>(.*?))>/
	format7 / <(?<info1>(.*?))>/
	format8 / <(?<info2>(.*?))>/
	format9 / <(?<info3>(.*?))>/
	format10 / <(?<sequenceNumber>(.*?))>/
	format11 / <(?<severity>(.*?))>/
	format12 / <(?<messageID>(.*?))>/
	format13 / <(?<message>(.*?))>/
	time_key timestamp
	keep_time_key true
  </parse>
</source>
<filter **>
  @type record_transformer
  <record>
    domainUID "#{ENV['DOMAIN_UID']}"
  </record>
</filter>
<match **>
  @type elasticsearch
  host "#{ENV['ELASTICSEARCH_HOST']}"
  port "#{ENV['ELASTICSEARCH_PORT']}"
  user "#{ENV['ELASTICSEARCH_USER']}"
  password "#{ENV['ELASTICSEARCH_PASSWORD']}"
  index_name "#{ENV['DOMAIN_UID']}"
  scheme https
  ssl_version TLSv1_2
  ssl_verify "#{ENV['ELASTICSEARCH_SSL_VERIFY']}"
  key_name timestamp 
  types timestamp:time
  include_timestamp true
</match>
`
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-config",
			Namespace: namespace,
			Labels:    labels,
		},
		Data: func() map[string]string {
			var data map[string]string
			data = make(map[string]string)
			data["fluentd.conf"] = parsingRules
			return data
		}(),
	}
}
