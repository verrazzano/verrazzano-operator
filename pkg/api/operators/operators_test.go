package operators

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	v1wlsopr "github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"testing"
)

const wlsNamespace, wlsOperatorName, wlsBindingName = "test", "wls-operator", "mytestbinding"

// TestInit ensures that a set of operators can be obtained from the provided set of listers.
func TestInit(t *testing.T) {
	// Get a set of clusters
	Operators = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	type args struct {
		listers controller.Listers
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testInit",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	for _, tt := range tests {
		// add a WLSOperator to one of the clusters
		t.Run(tt.name, func(t *testing.T) {
			// append the WLS operator
			createAndInsertWLSOperator(1, tt)
			// Validate that there is the correct number of listers, the correct number of operators (1), and the
			// operator has the expected name
			assert.Equal(t, tt.listers, listerSet, "Wrong number of listers")
			assert.Equal(t, 1, len(Operators), "Wrong number of WLS operators")
			assert.Equal(t, "wls-operator-mytestbinding-1", Operators[0].Name, "Operator has the wrong name")
		})
	}
}

//TestReturnNoOperators ensures that an empty response is returned if no operators are available
func TestReturnNoOperators(t *testing.T) {
	// ensure initialization will yield no operators
	Operators = nil
	emptyModelBindingPairs := make(map[string]*types.ModelBindingPair)
	Init(controller.Listers{
		ModelBindingPairs: &emptyModelBindingPairs,
	})
	// setup http request processing
	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/operators", func(w http.ResponseWriter, r *http.Request) {
		ReturnAllOperators(w, r)
	})
	request, _ := http.NewRequest("GET", "/operators", nil)
	// inovke HTTP request
	router.ServeHTTP(recorder, request)
	// assert response code 200
	assert.Equal(t, http.StatusOK, recorder.Code, "wrong response code")
	var ops []Operator
	err := json.Unmarshal(recorder.Body.Bytes(), &ops)
	if err != nil {
		t.Errorf("Error while reading response JSON: %s", err)
	}
	// assert empty response
	assert.Empty(t, ops, "response should have no operators")
}

// TestReturnSingleOperator will test that a configured operator is returned
func TestReturnSingleOperator(t *testing.T) {
	// Get a set of clusters
	Operators = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	type args struct {
		listers controller.Listers
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testReturnSingleOperator",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// append WLS operator
			createAndInsertWLSOperator(1, tt)
			// setup http request processing
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/operators/{id}", func(w http.ResponseWriter, r *http.Request) {
				ReturnSingleOperator(w, r)
			})
			request, _ := http.NewRequest("GET", "/operators/1", nil)
			// inovke HTTP request
			router.ServeHTTP(recorder, request)
			// assert response code 200
			assert.Equal(t, http.StatusOK, recorder.Code, "wrong response code")
			var op Operator
			err := json.Unmarshal(recorder.Body.Bytes(), &op)
			if err != nil {
				t.Errorf("Error while reading response JSON: %s", err)
			}
			// Validate the correct number of operators (1), and the operator has the expected name
			assert.NotNil(t, op, "Operator should not be nil")
			assert.Equal(t, "wls-operator-mytestbinding-1", op.Name, "Operator has the wrong name")
		})
	}
}

//TestReturnAllOperators ensures that an array of operators is returned
func TestReturnAllOperators(t *testing.T) {
	// Get a set of clusters
	Operators = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	type args struct {
		listers controller.Listers
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testReturnAllOperators",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	for _, tt := range tests {
		// add a WLSOperator to one of the clusters
		t.Run(tt.name, func(t *testing.T) {
			// append the WLS Operator
			createAndInsertWLSOperator(1, tt)
			createAndInsertWLSOperator(2, tt)
			// setup http request processing
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/operators", func(w http.ResponseWriter, r *http.Request) {
				ReturnAllOperators(w, r)
			})
			request, _ := http.NewRequest("GET", "/operators", nil)
			// inovke HTTP request
			router.ServeHTTP(recorder, request)
			// assert response code 200
			assert.Equal(t, http.StatusOK, recorder.Code, "wrong response code")
			var ops []Operator
			err := json.Unmarshal(recorder.Body.Bytes(), &ops)
			if err != nil {
				t.Errorf("Error while reading response JSON: %s", err)
			}
			// Validate that there is the correct number of operators (1), and the
			// operator has the expected name
			assert.Equal(t, 2, len(ops), "There should be one operator in the list")
			assert.Equal(t, "wls-operator-mytestbinding-1", ops[0].Name, "Operator has the wrong name")
			assert.Equal(t, "wls-operator-mytestbinding-2", ops[1].Name, "Operator has the wrong name")
		})
	}
}

//TestReturn404ForMissingOperator should return a 404 response for a request for a non-existent operator
func TestReturn404ForMissingOperator(t *testing.T) {
	// Get a set of clusters
	Operators = nil
	clusters := testutil.GetTestClusters()
	var clients kubernetes.Interface = fake.NewSimpleClientset()
	modelBindingPairs := map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.GetModelBindingPair(),
	}
	type args struct {
		listers controller.Listers
	}
	tests := []struct {
		name    string
		listers controller.Listers
	}{
		{name: "testReturn404ForMissingOperator",
			listers: testutilcontroller.NewControllerListers(&clients, clusters, &modelBindingPairs)},
	}
	for _, tt := range tests {
		// add a WLSOperator to one of the clusters
		t.Run(tt.name, func(t *testing.T) {
			// append the WLS operator
			createAndInsertWLSOperator(1, tt)
			// setup http request processing
			recorder := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/operators/{id}", func(w http.ResponseWriter, r *http.Request) {
				ReturnSingleOperator(w, r)
			})
			request, _ := http.NewRequest("GET", "/operators/2", nil)
			// inovke HTTP request
			router.ServeHTTP(recorder, request)
			// assert response code 404
			assert.Equal(t, http.StatusNotFound, recorder.Code, "wrong response code")
		})
	}
}

//createAndInsertWLSOperator will augment the clusters referenced by the listers with a WLSOperator in cluster2
func createAndInsertWLSOperator(number int, tt struct {
	name    string
	listers controller.Listers
}) {
	wlsOperator := createOperator(number)
	clusterName := fmt.Sprintf("%s%d", "cluster", number)
out:
	for _, mbp := range *tt.listers.ModelBindingPairs {
		// grab all of the domains
		for _, mc := range mbp.ManagedClusters {
			if mc.Name == clusterName {
				mc.WlsOperator = &wlsOperator
				break out
			}
		}
	}
	Init(tt.listers)
}

func createOperator(number int) v1wlsopr.WlsOperator {
	wlsOperator := v1wlsopr.WlsOperator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WlsOperator",
			APIVersion: "verrazzano.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", wlsOperatorName, number),
			UID:       k8stypes.UID(fmt.Sprintf("%d", number)),
			Namespace: fmt.Sprintf("%s%d", wlsNamespace, number),
			Labels:    make(map[string]string),
		},
		Spec: v1wlsopr.WlsOperatorSpec{
			Description:      fmt.Sprintf("WebLogic Operator for managed cluster %s using binding %s", "local", wlsBindingName),
			Name:             fmt.Sprintf("%s-%s-%d", wlsOperatorName, wlsBindingName, number),
			Namespace:        fmt.Sprintf("%s%d", wlsNamespace, number),
			ServiceAccount:   fmt.Sprintf("%s-%d", wlsOperatorName, number),
			Image:            "myregistry:/wlsoper:1",
			ImagePullPolicy:  "IfNotPresent",
			DomainNamespaces: []string{"default"},
		},
		Status: v1wlsopr.WlsOperatorStatus{},
	}
	return wlsOperator
}
