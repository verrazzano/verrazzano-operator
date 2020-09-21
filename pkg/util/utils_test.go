package util

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"math"
	"testing"
)

func TestGetManagedBindingLabels(t *testing.T) {
	assert := assert.New(t)

	const bindingName = "testbinding"
	binding := v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:            bindingName,
		},
	}
	const clusterName = "testCluster"
	bm := GetManagedBindingLabels(&binding, clusterName)
	assert.NotNil(bm, bm)
	v, ok := bm[constants.K8SAppLabel]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.K8SAppLabel)
	assert.Equal(constants.VerrazzanoGroup, v)

	v, ok = bm[constants.VerrazzanoBinding]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.VerrazzanoBinding)
	assert.Equal(bindingName, v)

	v, ok = bm[constants.VerrazzanoCluster]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.VerrazzanoCluster)
	assert.Equal(clusterName, v)
}

func TestGetManagedLabelsNoBinding(t *testing.T) {
	assert := assert.New(t)

	const clusterName = "testCluster"
	bm := GetManagedLabelsNoBinding(clusterName)
	assert.NotNil(bm, bm)
	v, ok := bm[constants.K8SAppLabel]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.K8SAppLabel)
	assert.Equal(constants.VerrazzanoGroup, v)

	v, ok = bm[constants.VerrazzanoCluster]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.VerrazzanoCluster)
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
			Name:            bindingName,
		},
	}
	const clusterName = "testCluster"
	bm := GetLocalBindingLabels(&binding)
	assert.NotNil(bm, bm)
	v, ok := bm[constants.K8SAppLabel]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.K8SAppLabel)
	assert.Equal(constants.VerrazzanoGroup, v)

	v, ok = bm[constants.VerrazzanoBinding]
	assert.True(ok, "ManagedBindingLabels missing key for " + constants.VerrazzanoBinding)
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
	assert.Equal(fmt.Sprintf("vmi.%s.%s", bindingName, uri), GetVmiUri(bindingName, uri))
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
	assert.False (Contains(arr, foo),"Contains should return false")

	arr = []string{"",foo}
	assert.True (Contains(arr, foo),"Contains should return true")

	arr = []string{"", foo,bar}
	assert.True (Contains(arr, foo),"Contains should return true")
	assert.True (Contains(arr, bar),"Contains should return true")

	arr = nil
	assert.False (Contains(arr, foo),"Contains should return false")
}

func TestGetManagedClustersForVerrazzanoBinding(t *testing.T) {
	assert := assert.New(t)

	mcc1 := ManagedClusterConnection{}
	mcc2 := ManagedClusterConnection{}
	const cname1 = "cluster1"
	const cname2 = "cluster2"

	mbPair := types.ModelBindingPair{
		Model:            &v1beta1v8o.VerrazzanoModel{},
		Binding:          &v1beta1v8o.VerrazzanoBinding{},
		ManagedClusters:  map[string]*types.ManagedCluster{
			cname2:  &types.ManagedCluster{Name:cname2},
		},
	}
	mcMap := map[string]*ManagedClusterConnection{
		cname1: &mcc1,
		cname2: &mcc2,
	}

	results, err := GetManagedClustersForVerrazzanoBinding(&mbPair, mcMap)
	assert.NoError(err, "Error calling GetManagedClustersForVerrazzanoBinding")
	v,ok := results[cname2]
	assert.True(ok, "Missing map entry returned by GetManagedClustersForVerrazzanoBinding")
	assert.Equal(&mcc2, v)
}
