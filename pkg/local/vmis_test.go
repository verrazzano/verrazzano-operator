// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	verrazzanov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned/typed/vmcontroller/v1"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

const EXISTING_BINDING = "existingBinding"
const EXISTING_BINDING_TO_BE_UPDATED = "existingBindingToBeUpdated"
const NONEXISTING_BINDING = "nonexistingBinding"
const ERROR_LIST_BINDING = "errorListBinding"
const ERROR_GET_BINDING = "errorGetBinding"
const ERROR_CREATE_BINDING = "errorCreateBinding"
const ERROR_DELETE_BINDING = "errorDeleteBinding"

func doTestCreateStorageOption(t *testing.T, option string, expected string) {
	storage := createStorageOption(option)
	prometheus := vmov1.Prometheus{
		Enabled: true,
		Storage: storage,
	}
	assert.Equal(t, expected, storage.Size, "right storage size")
	assert.Equal(t, expected, prometheus.Storage.Size, "right storage size")
}

func TestCreateStorageOption(t *testing.T) {
	doTestCreateStorageOption(t, "false", "")
	doTestCreateStorageOption(t, "False", "")
	doTestCreateStorageOption(t, "", "50Gi")
	doTestCreateStorageOption(t, "true", "50Gi")
	doTestCreateStorageOption(t, "True", "50Gi")
	doTestCreateStorageOption(t, "random", "50Gi")
}

func createTestBinding(bindingName string) *v1beta1v8o.VerrazzanoBinding {
	binding := v1beta1v8o.VerrazzanoBinding{}
	binding.Name = bindingName
	return &binding
}

func createTestInstanceWithDifferentUri(bindingName string) *vmov1.VerrazzanoMonitoringInstance {
	return createInstance(createTestBinding(bindingName), "anotherVerrazzanoURI", "")
}

func createTestInstance(bindingName string) *vmov1.VerrazzanoMonitoringInstance {
	return createInstance(createTestBinding(bindingName), "testVerrazzanoURI", "")
}

func TestCreateInstance(t *testing.T) {
	vmi := createTestInstance("testBinding")
	assert.Equal(t, "testBinding", vmi.Name, "right Namespace")
	assert.Equal(t, constants.VerrazzanoNamespace, vmi.Namespace, "right Namespace")
	assert.Equal(t, constants.VmiSecretName, vmi.Spec.SecretsName, "right SecretsName")
	assert.Equal(t, "vmi.testBinding.testVerrazzanoURI", vmi.Spec.URI, "right URI")
	assert.Equal(t, "verrazzano-ingress.testVerrazzanoURI", vmi.Spec.IngressTargetDNSName, "right IngressTargetDNSName")
}

// Fake VmoClientSet
type fakeVmoClientSet struct{}

func (f fakeVmoClientSet) Discovery() discovery.DiscoveryInterface {
	panic("Not implemented!")
}

func (f fakeVmoClientSet) VerrazzanoV1() verrazzanov1.VerrazzanoV1Interface {
	return fakeVmoVerrazzanoV1Client{}
}

// Fake VmoVerrazzanoV1Client
type fakeVmoVerrazzanoV1Client struct{}

func (f fakeVmoVerrazzanoV1Client) RESTClient() rest.Interface {
	panic("Not implemented!")
}

func (f fakeVmoVerrazzanoV1Client) VerrazzanoMonitoringInstances(namespace string) verrazzanov1.VerrazzanoMonitoringInstanceInterface {
	return fakeVmiInterface{}
}

// Fake VmiInterface
type fakeVmiInterface struct{}

func (f fakeVmiInterface) Create(ctx context.Context, verrazzanoMonitoringInstance *vmov1.VerrazzanoMonitoringInstance, opts metav1.CreateOptions) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if verrazzanoMonitoringInstance.Name == ERROR_CREATE_BINDING {
		return nil, errors.New("error create VMI")
	} else {
		return createTestInstance(verrazzanoMonitoringInstance.Name), nil
	}
}

func (f fakeVmiInterface) Update(ctx context.Context, verrazzanoMonitoringInstance *vmov1.VerrazzanoMonitoringInstance, opts metav1.UpdateOptions) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if verrazzanoMonitoringInstance.Name == EXISTING_BINDING_TO_BE_UPDATED {
		return createTestInstance(verrazzanoMonitoringInstance.Name), nil
	} else {
		return nil, errors.New("VMI can not be updated")
	}
}

func (f fakeVmiInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == EXISTING_BINDING {
		return nil
	} else if name == ERROR_DELETE_BINDING {
		return errors.New("error delete VMI")
	} else{
		return errors.New("VMI doesn't exist")
	}
}

func (f fakeVmiInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("Not implemented!")
}

func (f fakeVmiInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*vmov1.VerrazzanoMonitoringInstance, error) {
	panic("Not implemented!")
}

func (f fakeVmiInterface) List(ctx context.Context, opts metav1.ListOptions) (*vmov1.VerrazzanoMonitoringInstanceList, error) {
	panic("Not implemented!")
}

func (f fakeVmiInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("Not implemented!")
}

func (f fakeVmiInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *vmov1.VerrazzanoMonitoringInstance, err error) {
	panic("Not implemented!")
}

// Fake VmiLister
type fakeVmiLister struct{}

func (f fakeVmiLister) List(selector labels.Selector) (ret []*vmov1.VerrazzanoMonitoringInstance, err error) {
	panic("Not implemented!")
}

func (f fakeVmiLister) VerrazzanoMonitoringInstances(namespace string) vmolisters.VerrazzanoMonitoringInstanceNamespaceLister {
	return fakeVmiNamespaceLister{}
}

// Fake VmiNamespaceLister
type fakeVmiNamespaceLister struct{}

func (f fakeVmiNamespaceLister) List(selector labels.Selector) (ret []*vmov1.VerrazzanoMonitoringInstance, err error) {
	if selector.Matches(labels.Set(map[string]string{constants.VerrazzanoBinding: EXISTING_BINDING})) {
		return []*vmov1.VerrazzanoMonitoringInstance{createTestInstance(EXISTING_BINDING)}, nil
	} else if selector.Matches(labels.Set(map[string]string{constants.VerrazzanoBinding: ERROR_LIST_BINDING})) {
		return nil, errors.New("error list VMI")
	} else if selector.Matches(labels.Set(map[string]string{constants.VerrazzanoBinding: ERROR_DELETE_BINDING})) {
		return []*vmov1.VerrazzanoMonitoringInstance{createTestInstance(ERROR_DELETE_BINDING)}, nil
	} else {
		return nil, nil
	}
}

func (f fakeVmiNamespaceLister) Get(name string) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if name == EXISTING_BINDING_TO_BE_UPDATED {
		return createTestInstanceWithDifferentUri(EXISTING_BINDING_TO_BE_UPDATED), nil
	} else if name == ERROR_GET_BINDING {
		return nil, errors.New("error Get VMI")
	} else if name == NONEXISTING_BINDING {
		return nil, nil
	} else if name == ERROR_CREATE_BINDING {
		return nil, nil
	} else {
		return createTestInstance(name), nil
	}
}

func TestCreateUpdateVmi(t *testing.T) {
	type args struct {
		bindingName      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create existing",
			args: args{
				bindingName: EXISTING_BINDING,
			},
			wantErr: false,
		},
		{
			name: "update existing",
			args: args{
				bindingName: EXISTING_BINDING_TO_BE_UPDATED,
			},
			wantErr: false,
		},
		{
			name: "create non-existing",
			args: args{
				bindingName: NONEXISTING_BINDING,
			},
			wantErr: false,
		},
		{
			name: "error getting",
			args: args{
				bindingName: ERROR_GET_BINDING,
			},
			wantErr: true,
		},
		{
			name: "error creating",
			args: args{
				bindingName: ERROR_CREATE_BINDING,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateUpdateVmi(createTestBinding(tt.args.bindingName), fakeVmoClientSet{}, fakeVmiLister{}, "testVerrazzanoURI", ""); (err != nil) != tt.wantErr {
				t.Errorf("CreateUpdateVmi() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteVmi(t *testing.T) {
	type args struct {
		bindingName      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "delete existing",
			args: args{
				bindingName: EXISTING_BINDING,
			},
			wantErr: false,
		},
		{
			name: "delete non-existing",
			args: args{
				bindingName: NONEXISTING_BINDING,
			},
			wantErr: false,
		},
		{
			name: "error listing",
			args: args{
				bindingName: ERROR_LIST_BINDING,
			},
			wantErr: true,
		},
		{
			name: "error deleting",
			args: args{
				bindingName: ERROR_DELETE_BINDING,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteVmi(createTestBinding(tt.args.bindingName), fakeVmoClientSet{}, fakeVmiLister{}); (err != nil) != tt.wantErr {
				t.Errorf("DeleteVmi() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
