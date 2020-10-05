// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package domains

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	v8weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestInit ensures that a set of domains can be obtained from the provided set of listers.
func TestInit(t *testing.T) {
	// GIVEN an array of clusters and a model binding pair
	Domains = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": getModelBindingPair(),
	}
	// create the tests
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testInit",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	// run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// WHEN a WLS domain is appended to the cluster and the Init() method is invoked
			createAndInsertWLSDomain("cluster1", "test-weblogic", tt.listers)
			Init(tt.listers)
			// THEN the single domain with the expected name will be found in the Domains slice
			assert.Equal(t, tt.listers, listerSet, "Wrong number of listers")
			assert.Equal(t, 1, len(Domains), "Wrong number of Domains")
			assert.Equal(t, "test3", Domains[0].Namespace, "Domain has the wrong namespace")
		})
	}
}

//TestReturnNoDomains ensures that an empty response is returned if no domains are available
func TestReturnNoDomains(t *testing.T) {
	// GIVEN an empty model binding pair
	Domains = nil
	emptyModelBindingPairs := make(map[string]*types.ModelBindingPair)
	Init(controller.Listers{
		ModelBindingPairs: &emptyModelBindingPairs,
	})
	// setup http request processing
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/domains", nil)
	// WHEN an HTTP request for the list of domains is submitted
	ReturnAllDomains(response, request)
	// THEN a HTTP OK response with an empty array body will be returned
	assert.Equal(t, http.StatusOK, response.Code, "wrong response code")
	var domains []Domain
	err := json.Unmarshal(response.Body.Bytes(), &domains)
	if err != nil {
		t.Errorf("Error while reading response JSON: %s", err)
	}
	// assert empty response
	assert.Empty(t, domains, "response should have no domains")
}

//TestReturnAllDomains ensures that an array of domains is returned
func TestReturnAllDomains(t *testing.T) {
	Domains = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": getModelBindingPair(),
	}
	// setup the tests
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testReturnAllDomains",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	// run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN an array of clusters and a model binding pair that includes a reference to a WLS Domain
			createAndInsertWLSDomain("cluster1", "test-weblogic", tt.listers)
			createAndInsertWLSDomain("cluster2", "test-coherence", tt.listers)
			Init(tt.listers)
			// setup http request processing
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/domains", nil)
			// WHEN a request is submitted for a listing of domains
			ReturnAllDomains(response, request)
			// THEN the response with have an HTTP OK status (200) with a JSON array of domains in the response body
			assert.Equal(t, http.StatusOK, response.Code, "wrong response code")
			var domains []Domain
			err := json.Unmarshal(response.Body.Bytes(), &domains)
			if err != nil {
				t.Errorf("Error while reading response JSON: %s", err)
			}
			// Validate that there is the correct number of domains, and the
			// domains have the expected namespaces
			assert.Equal(t, 2, len(domains), "Wrong number of domains in the list")
			assert.True(t, foundByID(domains, "test-weblogic"), "Domain has the wrong ID")
			assert.True(t, foundByID(domains, "test-coherence"), "Domain has the wrong ID")
		})
	}
}

//TestReturn404ForMissingDomain should return a 404 response for a request for a non-existent domain
func TestReturn404ForMissingDomain(t *testing.T) {
	Domains = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": getModelBindingPair(),
	}

	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testReturn404ForMissingDomain",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	// run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN an array of clusters and a model binding pair that reference a WLS domain
			createAndInsertWLSDomain("cluster1", "test-weblogic", tt.listers)
			Init(tt.listers)
			// setup http request processing
			vars := map[string]string{
				"id": "non-existent",
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/domains", nil)
			request = mux.SetURLVars(request, vars)
			// WHEN a request is submitted for a non-existent domain
			ReturnSingleDomain(response, request)
			// THEN a NOT FOUND HTTP status response message will be returned
			assert.Equal(t, http.StatusNotFound, response.Code, "wrong response code")
		})
	}
}

// TestReturnSingleDomain will test that a configured domain is returned
func TestReturnSingleDomain(t *testing.T) {
	Domains = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": getModelBindingPair(),
	}

	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testReturnSingleDomain",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	// run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN an array of clusters and a model binding pair referencing a WLS domain
			createAndInsertWLSDomain("cluster1", "test-weblogic", tt.listers)
			Init(tt.listers)
			// setup http request processing
			vars := map[string]string{
				"id": "test-weblogic",
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/domains", nil)
			request = mux.SetURLVars(request, vars)
			// WHEN a request is submitted for domain with ID "test-weblogic"
			ReturnSingleDomain(response, request)
			// THEN the response will has a HTTP OK status (200) with a single domain JSON response body
			assert.Equal(t, http.StatusOK, response.Code, "wrong response code")
			var domain Domain
			err := json.Unmarshal(response.Body.Bytes(), &domain)
			if err != nil {
				t.Errorf("Error while reading response JSON: %s", err)
			}
			// Validate the correct number of domains (1), and the domain has the expected namespace
			assert.NotNil(t, domain, "Domain should not be nil")
			assert.Equal(t, "test3", domain.Namespace, "Domain has the wrong namespace")
		})
	}
}

//createAndInsertWLSDomain will augment the clusters referenced by the listers with a WLSDomain
func createAndInsertWLSDomain(clusterName string, domainName string, listers controller.Listers) {
	wlsDomain := createDomain(domainName)
out:
	for _, mbp := range *listers.ModelBindingPairs {
		// grab all of the domains
		for _, mc := range mbp.ManagedClusters {
			if mc.Name == clusterName {
				if mc.WlsDomainCRs == nil {
					mc.WlsDomainCRs = make([]*v8weblogic.Domain, 1)
				}
				mc.WlsDomainCRs[0] = &wlsDomain
				break out
			}
		}
	}
}

//createDomain creates a test WLS domain
func createDomain(domainName string) v8weblogic.Domain {
	domain := v8weblogic.Domain{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WlsDomain",
			APIVersion: "verrazzano.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      domainName,
			UID:       "3",
			Namespace: "test3",
			Labels:    make(map[string]string),
		},
		Spec: v8weblogic.DomainSpec{
			AdminServer: v8weblogic.AdminServer{
				AdminService: v8weblogic.AdminService{
					Channels: func() []v8weblogic.Channel {
						var channels []v8weblogic.Channel

						channels = append(channels, v8weblogic.Channel{
							ChannelName: "default",
						})

						channels = append(channels, v8weblogic.Channel{
							ChannelName: "T3Channel",
						})
						return channels

					}(),
				},
			},
			AllowReplicasBelowMinDynClusterSize: false,
			Clusters: func() []v8weblogic.Cluster {
				var clusters []v8weblogic.Cluster

				clusters = append(clusters, v8weblogic.Cluster{
					ClusterName: "test1",
				})

				return clusters

			}(),
			ConfigOverrides:             "",
			ConfigOverrideSecrets:       nil,
			Configuration:               v8weblogic.Configuration{},
			DataHome:                    "",
			DomainHome:                  "",
			DomainHomeInImage:           true,
			DomainHomeSourceType:        "",
			DomainUID:                   "",
			HttpAccessLogInLogHome:      false,
			Image:                       "",
			ImagePullPolicy:             "",
			ImagePullSecrets:            nil,
			IncludeServerOutInPodLog:    false,
			IntrospectVersion:           "",
			LogHome:                     "",
			LogHomeEnabled:              false,
			ManagedServers:              nil,
			MaxClusterConcurrentStartup: 0,
			Replicas:                    nil,
			RestartVersion:              "",
			ServerPod:                   v8weblogic.ServerPod{},
			ServerService:               v8weblogic.ServerService{},
			ServerStartPolicy:           "",
			ServerStartState:            "",
			WebLogicCredentialsSecret:   corev1.SecretReference{},
		},
		Status: v8weblogic.DomainStatus{
			Servers: func() []v8weblogic.ServerStatus {
				var servers []v8weblogic.ServerStatus

				servers = append(servers, v8weblogic.ServerStatus{
					ClusterName: "test1",
					NodeName:    "node1",
					ServerName:  "testmanaged",
				})

				servers = append(servers, v8weblogic.ServerStatus{
					ClusterName: "test2",
					NodeName:    "node2",
					ServerName:  "testadmin",
				})
				return servers

			}(),
		},
	}
	return domain
}

//getModelBindingPair returns a test model binding pair
func getModelBindingPair() *types.ModelBindingPair {
	return testutil.ReadModelBindingPair("../../testutil/testdata/test_model.yaml",
		"../../testutil/testdata/test_binding.yaml",
		"../../testutil/testdata/test_managed_cluster_1.yaml",
		"../../testutil/testdata/test_managed_cluster_2.yaml")
}

//foundByName returns the domain if the array contains it
func foundByID(domain []Domain, key string) bool {
	for _, domain := range Domains {
		if domain.ID == key {
			return true
		}
	}
	return false
}
