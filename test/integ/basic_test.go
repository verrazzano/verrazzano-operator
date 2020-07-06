// Copyright (c) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ

import (
	"context"
	"fmt"
	"testing"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/test/integ/framework"
	testutil "github.com/verrazzano/verrazzano-operator/test/integ/util"
)

// Register 2 Managed Clusters pointing (both pointing to the local cluster),
// create a binding, verify expected elements are created
func TestVerrazzanonBasic1(t *testing.T) {
	f := framework.Global

	clusterNames := []string{"cluster1", "cluster2"}
	for _, clusterName := range clusterNames {
		_, err := testutil.CreateManagedCluster(f, clusterName)
		defer testutil.DeleteManagedCluster(f, clusterName)
		if err != nil {
			t.Fatalf("Failed to create ManagedCluster: %v", err)
		}
	}

	componentNamesToClusters := map[string]string{"component1": clusterNames[0], "component2": clusterNames[0], "component3": clusterNames[1]}

	appModel, err := testutil.CreateAppModel(f, "my-model-"+f.RunID)
	if err != nil {
		t.Fatalf("Failed to create ApplicationModel: %v", err)
	}

	appBinding, err := testutil.CreateAppBinding(f, "my-app-binding-"+f.RunID, appModel.Name, componentNamesToClusters)
	if err != nil {
		t.Fatalf("Failed to create ApplicationBinding: %v", err)
	}
	defer func() {
		fmt.Println(fmt.Sprintf("Deleting VerrazzanoBinding %s", appBinding.Name))
		err = testutil.DeleteAppBinding(f, appBinding.Name)
		if err != nil {
			fmt.Println(fmt.Sprintf("Failed to delete VerrazzanoBinding: %v", err))
		}

		// Wait for the binding to be deleted before we delete the model
		fmt.Println(fmt.Sprintf("Waiting for binding VerrazzanoBinding %s to be deleted", appBinding.Name))
		err = testutil.Retry(testutil.DefaultRetry, func() (bool, error) {
			_, err = f.VerrazzanoOperatorClient.VerrazzanoV1beta1().VerrazzanoBindings(f.Namespace).Get(context.TODO(), appBinding.Name, v1.GetOptions{})
			if err != nil {
				if k8sErrors.IsNotFound(err) {
					return true, nil
				}
				fmt.Println(fmt.Sprintf("Failed to get VerrazzanoBinding: %v", err))
			}

			return false, nil
		})
		if err != nil {
			fmt.Println(fmt.Sprintf("Timed out waiting for binding to be deleted: %v", err))
		}

		fmt.Println(fmt.Sprintf("Deleting VerrazzanoModel %s", appModel.Name))
		err := testutil.DeleteAppModel(f, appModel.Name)
		if err != nil {
			fmt.Println(fmt.Sprintf("Failed to delete VerrazzanoModel: %v", err))
		}
	}()

	// Verify expected elements
	verifyBindingRelatedElements(t, appBinding)
}

// For now, this method assumes that all Managed Clusters are the same cluster (passed into the tests via --kubeconfig).
// If we ever want to support tests within *this* repo that can act multi-cluster, we'll need to adjust this logic
func verifyBindingRelatedElements(t *testing.T, binding *v1beta1v8o.VerrazzanoBinding) {
	fmt.Println("Verifying Managed Cluster elements related to ApplicationBinding " + binding.Name)
	f := framework.Global

	// Verify deployments
	deploymentNamespace := f.Namespace
	var expectedDeployments = []string{}

	for _, deploymentName := range expectedDeployments {
		fmt.Printf("Waiting for deployment %s", deploymentName)
		err := testutil.WaitForDeploymentAvailable(deploymentNamespace, deploymentName, 1, f.KubeClient)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Println("  ==> All deployments are available")

	// Verify service accounts
	var expectedServiceAccounts = []string{"verrazzano-operator"}
	for _, serviceAccount := range expectedServiceAccounts {
		fmt.Printf("Expected serviceAccount '%s'\n", serviceAccount)
		err := testutil.WaitForServiceAccountAvailable(deploymentNamespace, serviceAccount, f.KubeClient)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Println("  ==> All ServiceAccounts are available")
}
