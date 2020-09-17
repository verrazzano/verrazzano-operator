// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsopr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vz "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	wlsv1beta "github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	emptyBinding = &vz.VerrazzanoBinding{
		TypeMeta:   v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{},
		Spec:       vz.VerrazzanoBindingSpec{},
		Status:     vz.VerrazzanoBindingStatus{},
	}
)

func TestNewWlsOperatorCRMissingManagedClusterName(t *testing.T) {
	crConfig := WlsOperatorCRConfig{
		BindingName:        "mytestbinding",
		BindingNamespace:   "verazzano-testbinding",
		DomainNamespace:    "default",
		Labels:             make(map[string]string),
		ManagedClusterName: "",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}
	_, err := NewWlsOperatorCR(crConfig)
	if assert.Error(t, err) {
		assert.Equal(t, errors.New("ManagedClusterName is required"), err)
	}
}

func TestNewWlsOperatorCRMissingDomainNamespace(t *testing.T) {
	crConfig := WlsOperatorCRConfig{
		BindingName:        "mytestbinding",
		BindingNamespace:   "verazzano-testbinding",
		DomainNamespace:    "",
		Labels:             make(map[string]string),
		ManagedClusterName: "local",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}
	_, err := NewWlsOperatorCR(crConfig)
	if assert.Error(t, err) {
		assert.Equal(t, errors.New("DomainNamespace is required"), err)
	}
}

func TestNewWlsOperatorCR(t *testing.T) {
	assert := assert.New(t)
	crConfig := WlsOperatorCRConfig{
		BindingName:      "mytestbinding",
		BindingNamespace: "verazzano-testbinding",
		DomainNamespace:  "default",
		Labels: map[string]string{
			"mylabel": "mylabelvalue",
		},
		ManagedClusterName: "local",
		WlsOperatorImage:   "myregistry:/wlsoper:1",
	}

	wlsOperatorSpec := wlsv1beta.WlsOperatorSpec{
		Description:      fmt.Sprintf("WebLogic Operator for managed cluster %s using binding %s", crConfig.ManagedClusterName, crConfig.BindingName),
		Name:             fmt.Sprintf("%s-%s", operatorName, crConfig.BindingName),
		Namespace:        crConfig.BindingNamespace,
		ServiceAccount:   operatorName,
		Image:            crConfig.WlsOperatorImage,
		ImagePullPolicy:  "IfNotPresent",
		DomainNamespaces: []string{crConfig.DomainNamespace},
	}
	wlsopr, err := NewWlsOperatorCR(crConfig)
	if assert.NoError(err) {
		assert.Equal("WlsOperator", wlsopr.TypeMeta.Kind)
		assert.Equal("verrazzano.io/v1beta1", wlsopr.TypeMeta.APIVersion)
		assert.Equal(operatorName, wlsopr.ObjectMeta.Name)
		assert.Equal(crConfig.BindingNamespace, wlsopr.ObjectMeta.Namespace)
		assert.Equal(crConfig.Labels, wlsopr.ObjectMeta.Labels)
		assert.Equal(wlsv1beta.WlsOperatorStatus{}, wlsopr.Status)
		assert.Equal(wlsOperatorSpec, wlsopr.Spec)
	}

}
