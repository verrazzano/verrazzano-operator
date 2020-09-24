package managed

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestCreateServiceAccounts(t *testing.T) {
	mbPair := testutil.GetModelBindingPair()
	managedConnections := testutil.GetManagedClusterConnections()

	err := CreateServiceAccounts(mbPair, managedConnections)
	if err != nil {
		t.Fatal(fmt.Sprintf("can't create service account: %v", err))
	}

	for clusterName := range mbPair.ManagedClusters {
		managedClusterConnection := managedConnections[clusterName]
		serviceAccounts, err := managedClusterConnection.ServiceAccountLister.List(labels.Everything())
		if err != nil {
			t.Fatal(fmt.Sprintf("can't list service account for cluster %s: %v", err, clusterName))
		}

		for _, sa := range serviceAccounts {
			assertCreateServiceAccounts(t, sa, clusterName)
		}
	}

	/*
		managedClusterConnection := managedConnections["cluster1"]
		existingService, err := managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatal(fmt.Sprintf("can't list service for cluster %s: %v", err, "cluster1"))
		}
		assertCreateService(t, existingService, clusterName)

		// Update the port value so when we call CreateServices again the update code is executed.
		existingService.Items[0].Spec.Ports[0].Port = 9200
		_, err = managedClusterConnection.KubeClient.CoreV1().Services(constants.MonitoringNamespace).Update(context.TODO(), &existingService.Items[0], metav1.UpdateOptions{})
		if err != nil {
			t.Fatal(fmt.Sprintf("can't update service for cluster %s: %v", err, clusterName))
		}

	*/
}

func assertCreateServiceAccounts(t *testing.T, sa *corev1.ServiceAccount, clusterName string) {
	labels := util.GetManagedLabelsNoBinding(clusterName)

	assert := assert.New(t)

	assert.Equal(constants.VerrazzanoNamespace, sa.Name)
	assert.Equal(labels, sa.Labels)
	assert.Equal("test-imagePullSecret", sa.ImagePullSecrets[0].Name)
}

func Test_createServiceAccount(t *testing.T) {
	type args struct {
		binding                  *v1beta1v8o.VerrazzanoBinding
		managedClusterConnection *util.ManagedClusterConnection
		newServiceAccounts       []*corev1.ServiceAccount
		clusterName              string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := createServiceAccount(tt.args.binding, tt.args.managedClusterConnection, tt.args.newServiceAccounts, tt.args.clusterName); (err != nil) != tt.wantErr {
				t.Errorf("createServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newServiceAccounts(t *testing.T) {
	type args struct {
		binding          *v1beta1v8o.VerrazzanoBinding
		managedCluster   *types.ManagedCluster
		name             string
		labels           map[string]string
		namespaceName    string
		imagePullSecerts []corev1.LocalObjectReference
	}
	tests := []struct {
		name string
		args args
		want []*corev1.ServiceAccount
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newServiceAccounts(tt.args.binding, tt.args.managedCluster, tt.args.name, tt.args.labels, tt.args.namespaceName, tt.args.imagePullSecerts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newServiceAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}
