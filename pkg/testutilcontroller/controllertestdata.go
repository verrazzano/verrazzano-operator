// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutilcontroller

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	listers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/listers/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"reflect"
)

func testIndexFunc(obj interface{}) ([]string, error) {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return []string{}, nil
	}
	switch t := obj.(type) {
	case *metav1.ObjectMeta:
		return []string{obj.(*metav1.ObjectMeta).Namespace}, nil
	case *v1beta1.VerrazzanoManagedCluster:
		return []string{obj.(*v1beta1.VerrazzanoManagedCluster).Namespace}, nil
	default:
		msg := fmt.Sprintf("Unknown Type %T", t)
		fmt.Printf(msg)
		return nil, errors.New(msg)
	}
}

func testKeyFunc(obj interface{}) (string, error) {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return "", nil
	}
	switch t := obj.(type) {
	case *v1beta1.VerrazzanoManagedCluster:
		return string(obj.(*v1beta1.VerrazzanoManagedCluster).UID), nil
	case *types.LocationInfo:
		return string(obj.(*types.LocationInfo).UID), nil
	default:
		msg := fmt.Sprintf("Unknown Type %T", t)
		fmt.Printf(msg)
		return "", errors.New(msg)
	}
}

// NewControllerListers creates a fake set of listers to be used for unit tests
func NewControllerListers(clients *kubernetes.Interface, clusters []v1beta1.VerrazzanoManagedCluster, modelBindingPairs *map[string]*types.ModelBindingPair) controller.Listers {
	testIndexers := map[string]cache.IndexFunc{
		"namespace": testIndexFunc,
	}
	clusterIndexer := cache.NewIndexer(testKeyFunc, testIndexers)
	for i := range clusters {
		clusterIndexer.Add(&clusters[i])
	}

	clusterLister := listers.NewVerrazzanoManagedClusterLister(clusterIndexer)
	return controller.Listers{
		ManagedClusterLister: &clusterLister,
		KubeClientSet:        clients,
	}
}
