// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsdom

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vz "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
)

// Test using default value for enabling Fluentd
func TestFluentdEnabledDefault(t *testing.T) {
	namespace, domainName := "myNs", "myDomain"
	vzWeblogicDomain := createWeblogicDomainModel(domainName, true)
	mbPair := types.ModelBindingPair{
		Binding: &vz.VerrazzanoBinding{
			Spec: vz.VerrazzanoBindingSpec{},
		},
		VerrazzanoUri: "test.v8o.xyz.com",
		SslVerify:     true,
	}
	mbPair.Binding.Spec.WeblogicBindings = []vz.VerrazzanoWeblogicBinding{{Name: domainName}}
	labels := make(map[string]string)
	weblogicDomain := CreateWlsDomainCR(namespace, vzWeblogicDomain, &mbPair, labels, "", nil)
	checkDomainModel(t, weblogicDomain, domainName)
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

func checkDomainModel(t *testing.T, weblogicDomain *v8weblogic.Domain, domainName string) {
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
	assert.Equal(t, 9, len(containers[0].Env), "Expected env count to be 9")

	assert.Equal(t, 2, len(volumes), "Expected volumes count to be 2")
	name = "weblogic-domain-storage-volume"
	assert.Equal(t, name, volumes[0].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = "fluentd-config-volume"
	assert.Equal(t, name, volumes[1].Name, fmt.Sprintf("Expected volume name to be %s", name))
	name = "fluentd-config"
	assert.Equal(t, name, volumes[1].VolumeSource.ConfigMap.Name, fmt.Sprintf("Expected volume configmap name to be %s", name))
}
