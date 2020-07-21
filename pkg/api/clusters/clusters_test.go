// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	listers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/listers/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var cluster1Spec = v1beta1.VerrazzanoManagedClusterSpec{Type: "testCluster", ServerAddress: "test.com", Description: "Test Cluster"}
var cluster1MetaInfo = metav1.ObjectMeta{UID: "123-456-789", Name: "cluster1", Namespace: "default"}
var cluster1 = v1beta1.VerrazzanoManagedCluster{ObjectMeta: cluster1MetaInfo, Spec: cluster1Spec}

var cluster2Spec = v1beta1.VerrazzanoManagedClusterSpec{Type: "testCluster2", ServerAddress: "test.com", Description: "Test Cluster 2"}
var cluster2MetaInfo = metav1.ObjectMeta{UID: "123-456-789-2", Name: "cluster2", Namespace: "default"}
var cluster2 = v1beta1.VerrazzanoManagedCluster{ObjectMeta: cluster2MetaInfo, Spec: cluster2Spec}

// cluster from non 'default' namespace shouldn't be returned by list operation
var cluster3Spec = v1beta1.VerrazzanoManagedClusterSpec{Type: "testCluster3", ServerAddress: "test.com", Description: "Test Cluster 3"}
var cluster3MetaInfo = metav1.ObjectMeta{UID: "123-456-789-3", Name: "cluster3", Namespace: "default3"}
var cluster3 = v1beta1.VerrazzanoManagedCluster{ObjectMeta: cluster3MetaInfo, Spec: cluster3Spec}

func TestReturnAllClusters(t *testing.T) {
	initClusters()

	request, _ := http.NewRequest("GET", "/clusters", nil)
	responseRecorder := invokeHandler(request, "/clusters", ReturnAllClusters)

	assertEqual("2", responseRecorder.Header().Get("X-Total-Count"), "X-Total-Count header", t)

	var responseClusters []TestCluster
	json.Unmarshal(responseRecorder.Body.Bytes(), &responseClusters)

	assertClusters(responseClusters,
		map[string]v1beta1.VerrazzanoManagedCluster{
			cluster1.Name: cluster1,
			cluster2.Name: cluster2,
		}, t)
}

func TestReturnSingleCluster(t *testing.T) {
	initClusters()

	request, _ := http.NewRequest("GET", "/clusters/"+string(cluster1.UID), nil)
	responseRecorder := invokeHandler(request, "/clusters/{id}", ReturnSingleCluster)

	responseCluster := &TestCluster{}
	json.Unmarshal(responseRecorder.Body.Bytes(), responseCluster)

	assertCluster(NewTestCluster(cluster1), *responseCluster, t)
}

type TestCluster struct {
	Id            string
	Name          string
	Type          string
	ServerAddress string
	Description   string
	Status        string
}

func NewTestCluster(managedCluster v1beta1.VerrazzanoManagedCluster) *TestCluster {
	return &TestCluster{
		Id:            string(managedCluster.UID),
		Name:          managedCluster.Name,
		Type:          managedCluster.Spec.Type,
		ServerAddress: managedCluster.Spec.ServerAddress,
		Description:   managedCluster.Spec.Description,
		Status:        "OK",
	}
}

func invokeHandler(request *http.Request, path string, handler func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	responseRecorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc(path, handler)
	router.ServeHTTP(responseRecorder, request)

	return responseRecorder
}

func assertClusters(actualClusters []TestCluster, expectedClusters map[string]v1beta1.VerrazzanoManagedCluster, t *testing.T) {
	if len(actualClusters) != len(expectedClusters) {
		t.Errorf("Expected %d clusters to be returned but received %d clusters.", len(expectedClusters), len(actualClusters))
	}

	for _, cluster := range actualClusters {
		expectedCluster, expected := expectedClusters[cluster.Name]
		if !expected {
			t.Errorf("Cluster %s was returned but wasn't expected", cluster.Name)
		} else {
			assertCluster(NewTestCluster(expectedCluster), cluster, t)
		}
	}
}

func assertCluster(expected *TestCluster, actual TestCluster, t *testing.T) {
	assertEqual(expected.Id, actual.Id, "id", t)
	assertEqual(expected.Name, actual.Name, "name", t)
	assertEqual(expected.Type, actual.Type, "type", t)
	assertEqual(expected.ServerAddress, actual.ServerAddress, "serverAddress", t)
	assertEqual(expected.Description, actual.Description, "description", t)
	assertEqual(expected.Status, actual.Status, "OK", t)
}

func assertEqual(expected string, actual string, field string, t *testing.T) {
	if expected != actual {
		t.Errorf("Incorrect %s provided. Expected %s but was %s", field, expected, actual)
	}
}

func initClusters() {
	indexer := cache.NewIndexer(testKeyFunc, testIndexers)
	indexer.Add(&cluster1)
	indexer.Add(&cluster2)
	indexer.Add(&cluster3)

	managedClusterLister := listers.NewVerrazzanoManagedClusterLister(indexer)
	clusterListers := controller.Listers{ManagedClusterLister: &managedClusterLister}
	// init clusters
	Init(clusterListers)
}

var testIndexFunc cache.IndexFunc = func(obj interface{}) ([]string, error) {
	switch t := obj.(type) {
	case *metav1.ObjectMeta:
		return []string{obj.(*metav1.ObjectMeta).Namespace}, nil
	case *v1beta1.VerrazzanoManagedCluster:
		return []string{obj.(*v1beta1.VerrazzanoManagedCluster).Namespace}, nil
	default:
		fmt.Printf("Unknown Type %T", t)
		return nil, errors.New("unknown Type")
	}
}

var testKeyFunc cache.KeyFunc = func(obj interface{}) (string, error) {
	return string(obj.(*v1beta1.VerrazzanoManagedCluster).UID), nil
}

var testIndexers = map[string]cache.IndexFunc{
	"namespace": testIndexFunc,
}
