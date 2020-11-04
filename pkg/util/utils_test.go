// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetManagedBindingLabels(t *testing.T) {
	assert := assert.New(t)
	const bindingName = "testbinding"
	binding := v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: bindingName,
		},
	}
	const clusterName = "testCluster"
	bm := GetManagedBindingLabels(&binding, clusterName)
	assert.NotNil(bm, bm)
	v, ok := bm[constants.K8SAppLabel]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.K8SAppLabel)
	assert.Equal(constants.VerrazzanoGroup, v)

	v, ok = bm[constants.VerrazzanoBinding]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.VerrazzanoBinding)
	assert.Equal(bindingName, v)

	v, ok = bm[constants.VerrazzanoCluster]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.VerrazzanoCluster)
	assert.Equal(clusterName, v)
}

func TestGetManagedLabelsNoBinding(t *testing.T) {
	assert := assert.New(t)
	const clusterName = "testCluster"
	bm := GetManagedLabelsNoBinding(clusterName)
	assert.NotNil(bm, bm)
	v, ok := bm[constants.K8SAppLabel]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.K8SAppLabel)
	assert.Equal(constants.VerrazzanoGroup, v)

	v, ok = bm[constants.VerrazzanoCluster]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.VerrazzanoCluster)
	assert.Equal(clusterName, v)
}

func TestGetManagedNamespaceForBinding(t *testing.T) {
	assert := assert.New(t)
	const bindingName = "testbinding"
	binding := v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: bindingName,
		},
	}
	name := GetManagedNamespaceForBinding(&binding)
	assert.Equal(fmt.Sprintf("%s-%s", constants.VerrazzanoPrefix, binding.Name), name)
}

func TestGetLocalBindingLabels(t *testing.T) {
	assert := assert.New(t)
	const bindingName = "testbinding"
	binding := v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: bindingName,
		},
	}
	const clusterName = "testCluster"
	bm := GetLocalBindingLabels(&binding)
	assert.NotNil(bm, bm)
	v, ok := bm[constants.K8SAppLabel]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.K8SAppLabel)
	assert.Equal(constants.VerrazzanoGroup, v)

	v, ok = bm[constants.VerrazzanoBinding]
	assert.True(ok, "ManagedBindingLabels missing key for "+constants.VerrazzanoBinding)
	assert.Equal(bindingName, v)
}

func TestGetManagedClusterNamespaceForSystem(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(constants.VerrazzanoSystem, GetManagedClusterNamespaceForSystem())
}

func TestGetVmiNameForBinding(t *testing.T) {
	assert := assert.New(t)
	const bindingName = "testbinding"
	assert.Equal(bindingName, GetVmiNameForBinding(bindingName))
}

func TestGetGetVmiUri(t *testing.T) {
	assert := assert.New(t)
	const bindingName = "testbinding"
	const uri = "vzURI"
	assert.Equal(fmt.Sprintf("vmi.%s.%s", bindingName, uri), GetVmiURI(bindingName, uri))
}

func TestGetServiceAccountNameForSystem(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(constants.VerrazzanoSystem, GetServiceAccountNameForSystem())
}

func TestNewVal(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(int32(math.MaxInt32), *NewVal(math.MaxInt32))
}

func TestNew64Val(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(int64(math.MaxInt64), *New64Val(math.MaxInt64))
}

func TestContains(t *testing.T) {
	assert := assert.New(t)
	const foo = "foo"
	const bar = "bar"

	arr := []string{""}
	assert.False(Contains(arr, foo), "Contains should return false")

	arr = []string{"", foo}
	assert.True(Contains(arr, foo), "Contains should return true")

	arr = []string{"", foo, bar}
	assert.True(Contains(arr, foo), "Contains should return true")
	assert.True(Contains(arr, bar), "Contains should return true")

	arr = nil
	assert.False(Contains(arr, foo), "Contains should return false")
}

func TestGetManagedClustersForVerrazzanoBinding(t *testing.T) {
	assert := assert.New(t)
	mcc1 := ManagedClusterConnection{}
	mcc2 := ManagedClusterConnection{}
	const cname1 = "cluster1"
	const cname2 = "cluster2"

	mbPair := types.ModelBindingPair{
		Model:   &v1beta1v8o.VerrazzanoModel{},
		Binding: &v1beta1v8o.VerrazzanoBinding{},
		ManagedClusters: map[string]*types.ManagedCluster{
			cname1: {Name: cname1},
		},
	}
	mcMap := map[string]*ManagedClusterConnection{
		cname1: &mcc1,
		cname2: &mcc2,
	}

	results, err := GetManagedClustersForVerrazzanoBinding(&mbPair, mcMap)
	assert.NoError(err, "Error calling GetManagedClustersForVerrazzanoBinding")
	v, ok := results[cname1]
	assert.True(ok, "Missing map entry returned by GetManagedClustersForVerrazzanoBinding")
	assert.Equal(&mcc1, v)
	_, ok = results[cname2]
	assert.False(ok, "Map returned by GetManagedClustersForVerrazzanoBinding should not contain entry")
}

func TestGetManagedClustersNotForVerrazzanoBinding(t *testing.T) {
	assert := assert.New(t)
	mcc1 := ManagedClusterConnection{}
	mcc2 := ManagedClusterConnection{}
	const cname1 = "cluster1"
	const cname2 = "cluster2"

	mbPair := types.ModelBindingPair{
		Model:   &v1beta1v8o.VerrazzanoModel{},
		Binding: &v1beta1v8o.VerrazzanoBinding{},
		ManagedClusters: map[string]*types.ManagedCluster{
			cname2: {Name: cname1},
		},
	}
	mcMap := map[string]*ManagedClusterConnection{
		cname1: &mcc1,
		cname2: &mcc2,
	}
	results := GetManagedClustersNotForVerrazzanoBinding(&mbPair, mcMap)
	v, ok := results[cname2]
	assert.True(ok, "Missing map entry returned by GetManagedClustersForVerrazzanoBinding")
	assert.Equal(&mcc2, v)
	_, ok = results[cname1]
	assert.False(ok, "Map returned by GetManagedClustersForVerrazzanoBinding should not contain entry")
}

func TestIsClusterInBinding(t *testing.T) {
	assert := assert.New(t)
	const cname1 = "cluster1"
	const cname2 = "cluster2"
	mbMap := map[string]*types.ModelBindingPair{
		cname1: {
			Binding: &v1beta1v8o.VerrazzanoBinding{
				Spec: v1beta1v8o.VerrazzanoBindingSpec{
					Placement: []v1beta1v8o.VerrazzanoPlacement{{Name: cname1}},
				},
				Status: v1beta1v8o.VerrazzanoBindingStatus{},
			},
		},
	}
	assert.True(IsClusterInBinding(cname1, mbMap))
	assert.False(IsClusterInBinding(cname2, mbMap))
}

func TestGetComponentNamespace(t *testing.T) {
	assert := assert.New(t)
	const ns1 = "ns1"
	const ns2 = "ns2"
	const compname1 = "comp1"
	const compname2 = "comp2"
	const compname3 = "comp3"
	binding := &v1beta1v8o.VerrazzanoBinding{
		Spec: v1beta1v8o.VerrazzanoBindingSpec{
			Placement: []v1beta1v8o.VerrazzanoPlacement{
				{Namespaces: []v1beta1v8o.KubernetesNamespace{
					{Name: ns1,
						Components: []v1beta1v8o.BindingComponent{{
							Name: compname1}},
					},
					{Name: ns2,
						Components: []v1beta1v8o.BindingComponent{{
							Name: compname2}},
					},
				}},
			},
		},
		Status: v1beta1v8o.VerrazzanoBindingStatus{},
	}
	ns, err := GetComponentNamespace(compname1, binding)
	assert.NoError(err, "Error finding component in GetComponentNamespace")
	assert.Equal(ns1, ns)
	ns, err = GetComponentNamespace(compname2, binding)
	assert.NoError(err, "Error finding component in GetComponentNamespace")
	assert.Equal(ns2, ns)
	_, err = GetComponentNamespace(compname3, binding)
	assert.Error(err, "Error finding component in GetComponentNamespace. Component should not be found")
}

func TestLoadManifest(t *testing.T) {
	assert := assert.New(t)
	manifest, err := LoadManifest()
	assert.NotNil(manifest, "LoadManifiest returned nil")
	assert.Error(err, "Error loading manifest")
}

func TestIsDevProfile(t *testing.T) {
	assert := assert.New(t)

	assert.False(IsDevProfile(), "Expected DevProfile to return false")
	os.Setenv("SINGLE_SYSTEM_VMI", "true")
	assert.True(IsDevProfile(), "Expected DevProfile to return true")
}
