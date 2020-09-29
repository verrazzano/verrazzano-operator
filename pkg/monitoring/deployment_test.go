// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package monitoring

import (
	"fmt"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

type MockDeployment struct {
	namespace string
	deleted   string
}

func (mock *MockDeployment) DeleteDeployment(namespace, name string) error {
	mock.namespace = namespace
	mock.deleted = name
	return nil
}

// TestCreateAppServiceAccounts tests the creation of a VMI system deployment object.
// GIVEN a set of values, such as secrets and labels
//  WHEN I call GetSystemDeployments
//  THEN there should be a singe deployment objectcreated for the VMI system binding
//   AND that deployment should have the expected values
func TestCreateDeployment(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "cluster1"
	const url = "testURL"
	const payload = "match%5B%5D=%7Bjob%3D~%22" + constants.VmiSystemBindingName + "%2E%2A%22%7D"
	secrets, labels := getSecretsAndLabels()
	deps, err := GetSystemDeployments(clusterName, url, labels, secrets)

	// Assert the expected values
	assert.NoError(err, "Error getting deployments")
	assert.Len(deps, 1, "Expected 1 deployment")
	assert.Len(deps, 1, "Expected 1 deployment")
	dep := deps[0]
	assert.Equal(labels, dep.Labels, "Wrong list of labels")
	assert.Equal(pomPusherName(constants.VmiSystemBindingName), dep.Name, "Wrong deployment name")
	assert.Equal(constants.MonitoringNamespace, dep.Namespace, "Wrong namespace")
	assert.Equal(int32(1), *dep.Spec.Replicas, "Wrong replica count")
	assert.Equal(labels, dep.Spec.Selector.MatchLabels, "Wrong list of MatchLabels")
	assert.Equal(labels, dep.Spec.Template.Labels, "Wrong list of Template labels")
	assert.Len(dep.Spec.Template.Spec.Containers, 1, "Expected 1 element in container array")
	cont := dep.Spec.Template.Spec.Containers[0]
	assert.Equal("prometheus-pusher", cont.Name, "Wrong container name")
	assert.Equal(util.GetPromtheusPusherImage(), cont.Image, "Wrong container image")
	assert.Equal(corev1.PullIfNotPresent, cont.ImagePullPolicy, "Wrong container image pull policy")

	assert.Len(cont.Env, 9, "Expected 9 env vars")
	assert.Equal("PULL_URL_prometheus_pusher", cont.Env[0].Name, "Wrong env[0] name")
	assert.Equal("http://prometheus.istio-system.svc.cluster.local:9090/federate?"+payload, cont.Env[0].Value, "Wrong env[0] value")
	assert.Equal("PUSHGATEWAY_URL", cont.Env[1].Name, "Wrong env[1] name")
	assert.Equal(fmt.Sprintf("http://vmi-%s-prometheus-gw.%s.svc.cluster.local:9091", constants.VmiSystemBindingName,
		constants.VerrazzanoNamespace), cont.Env[1].Value, "Wrong env[1] value")
	assert.Equal("PUSHGATEWAY_USER", cont.Env[2].Name, "Wrong env[2] name")
	assert.Equal(constants.VmiUsername, cont.Env[2].Value, "Wrong env[2] value")
	assert.Equal("PUSHGATEWAY_PASSWORD", cont.Env[3].Name, "Wrong env[3] name")
	pw, _ := secrets.GetVmiPassword()
	assert.Equal(pw, cont.Env[3].Value, "Wrong env[3] value")
	assert.Equal("LOGLEVEL", cont.Env[4].Name, "Wrong env[4] name")
	assert.Equal("4", cont.Env[4].Value, "Wrong env[4] value")
	assert.Equal("SPLIT_SIZE", cont.Env[5].Name, "Wrong env[5] name")
	assert.Equal("1000", cont.Env[5].Value, "Wrong env[5] value")
	assert.Equal("no_proxy", cont.Env[6].Name, "Wrong env[6] name")
	assert.Equal("localhost,prometheus.istio-system.svc.cluster.local,127.0.0.1,/var/run/docker.sock", cont.Env[6].Value, "Wrong env[6] value")
	assert.Equal("NO_PROXY", cont.Env[7].Name, "Wrong env[7] name")
	assert.Equal("localhost,prometheus.istio-system.svc.cluster.local,127.0.0.1,/var/run/docker.sock", cont.Env[7].Value, "Wrong env[7] value")
	assert.Equal("PROM_CERT", cont.Env[8].Name, "Wrong env[8] name")
	assert.Equal("/verrazzano/certs/ca.crt", cont.Env[8].Value, "Wrong env[8] value")
	assert.Len(cont.Ports, 1, "Wrong number of container ports")
	assert.Equal("master", cont.Ports[0].Name, "Wrong container port name")
	assert.Equal(int32(9091), cont.Ports[0].ContainerPort, "Wrong container port value")
	assert.Len(cont.VolumeMounts, 1, "Wrong number of volume mounts")
	assert.Equal("cert-vol", cont.VolumeMounts[0].Name, "WrongVolumeMount name")
	assert.Equal("/verrazzano/certs", cont.VolumeMounts[0].MountPath, "WrongVolumeMount path")
	assert.Equal("/verrazzano/certs", cont.VolumeMounts[0].MountPath, "WrongVolumeMount path")
	assert.Equal(int64(1), *dep.Spec.Template.Spec.TerminationGracePeriodSeconds, "Wrong TerminationGracePeriodSeconds value")
	assert.Len(dep.Spec.Template.Spec.Volumes, 1, "Wrong number of volumes")
	assert.Equal("cert-vol", dep.Spec.Template.Spec.Volumes[0].Name, 1, "Wrong volume name")
	assert.Equal("system-tls", dep.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.SecretName, "Wrong volume secret name")
	assert.Equal(int32(420), *dep.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.DefaultMode,
		"Wrong volume secret name")
}

// TestCreateDeploymentMissingClusterName tests the error handling of GetSystemDeployments
// GIVEN an empty cluster name parameter
//  WHEN I call GetSystemDeployments
//  THEN an error should be returned
func TestCreateDeploymentMissingClusterName(t *testing.T) {
	assert := assert.New(t)
	const clusterName = ""
	const url = "url"
	secrets, labels := getSecretsAndLabels()
	deps, err := GetSystemDeployments(clusterName, url, labels, secrets)
	assert.Error(err, "Expected an error when cluster name is empty string")
	assert.Nil(deps, "Expected nil deployments when cluster name is empty string")
}

// TestCreateDeploymentMissingUrl tests the error handling of GetSystemDeployments
// GIVEN an empty URI parameter
//  WHEN I call GetSystemDeployments
//  THEN an error should be returned
func TestCreateDeploymentMissingUrl(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "cluster1"
	const url = ""
	secrets, labels := getSecretsAndLabels()
	deps, err := GetSystemDeployments(clusterName, url, labels, secrets)
	assert.Error(err, "Expected an error when url is empty string")
	assert.Nil(deps, "Expected nil deployments when url is empty string")
}

// TestDeletePomPusher tests the deletion of the POM pusher deployment
// GIVEN a POM pusher deployment
//  WHEN I call DeletePomPusher
//  THEN the deployment should get deleted
func TestDeletePomPusherDeployment(t *testing.T) {
	mock := &MockDeployment{}
	bindingName := "hello"
	pomPusher := pomPusherName(bindingName)
	DeletePomPusher(bindingName, mock)
	assert.Equal(t, constants.MonitoringNamespace, mock.namespace, "namespace")
	assert.Equal(t, pomPusher, mock.deleted, "deployment deleted")
}

// Get secrets and labels used for the tess
func getSecretsAndLabels() (secrets Secrets, labels map[string]string) {
	binding := v1beta1v8o.VerrazzanoBinding{}
	binding.Name = "vmiSecrets"
	vmiSecret := NewVmiSecret(&binding)
	secrets = &MockSecrets{secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}
	labels = map[string]string{"key1": "lable1", "key2": "label2"}
	return
}
