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
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func testIndexFunc(obj interface{}) ([]string, error) {
	switch t := obj.(type) {
	case *metav1.ObjectMeta:
		return []string{obj.(*metav1.ObjectMeta).Namespace}, nil
	case *v1beta1.VerrazzanoManagedCluster:
		return []string{obj.(*v1beta1.VerrazzanoManagedCluster).Namespace}, nil
	case *v1beta1.VerrazzanoModel:
		return []string{obj.(*v1beta1.VerrazzanoModel).Namespace}, nil
	case *v1beta1.VerrazzanoBinding:
		return []string{obj.(*v1beta1.VerrazzanoBinding).Namespace}, nil
	default:
		fmt.Printf("Unknown Type %T", t)
		return nil, errors.New("unknown Type")
	}
}
func testKeyFuncCluster(obj interface{}) (string, error) {
	return string(obj.(*v1beta1.VerrazzanoManagedCluster).UID), nil
}
func testKeyFuncModel(obj interface{}) (string, error) {
	return string(obj.(*v1beta1.VerrazzanoModel).UID), nil
}
func testKeyFuncBinding(obj interface{}) (string, error) {
	return string(obj.(*v1beta1.VerrazzanoBinding).UID), nil
}

func NewControllerListers(clusters []v1beta1.VerrazzanoManagedCluster, modelBindingPairs *map[string]*types.ModelBindingPair) controller.Listers {
	testIndexers := map[string]cache.IndexFunc{
		"namespace": testIndexFunc,
	}
	clusterIndexer := cache.NewIndexer(testKeyFuncCluster, testIndexers)
	for i := range clusters {
		clusterIndexer.Add(&clusters[i])
	}

	modelIndexer := cache.NewIndexer(testKeyFuncModel, testIndexers)
	for _, mb := range *modelBindingPairs {
		modelIndexer.Add(mb.Model)
	}

	bindingIndexer := cache.NewIndexer(testKeyFuncBinding, testIndexers)
	for _, mb := range *modelBindingPairs {
		bindingIndexer.Add(mb.Binding)
	}

	clusterLister := listers.NewVerrazzanoManagedClusterLister(clusterIndexer)
	modelLister := listers.NewVerrazzanoModelLister(modelIndexer)
	bindingLister := listers.NewVerrazzanoBindingLister(bindingIndexer)
	var clientSet kubernetes.Interface = fake.NewSimpleClientset()
	return controller.Listers{
		ManagedClusterLister: &clusterLister,
		ModelLister:          &modelLister,
		BindingLister:        &bindingLister,
		ModelBindingPairs:    modelBindingPairs,
		KubeClientSet:        &clientSet,
	}
}
