// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// TestCreateCrdDefinitions tests the creation of CRDs
// GIVEN a cluster which does not have any CRDs
//  WHEN I call CreateCrdDefinitions
//  THEN there should be exactly 3 CRDs created (coh, wls & helidon)
//   AND the property values of the CRDs should match the yaml descriptors used by the test
func TestCreateCrdDefinitions(t *testing.T) {
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	managedClusters := testutil.GetTestClusters()
	manifest := testutil.GetManifest()

	err := CreateCrdDefinitions(clusterConnection, &managedClusters[0], &manifest)
	if err != nil {
		t.Fatalf("got an error from CreateCrdDefinitions: %v", err)
	}
	// assert that the expected CRDs exist and have the expected property values
	assertExpectedCRDs(t, clusterConnection)
}

// TestCreateCrdDefinitionsUpdateAnExistingCRD tests creation of CRDs when an expected CRD already exists.
// The expectation is that the existing CRD will be updated with the properties from it's yaml descriptor.
// GIVEN a cluster which has one expected CRD (coh)
//  WHEN I call CreateCrdDefinitions
//  THEN there should be exactly 3 CRDs - 2 created (wls & helidon) and 1 updated (coh)
//   AND the property values of the CRDs should match the yaml descriptors used by the test
func TestCreateCrdDefinitionsUpdateAnExistingCRD(t *testing.T) {
	clusterConnections := testutil.GetManagedClusterConnections()
	clusterConnection := clusterConnections["cluster1"]
	managedClusters := testutil.GetTestClusters()
	manifest := testutil.GetManifest()

	// create an expected CRD with just a name
	// the expectation is that the call to CreateCrdDefinitions will update it with the properties from the CRD yaml
	crd := v1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "coh.test.crd.com",
		},
	}
	_, err := clusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), &crd, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("got an error creating a CRD: %v", err)
	}

	err = CreateCrdDefinitions(clusterConnection, &managedClusters[0], &manifest)
	if err != nil {
		t.Fatalf("got an error from CreateCrdDefinitions: %v", err)
	}
	// assert that the expected CRDs exist and have the expected property values
	assertExpectedCRDs(t, clusterConnection)
}

// assertExpectedCRDs asserts that the CRDs for the given cluster connection exist and have the expected property values
func assertExpectedCRDs(t *testing.T, clusterConnection *util.ManagedClusterConnection) {
	assert := assert.New(t)

	list, err := clusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("got an error listing CRDs: %v", err)
	}
	assert.Len(list.Items, 3, "expected exactly 3 CRDs")

	crd, err := clusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coh.test.crd.com", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error getting a CRD: %v", err)
	}
	assert.Equal("coh.test.crd.com", crd.Name)
	assert.Equal("Coh", crd.Spec.Names.Kind)
	assert.Equal("CohList", crd.Spec.Names.ListKind)
	assert.Equal("coh", crd.Spec.Names.Singular)
	assert.Equal("cohs", crd.Spec.Names.Plural)

	crd, err = clusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "wls.test.crd.com", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error getting a CRD: %v", err)
	}
	assert.Equal("wls.test.crd.com", crd.Name)
	assert.Equal("Wls", crd.Spec.Names.Kind)
	assert.Equal("WlsList", crd.Spec.Names.ListKind)
	assert.Equal("wls", crd.Spec.Names.Singular)
	assert.Equal("wlss", crd.Spec.Names.Plural)

	crd, err = clusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "helidon.test.crd.com", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("got an error getting a CRD: %v", err)
	}
	assert.Equal("helidon.test.crd.com", crd.Name)
	assert.Equal("Helidon", crd.Spec.Names.Kind)
	assert.Equal("HelidonList", crd.Spec.Names.ListKind)
	assert.Equal("helidon", crd.Spec.Names.Singular)
	assert.Equal("helidons", crd.Spec.Names.Plural)
}
