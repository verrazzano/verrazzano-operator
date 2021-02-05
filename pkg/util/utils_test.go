// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetManagedBindingLabels(t *testing.T) {
	assert := assert.New(t)
	const bindingName = "testbinding"
	binding := types.ResourceLocation{
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
	binding := types.ResourceLocation{
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
	binding := types.ResourceLocation{
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

	vzLocation := types.VerrazzanoLocation{
		Location: &types.ResourceLocation{},
		ManagedClusters: map[string]*types.ManagedCluster{
			cname1: {Name: cname1},
		},
	}
	mcMap := map[string]*ManagedClusterConnection{
		cname1: &mcc1,
		cname2: &mcc2,
	}

	results, err := GetManagedClustersForVerrazzanoBinding(&vzLocation, mcMap)
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

	vzLocation := types.VerrazzanoLocation{
		Location: &types.ResourceLocation{},
		ManagedClusters: map[string]*types.ManagedCluster{
			cname2: {Name: cname1},
		},
	}
	mcMap := map[string]*ManagedClusterConnection{
		cname1: &mcc1,
		cname2: &mcc2,
	}
	results := GetManagedClustersNotForVerrazzanoBinding(&vzLocation, mcMap)
	v, ok := results[cname2]
	assert.True(ok, "Missing map entry returned by GetManagedClustersForVerrazzanoBinding")
	assert.Equal(&mcc2, v)
	_, ok = results[cname1]
	assert.False(ok, "Map returned by GetManagedClustersForVerrazzanoBinding should not contain entry")
}

func TestSharedVMIDefault(t *testing.T) {
	assert := assert.New(t)
	os.Unsetenv("USE_SYSTEM_VMI")
	assert.False(SharedVMIDefault(), "Expected SharedVMIDefault() to return false when var not set")
	os.Setenv("USE_SYSTEM_VMI", "false")
	assert.False(SharedVMIDefault(), "Expected SharedVMIDefault() to return false when var set to false")
	os.Setenv("USE_SYSTEM_VMI", "boo")
	assert.False(SharedVMIDefault(), "Expected SharedVMIDefault() to return false when var set to bad value")
	os.Setenv("USE_SYSTEM_VMI", "true")
	assert.True(SharedVMIDefault(), "Expected SharedVMIDefault() to return true")
	os.Unsetenv("USE_SYSTEM_VMI")
}

func TestGetProfileBindingName(t *testing.T) {
	assert := assert.New(t)
	os.Unsetenv("USE_SYSTEM_VMI")
	assert.Equal("foobar", GetProfileBindingName("foobar"))
	os.Setenv("USE_SYSTEM_VMI", "true")
	assert.Equal(constants.VmiSystemBindingName, GetProfileBindingName("foobar"))
	os.Unsetenv("USE_SYSTEM_VMI")
}

func TestRemoveDuplicateValues(t *testing.T) {
	assert := assert.New(t)
	testSlice := []string{
		"abc",
		"def",
		"ghi",
		"def",
	}
	expectedOutput := []string{
		"abc",
		"def",
		"ghi",
	}
	assert.ElementsMatch(expectedOutput, RemoveDuplicateValues(testSlice))
}

func TestIsSystemProfileBindingName(t *testing.T) {
	assert := assert.New(t)
	assert.True(IsSystemProfileBindingName(constants.VmiSystemBindingName),
		"Expected IsSystemProfileBindingName() to return true")
	assert.False(IsSystemProfileBindingName("dummy"),
		"Expected IsSystemProfileBindingName() to return false for binding name dummy")
}
