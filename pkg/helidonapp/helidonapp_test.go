// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonapp

import (
	"fmt"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/stretchr/testify/assert"
	vz "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v1helidonapp "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateHelidonAppCR(t *testing.T) {
	mcName, namespace, appName := "myCluster", "myNs", "myHelidonApp"
	vzHelidon := vz.VerrazzanoHelidon{Name: appName}
	mbPair := types.ModelBindingPair{Binding: &vz.VerrazzanoBinding{Spec: vz.VerrazzanoBindingSpec{}}}
	mbPair.Binding.Spec.HelidonBindings = []vz.VerrazzanoHelidonBinding{{Name: appName}}
	labels := make(map[string]string)
	helidonApp := CreateHelidonAppCR(mcName, namespace, &vzHelidon, &mbPair, labels)
	assert.Equal(t, DefaultPort, helidonApp.Spec.Port, "Expected default Port")
	assert.Equal(t, DefaultTargetPort, helidonApp.Spec.TargetPort, "Expected default TargetPort")
	mbPair.Binding.Spec.HelidonBindings = []vz.VerrazzanoHelidonBinding{{
		Name: appName,
	}}
	port := uint(8887)
	targetPort := uint(8889)
	vzHelidon.Port = port
	vzHelidon.TargetPort = targetPort
	helidonApp = CreateHelidonAppCR(mcName, namespace, &vzHelidon, &mbPair, labels)
	assert.Equal(t, int32(port), helidonApp.Spec.Port, "Expected Port")
	assert.Equal(t, int32(targetPort), helidonApp.Spec.TargetPort, "Expected TargetPort")
}

// Test using default value for enabling Fluentd
func TestFluentdEnabledDefault(t *testing.T) {
	mcName, namespace, appName := "myCluster", "myNs", "myHelidonApp"
	vzHelidon := vz.VerrazzanoHelidon{Name: appName}
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec: vz.VerrazzanoBindingSpec{},
		},
		VerrazzanoUri: "test.v8o.xyz.com",
		SslVerify:     true,
	}
	mbPair.Binding.Spec.HelidonBindings = []vz.VerrazzanoHelidonBinding{{Name: appName}}
	labels := make(map[string]string)
	helidonApp := CreateHelidonAppCR(mcName, namespace, &vzHelidon, &mbPair, labels)
	checkFluentdEnabled(t, helidonApp, appName)
}

// Test using Fluentd explicitly enabled
func TestFluentdEnabledExplicitly(t *testing.T) {
	mcName, namespace, appName := "myCluster", "myNs", "myHelidonApp"
	enabled := true
	vzHelidon := vz.VerrazzanoHelidon{
		Name:           appName,
		FluentdEnabled: &enabled,
	}
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec: vz.VerrazzanoBindingSpec{},
		},
		VerrazzanoUri: "test.v8o.xyz.com",
		SslVerify:     true,
	}
	mbPair.Binding.Spec.HelidonBindings = []vz.VerrazzanoHelidonBinding{{Name: appName}}
	labels := make(map[string]string)
	helidonApp := CreateHelidonAppCR(mcName, namespace, &vzHelidon, &mbPair, labels)
	checkFluentdEnabled(t, helidonApp, appName)
}

// Test using Fluentd disabled
func TestFluentdDisabled(t *testing.T) {
	mcName, namespace, appName := "myCluster", "myNs", "myHelidonApp"
	enabled := false
	vzHelidon := vz.VerrazzanoHelidon{
		Name:           appName,
		FluentdEnabled: &enabled,
	}
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec: vz.VerrazzanoBindingSpec{},
		},
		VerrazzanoUri: "test.v8o.xyz.com",
		SslVerify:     true,
	}
	mbPair.Binding.Spec.HelidonBindings = []vz.VerrazzanoHelidonBinding{{Name: appName}}
	labels := make(map[string]string)
	helidonApp := CreateHelidonAppCR(mcName, namespace, &vzHelidon, &mbPair, labels)
	assert.Equal(t, 0, len(helidonApp.Spec.Containers), "Expected containers count to be 0")
	assert.Equal(t, 0, len(helidonApp.Spec.Volumes), "Expected volumes count to be 0")
}

func checkFluentdEnabled(t *testing.T, helidonApp *v1helidonapp.HelidonApp, appName string) {
	assert.Equal(t, 1, len(helidonApp.Spec.Containers), "Expected containers count to be 1")
	assert.Equal(t, 2, len(helidonApp.Spec.Containers[0].Args), "Expected args count to be 2")
	name := "fluentd"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].Name, fmt.Sprintf("Expect container name to be %s", name))
	name = "-c"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].Args[0], fmt.Sprintf("Expect fist argument to be %s", name))
	name = "/etc/fluent.conf"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].Args[1], fmt.Sprintf("Expect second argument to be %s", name))
	assert.Equal(t, util.GetFluentdImage(), helidonApp.Spec.Containers[0].Image, fmt.Sprintf("Expect image name to be %s", util.GetFluentdImage()))
	assert.Equal(t, 3, len(helidonApp.Spec.Containers[0].VolumeMounts), "Expected volume mounts count to be 3")
	name = "fluentd-config-volume"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[0].Name, fmt.Sprintf("Expect volume mount name to be %s", name))
	name = "/fluentd/etc/fluentd.conf"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[0].MountPath, fmt.Sprintf("Expect volume mount path to be %s", name))
	name = "fluentd.conf"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[0].SubPath, fmt.Sprintf("Expect volume mount subpath to be %s", name))
	assert.Equal(t, true, helidonApp.Spec.Containers[0].VolumeMounts[0].ReadOnly, "Expect volume mount to be readOnly")
	name = "varlog"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[1].Name, fmt.Sprintf("Expect volume mount name to be %s", name))
	name = "/var/log"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[1].MountPath, fmt.Sprintf("Expect volume mount path to be %s", name))
	assert.Equal(t, true, helidonApp.Spec.Containers[0].VolumeMounts[1].ReadOnly, "Expect volume mount to be readOnly")
	name = "datadockercontainers"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[2].Name, fmt.Sprintf("Expect volume mount name to be %s", name))
	name = "/u01/data/docker/containers"
	assert.Equal(t, name, helidonApp.Spec.Containers[0].VolumeMounts[2].MountPath, fmt.Sprintf("Expect volume mount path to be %s", name))
	assert.Equal(t, true, helidonApp.Spec.Containers[0].VolumeMounts[2].ReadOnly, "Expect volume mount to be readOnly")
	assert.Equal(t, 8, len(helidonApp.Spec.Containers[0].Env), "Expected env count to be 8")
	es_username := findEnv(helidonApp.Spec.Containers[0].Env, "ELASTICSEARCH_USER")
	es_password := findEnv(helidonApp.Spec.Containers[0].Env, "ELASTICSEARCH_PASSWORD")
	assert.NotNil(t, es_username)
	assert.NotNil(t, es_password)
	assert.Equal(t, constants.VmiSecretName, es_username.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, constants.VmiSecretName, es_password.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, 3, len(helidonApp.Spec.Volumes), "Expected volumes count to be 3")
	name = "varlog"
	assert.Equal(t, name, helidonApp.Spec.Volumes[0].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = "/var/log"
	assert.Equal(t, name, helidonApp.Spec.Volumes[0].VolumeSource.HostPath.Path, fmt.Sprintf("Expected volume hostpath to be %s", name))
	name = "datadockercontainers"
	assert.Equal(t, name, helidonApp.Spec.Volumes[1].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = "/u01/data/docker/containers"
	assert.Equal(t, name, helidonApp.Spec.Volumes[1].VolumeSource.HostPath.Path, fmt.Sprintf("Expected volume hostpath to be %s", name))
	name = "fluentd-config-volume"
	assert.Equal(t, name, helidonApp.Spec.Volumes[2].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = fmt.Sprintf("%s-fluentd", appName)
	assert.Equal(t, name, helidonApp.Spec.Volumes[2].VolumeSource.ConfigMap.Name, fmt.Sprintf("Expected volume hostpath to be %s", name))
}

func findEnv(envvars []corev1.EnvVar, name string) *corev1.EnvVar {
	for _, env := range envvars {
		if name == env.Name {
			return &env
		}
	}
	return nil
}
