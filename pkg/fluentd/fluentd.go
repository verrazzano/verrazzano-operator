package fluentd

import (
	"fmt"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fluentdConf = "fluentd.conf"

// GenericComponentFluentdConfiguration Fluentd rules for reading/parsing generic component log files
const GenericComponentFluentdConfiguration = `<label @FLUENT_LOG>
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

// HelidonFluentdConfiguration Fluentd rules for reading/parsing generic component log files
const HelidonFluentdConfiguration = `<label @FLUENT_LOG>
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
  # Helidon application messages are expected to look like this:
  # 2020.04.22 16:09:21 INFO org.books.bobby.Main Thread[main,5,main]: http://localhost:8080/books
  <parse>
    @type multi_format
    <pattern>
      # Docker output
      format json
      time_format %Y-%m-%dT%H:%M:%S.%NZ
    </pattern>
    <pattern>
      # cri-o output
      format regexp
      expression /^(?<timestamp>(.*?)) (?<stream>stdout|stderr) (?<log>.*)$/
      time_format %Y-%m-%dT%H:%M:%S.%N%:z
    </pattern>
  </parse>
</source>
<filter **>
  @type parser
  key_name log
  <parse>
    @type grok
    <grok>
      name helidon-pattern
      pattern %{DATESTAMP:timestamp} %{DATA:loglevel} %{DATA:subsystem} %{DATA:thread} %{GREEDYDATA:message}
    </grok>
    <grok>
      name coherence-pattern
      pattern %{DATESTAMP:timestamp}/%{NUMBER:increment} %{DATA:subsystem} <%{DATA:loglevel}> (%{DATA:thread}): %{GREEDYDATA:message}
    </grok>
    <grok>
      name catchall-pattern
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

// CreateFluentdConfigMap creates the Fluentd configmap for a given generic application
func CreateFluentdConfigMap(fluentdConfig string, componentName string, namespace string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFluentdConfigMapName(componentName),
			Namespace: namespace,
			Labels:    labels,
		},
		Data: func() map[string]string {
			var data map[string]string
			data = make(map[string]string)
			data[fluentdConf] = fluentdConfig
			return data
		}(),
	}
}

// CreateFluentdContainer creates a Fluentd sidecar container.
func CreateFluentdContainer(bindingName string, componentName string) corev1.Container {
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
				Value: fluentdConf,
			},
			{
				Name:  "FLUENT_ELASTICSEARCH_SED_DISABLE",
				Value: "true",
			},
			{
				Name:  "ELASTICSEARCH_HOST",
				Value: fmt.Sprintf("vmi-%s-es-ingest.%s.svc.cluster.local", bindingName, constants.VerrazzanoNamespace),
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
				SubPath:   fluentdConf,
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

// CreateFluentdHostPathVolumes creates hostPath volumes to access container logs.
func CreateFluentdHostPathVolumes() []corev1.Volume {
	return []corev1.Volume{
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

// CreateFluentdConfigMapVolume create a config map volume for Fluentd.
func CreateFluentdConfigMapVolume(componentName string) corev1.Volume {
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

// getFluentdConfigMapName returns the name of a components Fluentd config map
func getFluentdConfigMapName(componentName string) string {
	return fmt.Sprintf("%s-fluentd", componentName)
}
