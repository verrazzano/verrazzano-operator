// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsdom

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	vz "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
)

// TestIntrospectorJobActiveDeadlineSecondsWithDefaultValue tests using default value for introspector job active deadline.
// GIVEN a default Verrazzano weblogic domain, model and binding
// WHEN a WLS domain CR is created without an override
// THEN verify the correct default value is used for IntrospectorJobActiveDeadlineSeconds
func TestIntrospectorJobActiveDeadlineSecondsWithDefaultValue(t *testing.T) {
	vzWeblogicDomain := createWeblogicDomainModel("testDomain", true)
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec:       vz.VerrazzanoBindingSpec{},
			ObjectMeta: metav1.ObjectMeta{Name: vzWeblogicDomain.Name},
		},
		VerrazzanoURI: "test.v8o.xyz.com",
	}
	labels := make(map[string]string)
	secrets := []string{}
	wlsDomainCR := CreateWlsDomainCR("testNamespace", vzWeblogicDomain, &mbPair, labels, "testConfigMap", secrets)
	assert.Equal(t, 600, wlsDomainCR.Spec.Configuration.IntrospectorJobActiveDeadlineSeconds, "Expect default value to be used.")
}

// TestIntrospectorJobActiveDeadlineSecondsWithOverrideValue tests using override value for introspector job active deadline.
// GIVEN a Verrazzano weblogic domain, model and binding with an override value for IntrospectorJobActiveDeadlineSeconds
// WHEN a WLS domain CR is created with an override for IntrospectorJobActiveDeadlineSeconds
// THEN verify the correct value is used for IntrospectorJobActiveDeadlineSeconds
func TestIntrospectorJobActiveDeadlineSecondsWithOverrideValue(t *testing.T) {
	vzWeblogicDomain := createWeblogicDomainModel("testDomain", true)
	vzWeblogicDomain.DomainCRValues.Configuration.IntrospectorJobActiveDeadlineSeconds = 900
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec:       vz.VerrazzanoBindingSpec{},
			ObjectMeta: metav1.ObjectMeta{Name: vzWeblogicDomain.Name},
		},
		VerrazzanoURI: "test.v8o.xyz.com",
	}
	labels := make(map[string]string)
	secrets := []string{}
	wlsDomainCR := CreateWlsDomainCR("testNamespace", vzWeblogicDomain, &mbPair, labels, "testConfigMap", secrets)
	assert.Equal(t, 900, wlsDomainCR.Spec.Configuration.IntrospectorJobActiveDeadlineSeconds, "Expect override value to be used.")
}

// Test using default value for enabling Fluentd
func TestFluentdEnabledDefault(t *testing.T) {
	namespace, domainName := "myNs", "myDomain"
	vzWeblogicDomain := createWeblogicDomainModel(domainName, true)
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec:       vz.VerrazzanoBindingSpec{},
			ObjectMeta: metav1.ObjectMeta{Name: domainName},
		},
		VerrazzanoURI: "test.v8o.xyz.com",
	}
	mbPair.Binding.Spec.WeblogicBindings = []vz.VerrazzanoWeblogicBinding{{Name: domainName}}
	labels := make(map[string]string)

	useSystemVmi := false
	weblogicDomain := CreateWlsDomainCR(namespace, vzWeblogicDomain, &mbPair, labels, "", nil)
	checkDomainModel(t, weblogicDomain, domainName, useSystemVmi)

	// Repeat using the system VMI and re-check the fluentd config (and entire domain model)
	origLookupEnvFunc := util.LookupEnvFunc
	defer func() { util.LookupEnvFunc = origLookupEnvFunc }()
	util.LookupEnvFunc = func(key string) (string, bool) {
		if key == "USE_SYSTEM_VMI" {
			return "true", true
		}
		return origLookupEnvFunc(key)
	}
	useSystemVmi = true
	systemVmiWeblogicDomain := CreateWlsDomainCR(namespace, vzWeblogicDomain, &mbPair, labels, "", nil)
	checkDomainModel(t, systemVmiWeblogicDomain, domainName, useSystemVmi)
}

// Test arbitrary domain secrets
func TestDomainSecrets(t *testing.T) {
	namespace, domainName := "myNs", "myDomain"
	vzWeblogicDomain := createWeblogicDomainModel(domainName, true)
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec:       vz.VerrazzanoBindingSpec{},
			ObjectMeta: metav1.ObjectMeta{Name: domainName},
		},
		VerrazzanoURI: "test.v8o.xyz.com",
	}
	mbPair.Binding.Spec.WeblogicBindings = []vz.VerrazzanoWeblogicBinding{{Name: domainName}}
	labels := make(map[string]string)

	testSecret1 := "test-secret-1"
	testSecret2 := "test-secret-2"
	vzWeblogicDomain.DomainCRValues.Configuration = v8weblogic.Configuration{
		Secrets: []string{testSecret1, testSecret2},
	}
	weblogicDomain := CreateWlsDomainCR(namespace, vzWeblogicDomain, &mbPair, labels, "", nil)
	checkDomainModel(t, weblogicDomain, domainName, false)

	assert.Equal(t, 2, len(weblogicDomain.Spec.Configuration.Secrets), fmt.Sprint("Expected 2 secrets"))
	assert.Equal(t, testSecret1, weblogicDomain.Spec.Configuration.Secrets[0], fmt.Sprintf("Expected secret to be %s", testSecret1))
	assert.Equal(t, testSecret2, weblogicDomain.Spec.Configuration.Secrets[1], fmt.Sprintf("Expected secret to be %s", testSecret2))
}

// Test DB secrets
func TestDbSecrets(t *testing.T) {
	namespace, domainName := "myNs", "myDomain"
	vzWeblogicDomain := createWeblogicDomainModel(domainName, true)
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec:       vz.VerrazzanoBindingSpec{},
			ObjectMeta: metav1.ObjectMeta{Name: domainName},
		},
		VerrazzanoURI: "test.v8o.xyz.com",
	}
	mbPair.Binding.Spec.WeblogicBindings = []vz.VerrazzanoWeblogicBinding{{Name: domainName}}
	labels := make(map[string]string)

	testDbSecret := "test-db-secret"
	dbSecrets := []string{testDbSecret}
	weblogicDomain := CreateWlsDomainCR(namespace, vzWeblogicDomain, &mbPair, labels, "", dbSecrets)
	checkDomainModel(t, weblogicDomain, domainName, false)

	assert.Equal(t, 1, len(weblogicDomain.Spec.Configuration.Secrets), fmt.Sprint("Expected 1 secret"))
	assert.Equal(t, testDbSecret, weblogicDomain.Spec.Configuration.Secrets[0], fmt.Sprintf("Expected secret to be %s", testDbSecret))
}

// Test arbitrary domain secrets with DB secrets
func TestDomainSecretsWithDbSecrets(t *testing.T) {
	namespace, domainName := "myNs", "myDomain"
	vzWeblogicDomain := createWeblogicDomainModel(domainName, true)
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec:       vz.VerrazzanoBindingSpec{},
			ObjectMeta: metav1.ObjectMeta{Name: domainName},
		},
		VerrazzanoURI: "test.v8o.xyz.com",
	}
	mbPair.Binding.Spec.WeblogicBindings = []vz.VerrazzanoWeblogicBinding{{Name: domainName}}
	labels := make(map[string]string)

	testSecret1 := "test-secret-1"
	testSecret2 := "test-secret-2"
	vzWeblogicDomain.DomainCRValues.Configuration = v8weblogic.Configuration{
		Secrets: []string{testSecret1, testSecret2},
	}
	testDbSecret := "test-db-secret"
	dbSecrets := []string{testDbSecret}
	weblogicDomain := CreateWlsDomainCR(namespace, vzWeblogicDomain, &mbPair, labels, "", dbSecrets)
	checkDomainModel(t, weblogicDomain, domainName, false)

	assert.Equal(t, 3, len(weblogicDomain.Spec.Configuration.Secrets), fmt.Sprint("Expected 3 secrets"))
	assert.Equal(t, testSecret1, weblogicDomain.Spec.Configuration.Secrets[0], fmt.Sprintf("Expected secret to be %s", testSecret1))
	assert.Equal(t, testSecret2, weblogicDomain.Spec.Configuration.Secrets[1], fmt.Sprintf("Expected secret to be %s", testSecret2))
	assert.Equal(t, testDbSecret, weblogicDomain.Spec.Configuration.Secrets[2], fmt.Sprintf("Expected secret to be %s", testDbSecret))
}

func createWeblogicDomainModel(name string, fluentd bool) vz.VerrazzanoWebLogicDomain {
	return vz.VerrazzanoWebLogicDomain{
		Name:           name,
		AdminPort:      80,
		T3Port:         81,
		FluentdEnabled: fluentd,
		DomainCRValues: v8weblogic.DomainSpec{
			DomainUID: name,
		},
	}
}

func checkDomainModel(t *testing.T, weblogicDomain *v8weblogic.Domain, domainName string, useSystemVmi bool) {
	containers := weblogicDomain.Spec.ServerPod.Containers
	volumes := weblogicDomain.Spec.ServerPod.Volumes

	assert.Equal(t, domainName, weblogicDomain.Name, fmt.Sprintf("Expected domain name to be %s", domainName))
	assert.Equal(t, domainName, weblogicDomain.Spec.DomainUID, fmt.Sprintf("Expected domain UID to be %s", domainName))
	assert.Equal(t, 1, len(containers), "Expected containers count to be 1")
	assert.Equal(t, 2, len(containers[0].Args), "Expected args count to be 2")
	name := "fluentd"
	assert.Equal(t, name, containers[0].Name, fmt.Sprintf("Expect container name to be %s", name))
	name = "-c"
	assert.Equal(t, name, containers[0].Args[0], fmt.Sprintf("Expect fist argument to be %s", name))
	name = "/etc/fluent.conf"
	assert.Equal(t, name, containers[0].Args[1], fmt.Sprintf("Expect second argument to be %s", name))
	assert.Equal(t, util.GetFluentdImage(), containers[0].Image, fmt.Sprintf("Expect image name to be %s", util.GetFluentdImage()))
	assert.Equal(t, 2, len(containers[0].VolumeMounts), "Expected volume mounts count to be 2")
	name = "fluentd-config-volume"
	assert.Equal(t, name, containers[0].VolumeMounts[0].Name, fmt.Sprintf("Expect volume mount name to be %s", name))
	name = "/fluentd/etc/fluentd.conf"
	assert.Equal(t, name, containers[0].VolumeMounts[0].MountPath, fmt.Sprintf("Expect volume mount path to be %s", name))
	name = "fluentd.conf"
	assert.Equal(t, name, containers[0].VolumeMounts[0].SubPath, fmt.Sprintf("Expect volume mount subpath to be %s", name))
	assert.Equal(t, true, containers[0].VolumeMounts[0].ReadOnly, "Expect volume mount to be readOnly")
	name = "weblogic-domain-storage-volume"
	assert.Equal(t, name, containers[0].VolumeMounts[1].Name, fmt.Sprintf("Expect volume mount name to be %s", name))
	name = "/scratch"
	assert.Equal(t, name, containers[0].VolumeMounts[1].MountPath, fmt.Sprintf("Expect volume mount path to be %s", name))
	assert.Equal(t, true, containers[0].VolumeMounts[1].ReadOnly, "Expect volume mount to be readOnly")

	checkFluentdEnv(t, containers[0], domainName, useSystemVmi)
	assert.Equal(t, 2, len(volumes), "Expected volumes count to be 2")
	name = "weblogic-domain-storage-volume"
	assert.Equal(t, name, volumes[0].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = "fluentd-config-volume"
	assert.Equal(t, name, volumes[1].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = "fluentd-config"
	assert.Equal(t, name, volumes[1].VolumeSource.ConfigMap.Name, fmt.Sprintf("Expected volume configmap name to be %s", name))
}

func checkFluentdEnv(t *testing.T, fluentdContainer corev1.Container, domainName string, useSystemVmi bool) {
	assert.Equal(t, 10, len(fluentdContainer.Env), "Expected env count to be 10")
	esHostVmiSuffix := domainName
	if useSystemVmi {
		esHostVmiSuffix = "system"
	}
	expectedEsHost := "vmi-" + esHostVmiSuffix + "-es-ingest.verrazzano-system.svc.cluster.local"

	for _, envVar := range fluentdContainer.Env {
		if envVar.Name == "ELASTICSEARCH_HOST" {
			assert.Equal(t, expectedEsHost, envVar.Value)
			break
		}
	}
}

func TestSliceUniqMap(t *testing.T) {
	input := []string{"ant", "cat", "dog", "ant", "anteater", "cow"}

	output := SliceUniqMap(input)

	if !(len(output) == 5 &&
		contains(output, "ant") &&
		contains(output, "cat") &&
		contains(output, "dog") &&
		contains(output, "anteater") &&
		contains(output, "cow")) {
		t.Errorf("SliceUniqMap was incorrect, got: %s, wanted %s", output, []string{"ant", "cat", "dog", "anteater", "cow"})
	}
}

// Contains returns true if a slice element has an exact match to the input string
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
