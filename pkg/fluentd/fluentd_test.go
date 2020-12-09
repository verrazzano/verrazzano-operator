// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"os"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

// TestCreateGenericFluentdConfigMap tests that a Fluentd config map is created for a generic component
// GIVEN a generic component Fluentd configuration
//  WHEN I call CreateFluentdConfigMap
//  THEN the expected config map should be created
func TestCreateGenericFluentdConfigMap(t *testing.T) {
	assertion := assert.New(t)

	labels := map[string]string{
		"myLabel": "hello",
	}
	cm := CreateFluentdConfigMap(GenericComponentFluentdConfiguration, "test-generic", "test-namespace", labels)
	assertion.Equal("test-generic-fluentd", cm.Name, "config map name not equal to expected value")
	assertion.Equal("test-namespace", cm.Namespace, "config map namespace not equal to expected value")
	assertion.Equal(labels, cm.Labels, "config map labels not equal to expected value")
	assertion.Equal(1, len(cm.Data), "config map data size not equal to expected value")
	assertion.NotNil(cm.Data[fluentdConf], "Expected fluentd.conf")
	assertion.Equal(cm.Data[fluentdConf], GenericComponentFluentdConfiguration, "config map data not equal to expected value")
}

// TestCreateHelidonFluentdConfigMap tests that a Fluentd config map is created for a Helidon component
// GIVEN a Helidon Fluentd configuration
//  WHEN I call CreateFluentdConfigMap
//  THEN the expected config map should be created
func TestCreateHelidonFluentdConfigMap(t *testing.T) {
	assertion := assert.New(t)

	labels := map[string]string{
		"myLabel": "hello",
	}
	cm := CreateFluentdConfigMap(HelidonFluentdConfiguration, "test-helidon", "test-namespace", labels)
	assertion.Equal("test-helidon-fluentd", cm.Name, "config map name not equal to expected value")
	assertion.Equal("test-namespace", cm.Namespace, "config map namespace not equal to expected value")
	assertion.Equal(labels, cm.Labels, "config map labels not equal to expected value")
	assertion.Equal(1, len(cm.Data), "config map data size not equal to expected value")
	assertion.NotNil(cm.Data[fluentdConf], "Expected fluentd.conf")
	assertion.Equal(cm.Data[fluentdConf], HelidonFluentdConfiguration, "config map data not equal to expected value")
}

// TestCreateFluentdContainer tests that a Fluentd container is created
// GIVEN a binding name and component name
//  WHEN I call CreateFluentdContainer
//  THEN the expected Fluentd container should be created
func TestCreateFluentdContainer(t *testing.T) {
	assertion := assert.New(t)

	fluentd := CreateFluentdContainer("test-binding", "test-component")

	assertion.Equal("fluentd", fluentd.Name, "Fluentd container name not equal to expected value")
	assertion.Equal(2, len(fluentd.Args), "Fluentd container args count not equal to expected value")
	assertion.Equal("-c", fluentd.Args[0], "Fluentd container arg not equal to expected value")
	assertion.Equal("/etc/fluent.conf", fluentd.Args[1], "Fluentd container arg not equal to expected value")
	assertion.Equal(7, len(fluentd.Env), "Fluentd container envs count not equal to expected value")
	assertion.Equal("APPLICATION_NAME", fluentd.Env[0].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("test-component", fluentd.Env[0].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("FLUENTD_CONF", fluentd.Env[1].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("fluentd.conf", fluentd.Env[1].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("FLUENT_ELASTICSEARCH_SED_DISABLE", fluentd.Env[2].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("true", fluentd.Env[2].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("ELASTICSEARCH_HOST", fluentd.Env[3].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("vmi-test-binding-es-ingest.verrazzano-system.svc.cluster.local", fluentd.Env[3].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("ELASTICSEARCH_PORT", fluentd.Env[4].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("9200", fluentd.Env[4].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("ELASTICSEARCH_USER", fluentd.Env[5].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal(constants.VmiSecretName, fluentd.Env[5].ValueFrom.SecretKeyRef.Name, "Fluentd container envs value from secret name not equal to expected value")
	assertion.Equal("username", fluentd.Env[5].ValueFrom.SecretKeyRef.Key, "Fluentd container envs value from secret key not equal to expected value")
	assertion.Equal(true, *fluentd.Env[5].ValueFrom.SecretKeyRef.Optional, "Fluentd container envs value from secret optional not equal to expected value")
	assertion.Equal(constants.VmiSecretName, fluentd.Env[5].ValueFrom.SecretKeyRef.Name, "Fluentd container envs value from secret name not equal to expected value")
	assertion.Equal("password", fluentd.Env[6].ValueFrom.SecretKeyRef.Key, "Fluentd container envs value from secret key not equal to expected value")
	assertion.Equal(true, *fluentd.Env[6].ValueFrom.SecretKeyRef.Optional, "Fluentd container envs value from secret optional not equal to expected value")
	assertion.Equal(3, len(fluentd.VolumeMounts), "Fluentd container volume mounts count not equal to expected value")
	assertion.Equal("/fluentd/etc/fluentd.conf", fluentd.VolumeMounts[0].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assertion.Equal("fluentd-config-volume", fluentd.VolumeMounts[0].Name, "Fluentd container volume mounts name not equal to expected value")
	assertion.Equal("fluentd.conf", fluentd.VolumeMounts[0].SubPath, "Fluentd container volume mounts sub path not equal to expected value")
	assertion.Equal(true, fluentd.VolumeMounts[0].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
	assertion.Equal("/var/log", fluentd.VolumeMounts[1].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assertion.Equal("varlog", fluentd.VolumeMounts[1].Name, "Fluentd container volume mounts name not equal to expected value")
	assertion.Equal(true, fluentd.VolumeMounts[1].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
	assertion.Equal("/u01/data/docker/containers", fluentd.VolumeMounts[2].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assertion.Equal("datadockercontainers", fluentd.VolumeMounts[2].Name, "Fluentd container volume mounts name not equal to expected value")
	assertion.Equal(true, fluentd.VolumeMounts[2].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")

	// In addition test that when Dev mode is set we get the expected elastic search host
	os.Setenv("USE_SYSTEM_VMI", "true")

	origLookupEnvFunc := util.LookupEnvFunc
	util.LookupEnvFunc = makeLookupEnvFunc("USE_SYSTEM_VMI", "true")
	defer func() { util.LookupEnvFunc = origLookupEnvFunc }()
	fluentd = CreateFluentdContainer("test-binding", "test-component")

	assertion.Equal("fluentd", fluentd.Name, "Fluentd container name not equal to expected value")
	assertion.Equal(2, len(fluentd.Args), "Fluentd container args count not equal to expected value")
	assertion.Equal("-c", fluentd.Args[0], "Fluentd container arg not equal to expected value")
	assertion.Equal("/etc/fluent.conf", fluentd.Args[1], "Fluentd container arg not equal to expected value")
	assertion.Equal(7, len(fluentd.Env), "Fluentd container envs count not equal to expected value")
	assertion.Equal("APPLICATION_NAME", fluentd.Env[0].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("test-component", fluentd.Env[0].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("FLUENTD_CONF", fluentd.Env[1].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("fluentd.conf", fluentd.Env[1].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("FLUENT_ELASTICSEARCH_SED_DISABLE", fluentd.Env[2].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("true", fluentd.Env[2].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("ELASTICSEARCH_HOST", fluentd.Env[3].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("vmi-"+constants.VmiSystemBindingName+"-es-ingest.verrazzano-system.svc.cluster.local", fluentd.Env[3].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("ELASTICSEARCH_PORT", fluentd.Env[4].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal("9200", fluentd.Env[4].Value, "Fluentd container envs value not equal to expected value")
	assertion.Equal("ELASTICSEARCH_USER", fluentd.Env[5].Name, "Fluentd container envs name not equal to expected value")
	assertion.Equal(constants.VmiSecretName, fluentd.Env[5].ValueFrom.SecretKeyRef.Name, "Fluentd container envs value from secret name not equal to expected value")
	assertion.Equal("username", fluentd.Env[5].ValueFrom.SecretKeyRef.Key, "Fluentd container envs value from secret key not equal to expected value")
	assertion.Equal(true, *fluentd.Env[5].ValueFrom.SecretKeyRef.Optional, "Fluentd container envs value from secret optional not equal to expected value")
	assertion.Equal(constants.VmiSecretName, fluentd.Env[5].ValueFrom.SecretKeyRef.Name, "Fluentd container envs value from secret name not equal to expected value")
	assertion.Equal("password", fluentd.Env[6].ValueFrom.SecretKeyRef.Key, "Fluentd container envs value from secret key not equal to expected value")
	assertion.Equal(true, *fluentd.Env[6].ValueFrom.SecretKeyRef.Optional, "Fluentd container envs value from secret optional not equal to expected value")
	assertion.Equal(3, len(fluentd.VolumeMounts), "Fluentd container volume mounts count not equal to expected value")
	assertion.Equal("/fluentd/etc/fluentd.conf", fluentd.VolumeMounts[0].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assertion.Equal("fluentd-config-volume", fluentd.VolumeMounts[0].Name, "Fluentd container volume mounts name not equal to expected value")
	assertion.Equal("fluentd.conf", fluentd.VolumeMounts[0].SubPath, "Fluentd container volume mounts sub path not equal to expected value")
	assertion.Equal(true, fluentd.VolumeMounts[0].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
	assertion.Equal("/var/log", fluentd.VolumeMounts[1].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assertion.Equal("varlog", fluentd.VolumeMounts[1].Name, "Fluentd container volume mounts name not equal to expected value")
	assertion.Equal(true, fluentd.VolumeMounts[1].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
	assertion.Equal("/u01/data/docker/containers", fluentd.VolumeMounts[2].MountPath, "Fluentd container volume mounts mount path not equal to expected value")
	assertion.Equal("datadockercontainers", fluentd.VolumeMounts[2].Name, "Fluentd container volume mounts name not equal to expected value")
	assertion.Equal(true, fluentd.VolumeMounts[2].ReadOnly, "Fluentd container volume mounts read only flag not equal to expected value")
}

// Returns a function with the same signature as os.LookupEnv, which has an override value for the
// given key
func makeLookupEnvFunc(givenKey string, overrideValue string) func(key string) (string, bool) {
	return func(key string) (string, bool) {
		if key == givenKey {
			return overrideValue, true
		}
		return os.LookupEnv(key)
	}
}

// TestCreateFluentdHostPathVolumes tests that a Fluentd host path volumes are created
//  WHEN I call CreateFluentdHostPathVolumes
//  THEN the expected Fluentd host path volumes should be created
func TestCreateFluentdHostPathVolumes(t *testing.T) {
	assertion := assert.New(t)

	volumes := CreateFluentdHostPathVolumes()

	assertion.Equal(2, len(volumes), "Fluentd container volumes count not equal to expected value")
	assertion.Equal("varlog", volumes[0].Name, "Fluentd volume name not equal to expected value")
	assertion.Equal("/var/log", volumes[0].VolumeSource.HostPath.Path, "Fluentd volume host path not equal to expected value")
	assertion.Equal("datadockercontainers", volumes[1].Name, "Fluentd volume name not equal to expected value")
	assertion.Equal("/u01/data/docker/containers", volumes[1].VolumeSource.HostPath.Path, "Fluentd volume host path not equal to expected value")
}

// TestCreateFluentdConfigMapVolume tests that a Fluentd config map volume is created
// GIVEN a component name
//  WHEN I call CreateFluentdConfigMapVolume
//  THEN the expected Fluentd config map volume should be created
func TestCreateFluentdConfigMapVolume(t *testing.T) {
	assertion := assert.New(t)

	volume := CreateFluentdConfigMapVolume("test-component")

	assertion.Equal("fluentd-config-volume", volume.Name, "Fluentd volume name not equal to expected value")
	assertion.Equal("test-component-fluentd", volume.VolumeSource.ConfigMap.Name, "Fluentd volume config map name not equal to expected value")
	assertion.Equal(int32(420), *volume.VolumeSource.ConfigMap.DefaultMode, "Fluentd volume config map default mode not equal to expected value")
}
