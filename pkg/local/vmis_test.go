// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"

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

const (
	ExistingBinding            = "existingBinding"
	ExistingBindingToBeUpdated = "existingBindingToBeUpdated"
	NonExistingBinding         = "nonExistingBinding"

	ErrorListBinding   = "errorListBinding"
	ErrorGetBinding    = "errorGetBinding"
	ErrorCreateBinding = "errorCreateBinding"
	ErrorDeleteBinding = "errorDeleteBinding"

	ErrMsgCreate = "error create VMI"
	ErrMsgUpdate = "VMI can not be updated"
	ErrMsgDelete = "error delete VMI"
	ErrMsgList   = "error list VMI"
	ErrMsgGet    = "error Get VMI"
)

func TestCreateStorageOption(t *testing.T) {
	type args struct {
		storageValue            string
		enableMonitoringStorage string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"50Gi Enabled",
			args{"50Gi", "true"},
			"50Gi",
		},
		{
			"100Gi Enabled",
			args{"100Gi", "true"},
			"100Gi",
		},
		{
			"None Enabled",
			args{"", "true"},
			"",
		},
		{
			"100Gi Disabled",
			args{"100Gi", "false"},
			"",
		},
		{
			"None Disabled",
			args{"", "false"},
			"",
		},
		{
			"100Gi InvalidFlag",
			args{"100Gi", "boo"},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createStorageOption(tt.args.storageValue, tt.args.enableMonitoringStorage).Size; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createStorageOption() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateInstance(t *testing.T) {
	type args struct {
		bindingName   string
		verrazzanoURI string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Valid",
			args{
				"testBinding",
				"validURI",
			},
			false,
		},
		{
			"Valid2",
			args{
				"testBinding2",
				"validURI2",
			},
			false,
		},
		{
			"Invalid",
			args{
				"testBinding",
				"",
			},
			true,
		},
		{
			"DevProfile",
			args{
				"devProfile",
				"systemURI",
			},
			false,
		},
	}
	defer func() { os.Unsetenv("USE_SYSTEM_VMI") }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.bindingName == "devProfile" {
				os.Setenv("USE_SYSTEM_VMI", "true")
			} else {
				os.Unsetenv("USE_SYSTEM_VMI")
			}
			got, err := createInstance(createTestBinding(tt.args.bindingName), tt.args.verrazzanoURI, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("createInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got.Name != tt.args.bindingName {
					t.Errorf("createInstance() got Name = %v, want %v", got.Name, tt.args.bindingName)
				}
				if got.Namespace != constants.VerrazzanoNamespace {
					t.Errorf("createInstance() got Namespace = %v, want %v", got.Namespace, constants.VerrazzanoNamespace)
				}
				if got.Spec.SecretsName != constants.VmiSecretName {
					t.Errorf("createInstance() got Spec.SecretsName = %v, want %v", got.Spec.SecretName, constants.VmiSecretName)
				}
				if tt.args.bindingName == "devProfile" {
					if got.Spec.URI != "vmi."+"system."+tt.args.verrazzanoURI {
						t.Errorf("createInstance() got Spec.URI = %v, want %v", got.Spec.URI, "vmi."+"system."+tt.args.verrazzanoURI)
					}
				} else {
					if got.Spec.URI != "vmi."+tt.args.bindingName+"."+tt.args.verrazzanoURI {
						t.Errorf("createInstance() got Spec.URI = %v, want %v", got.Spec.URI, "vmi."+tt.args.bindingName+"."+tt.args.verrazzanoURI)
					}
				}
				if got.Spec.IngressTargetDNSName != "verrazzano-ingress."+tt.args.verrazzanoURI {
					t.Errorf("createInstance() got Spec.IngressTargetDNSName = %v, want %v", got.Spec.IngressTargetDNSName, "verrazzano-ingress."+tt.args.verrazzanoURI)
				}
			}
		})
	}
}

func TestCreateInstanceError(t *testing.T) {
	type args struct {
		bindingName   string
		verrazzanoURI string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"TestError",
			args{
				"testCreateInstanceError",
				"",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			originalCreateInstance := createInstanceFunc
			defer func() { createInstanceFunc = originalCreateInstance }()
			createInstanceFunc = func(binding *types.ClusterBinding, verrazzanoURI string, enableMonitoringStorage string) (*vmov1.VerrazzanoMonitoringInstance, error) {
				return nil, errors.New("Test Error")
			}
			defer func() { createInstanceFunc = originalCreateInstance }()
			err := CreateUpdateVmi(createTestBinding(tt.args.bindingName), fakeVmoClientSet{}, fakeVmiLister{}, "testVerrazzanoURI", "")
			assert.EqualError(err, "Test Error", "createInstanceFailed to return an error")
		})
	}
}

func createTestBinding(bindingName string) *types.ClusterBinding {
	binding := types.ClusterBinding{}
	binding.Name = bindingName
	return &binding
}

func createTestInstanceWithDifferentURI(bindingName string) *vmov1.VerrazzanoMonitoringInstance {
	val, _ := createInstance(createTestBinding(bindingName), "anotherVerrazzanoURI", "")
	return val
}

func createTestInstance(bindingName string) *vmov1.VerrazzanoMonitoringInstance {
	val, _ := createInstance(createTestBinding(bindingName), "testVerrazzanoURI", "")
	return val
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
	if verrazzanoMonitoringInstance.Name == ErrorCreateBinding {
		return nil, errors.New(ErrMsgCreate)
	}
	return createTestInstance(verrazzanoMonitoringInstance.Name), nil
}

func (f fakeVmiInterface) Update(ctx context.Context, verrazzanoMonitoringInstance *vmov1.VerrazzanoMonitoringInstance, opts metav1.UpdateOptions) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if verrazzanoMonitoringInstance.Name == ExistingBindingToBeUpdated {
		return createTestInstance(verrazzanoMonitoringInstance.Name), nil
	}
	return nil, errors.New(ErrMsgUpdate)
}

func (f fakeVmiInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == ExistingBinding {
		return nil
	}
	return errors.New(ErrMsgDelete)
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
	if selector.Matches(labels.Set(map[string]string{constants.VerrazzanoBinding: ExistingBinding})) {
		return []*vmov1.VerrazzanoMonitoringInstance{createTestInstance(ExistingBinding)}, nil
	} else if selector.Matches(labels.Set(map[string]string{constants.VerrazzanoBinding: ErrorListBinding})) {
		return nil, errors.New(ErrMsgList)
	} else if selector.Matches(labels.Set(map[string]string{constants.VerrazzanoBinding: ErrorDeleteBinding})) {
		return []*vmov1.VerrazzanoMonitoringInstance{createTestInstance(ErrorDeleteBinding)}, nil
	} else {
		return nil, nil
	}
}

func (f fakeVmiNamespaceLister) Get(name string) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if name == ExistingBindingToBeUpdated {
		return createTestInstanceWithDifferentURI(ExistingBindingToBeUpdated), nil
	} else if name == ErrorGetBinding {
		return nil, errors.New(ErrMsgGet)
	} else if name == NonExistingBinding {
		return nil, nil
	} else if name == ErrorCreateBinding {
		return nil, nil
	} else {
		return createTestInstance(name), nil
	}
}

func TestCreateUpdateVmi(t *testing.T) {
	type args struct {
		bindingName string
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{
			name: "create existing",
			args: args{
				bindingName: ExistingBinding,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "update existing",
			args: args{
				bindingName: ExistingBindingToBeUpdated,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "create non-existing",
			args: args{
				bindingName: NonExistingBinding,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "error getting",
			args: args{
				bindingName: ErrorGetBinding,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "error creating",
			args: args{
				bindingName: ErrorCreateBinding,
			},
			wantErr:     true,
			expectedErr: ErrMsgCreate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateUpdateVmi(createTestBinding(tt.args.bindingName), fakeVmoClientSet{}, fakeVmiLister{}, "testVerrazzanoURI", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUpdateVmi() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.expectedErr != err.Error() {
					t.Errorf("CreateUpdateVmi() error = %v, expect err %v", err, tt.expectedErr)
				}
			}
		})
	}
}

func TestSharedVMIAppBinding(t *testing.T) {
	os.Setenv("USE_SYSTEM_VMI", "true")
	defer os.Unsetenv("USE_SYSTEM_VMI")

	err := CreateUpdateVmi(createTestBinding("foo"), fakeVmoClientSet{}, fakeVmiLister{}, "testVerrazzanoURI", "")
	assert.Nil(t, err, "Unexpected error for shared VMI with app binding")
}

func TestDeleteVmi(t *testing.T) {
	type args struct {
		bindingName string
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{
			name: "delete existing",
			args: args{
				bindingName: ExistingBinding,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "delete non-existing",
			args: args{
				bindingName: NonExistingBinding,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "error listing",
			args: args{
				bindingName: ErrorListBinding,
			},
			wantErr:     true,
			expectedErr: ErrMsgList,
		},
		{
			name: "error deleting",
			args: args{
				bindingName: ErrorDeleteBinding,
			},
			wantErr:     true,
			expectedErr: ErrMsgDelete,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteVmi(createTestBinding(tt.args.bindingName), fakeVmoClientSet{}, fakeVmiLister{})
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteVmi() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.expectedErr != err.Error() {
					t.Errorf("DeleteVmi() error = %v, expect err %v", err, tt.expectedErr)
				}
			}
		})
	}
}
