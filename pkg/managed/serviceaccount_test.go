package managed

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestCreateServiceAccounts(t *testing.T) {
	// Setup the needed structures to test creating service accounts
	mbPair := getAppMbPair()
	managedConnections := getManagedConnections()
	createAppNamespaces(t, managedConnections)

	// Create the expected service accounts
	err := CreateServiceAccounts(&mbPair, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	// Validate the service accounts created for the "local" cluster
	managedClusterConnection := managedConnections["local"]
	serviceAccounts, err := managedClusterConnection.ServiceAccountLister.List(labels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster local: %v", err))
	}
	assert.Equal(t, 2, len(serviceAccounts), "2 service accounts were expected for cluster local")
	for _, sa := range serviceAccounts {
		assertCreateServiceAccounts(t, sa, "local")
	}

	// Validate the service accounts created for the "cluster-1" cluster
	managedClusterConnection = managedConnections["cluster-1"]
	serviceAccounts, err = managedClusterConnection.ServiceAccountLister.List(labels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster cluster-1: %v", err))
	}
	assert.Equal(t, 1, len(serviceAccounts), "1 service account was expected for cluster cluster-1")
	for _, sa := range serviceAccounts {
		assertCreateServiceAccounts(t, sa, "cluster-1")
	}

	// Update the imagePullSecret for service account we created in "cluster-1" cluster.
	serviceAccounts[0].ImagePullSecrets[0].Name = ""
	_, err = managedClusterConnection.KubeClient.CoreV1().ServiceAccounts("test3").Update(context.TODO(), serviceAccounts[0], metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't update service account for cluster cluster-1: %v", err))
	}

	// Call CreateServiceAccounts again the update code will be executed to make the service account right.
	err = CreateServiceAccounts(&mbPair, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	serviceAccounts, err = managedClusterConnection.ServiceAccountLister.List(labels.Everything())
	if err != nil {
		t.Fatal(fmt.Sprintf("can't list service account for cluster cluster-1: %v", err))
	}
	assert.Equal(t, 1, len(serviceAccounts), "1 service account was expected for cluster cluster-1")
	for _, sa := range serviceAccounts {
		assertCreateServiceAccounts(t, sa, "cluster-1")
	}

}

func getManagedConnections() map[string]*util.ManagedClusterConnection {
	return map[string]*util.ManagedClusterConnection{
		"local":     testutil.GetManagedClusterConnection("local"),
		"cluster-1": testutil.GetManagedClusterConnection("cluster-1"),
	}
}

func getAppMbPair() types.ModelBindingPair {
	return types.ModelBindingPair{
		Model: &v1beta1v8o.VerrazzanoModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testModel",
				Namespace: "default",
			},
			Spec: v1beta1v8o.VerrazzanoModelSpec{
				Description: "A test model",
			},
		},
		Binding: &v1beta1v8o.VerrazzanoBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testBinding",
				Namespace: "default",
			},
			Spec: v1beta1v8o.VerrazzanoBindingSpec{
				Description: "A test binding",
				ModelName:   "testModel",
				Placement: []v1beta1v8o.VerrazzanoPlacement{
					{
						Name: "local",
						Namespaces: []v1beta1v8o.KubernetesNamespace{
							{
								Name: "test1",
								Components: []v1beta1v8o.BindingComponent{
									{
										Name: "test-1",
									},
								},
							},
							{
								Name: "test2",
								Components: []v1beta1v8o.BindingComponent{
									{
										Name: "test-2",
									},
								},
							},
						},
					},
					{
						Name: "cluster-1",
						Namespaces: []v1beta1v8o.KubernetesNamespace{
							{
								Name: "test3",
								Components: []v1beta1v8o.BindingComponent{
									{
										Name: "test-3",
									},
								},
							},
						},
					},
				},
			},
		},
		ManagedClusters: map[string]*types.ManagedCluster{
			"local": {
				Name:       "local",
				Namespaces: []string{"test1", "test2"},
			},
			"cluster-1": {
				Name:       "cluster-1",
				Namespaces: []string{"test3"},
			},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test-imagePullSecret",
			},
		},
	}
}

func createAppNamespaces(t *testing.T, managedConnections map[string]*util.ManagedClusterConnection) {
	var ns = corev1.Namespace{}

	clusterConnection := managedConnections["local"]
	ns.Name = "test1"
	_, err := clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create namespace %s: %v", ns.Name, err))
	}
	ns.Name = "test2"
	clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create namespace %s: %v", ns.Name, err))
	}

	clusterConnection = managedConnections["cluster-1"]
	ns.Name = "test3"
	clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create namespace %s: %v", ns.Name, err))
	}
}

func assertCreateServiceAccounts(t *testing.T, sa *corev1.ServiceAccount, clusterName string) {
	labels := util.GetManagedLabelsNoBinding(clusterName)

	assert := assert.New(t)

	assert.Equal(constants.VerrazzanoNamespace, sa.Name, "namespace %s service account name name was not as expected", sa.Namespace)
	assert.Equal(labels, sa.Labels, "namespace %s service account namespace label was not as expected", sa.Namespace)
	assert.Equal("test-imagePullSecret", sa.ImagePullSecrets[0].Name, "namespace %s service account image pull secret was not as expected", sa.Namespace)
}
