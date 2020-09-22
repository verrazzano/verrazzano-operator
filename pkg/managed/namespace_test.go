// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCreateNamespaces(t *testing.T) {
	type args struct {
		mbPair              *types.ModelBindingPair
		filteredConnections map[string]*util.ManagedClusterConnection
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create namespaces",
			args: args{
				mbPair:              getModelBindingPair(),
				filteredConnections: GetManagedClusterConnections(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateNamespaces(tt.args.mbPair, tt.args.filteredConnections); (err != nil) != tt.wantErr {
				t.Errorf("CreateNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}
			clusterConnection := tt.args.filteredConnections["cluster1"]

			namespaces, err := clusterConnection.KubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				t.Errorf("can't get namespaces: %v", err)
			}
			assert.Equal(t, 6, len(namespaces.Items))

			assertNamespace(t, clusterConnection, "test", true)
			assertNamespace(t, clusterConnection, "test2", true)
			assertNamespace(t, clusterConnection, "test3", true)
			assertNamespace(t, clusterConnection, "default", true)
			assertNamespace(t, clusterConnection, "istio-system", true)
		})
	}
}

func assertNamespace(t *testing.T, clusterConnection *util.ManagedClusterConnection, name string, wantExist bool) {
	namespace, err := clusterConnection.KubeClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if wantExist {
		if err != nil {
			t.Errorf("can't get namespace: %v", err)
		}
		assert.Equal(t, name, namespace.Name)
	} else {
		if err == nil || namespace != nil {
			t.Errorf("namespace %s exists", name)
		}
	}
}

func TestCleanupOrphanedNamespaces(t *testing.T) {
	type args struct {
		mbPair                             *types.ModelBindingPair
		availableManagedClusterConnections map[string]*util.ManagedClusterConnection
		allMbPairs                         map[string]*types.ModelBindingPair
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "cleanup namespaces",
			args: args{
				mbPair:                             getModelBindingPair(),
				availableManagedClusterConnections: GetManagedClusterConnections(),
				allMbPairs: map[string]*types.ModelBindingPair{
					"testBinding": getModelBindingPair(),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			clusterConnection := tt.args.availableManagedClusterConnections["cluster3"]

			ns := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"k8s-app":            "verrazzano.io",
						"verrazzano.cluster": "cluster3",
					},
				},
			}
			_, err := clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("can't create namespace: %v", err)
			}

			assertNamespace(t, clusterConnection, "foo", true)
			assertNamespace(t, clusterConnection, "test", true)

			if err = CleanupOrphanedNamespaces(tt.args.mbPair, tt.args.availableManagedClusterConnections, tt.args.allMbPairs); (err != nil) != tt.wantErr {
				t.Errorf("CleanupOrphanedNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertNamespace(t, clusterConnection, "foo", false)
			assertNamespace(t, clusterConnection, "test", false)
		})
	}
}

func TestDeleteNamespaces(t *testing.T) {
	type args struct {
		mbPair                             *types.ModelBindingPair
		availableManagedClusterConnections map[string]*util.ManagedClusterConnection
		bindingLabel                       bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "delete namespaces binding label true",
			args: args{
				mbPair:                             getModelBindingPair(),
				availableManagedClusterConnections: GetManagedClusterConnections(),
				bindingLabel:                       true,
			},
			wantErr: false,
		},
		{
			name: "delete namespaces binding label false",
			args: args{
				mbPair:                             getModelBindingPair(),
				availableManagedClusterConnections: GetManagedClusterConnections(),
				bindingLabel:                       false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterConnection := tt.args.availableManagedClusterConnections["cluster1"]

			ns := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"k8s-app":            "verrazzano.io",
						"verrazzano.cluster": "cluster1",
					},
				},
			}
			_, err := clusterConnection.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &ns, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("can't create namespace: %v", err)
			}

			assertNamespace(t, clusterConnection, "foo", true)
			assertNamespace(t, clusterConnection, "test", true)
			assertNamespace(t, clusterConnection, "test2", true)
			assertNamespace(t, clusterConnection, "test3", true)

			if err := DeleteNamespaces(tt.args.mbPair, tt.args.availableManagedClusterConnections, tt.args.bindingLabel); (err != nil) != tt.wantErr {
				t.Errorf("DeleteNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}

			assertNamespace(t, clusterConnection, "foo", tt.args.bindingLabel)
			assertNamespace(t, clusterConnection, "test", !tt.args.bindingLabel)
			assertNamespace(t, clusterConnection, "test2", !tt.args.bindingLabel)
			assertNamespace(t, clusterConnection, "test3", !tt.args.bindingLabel)
		})
	}
}
