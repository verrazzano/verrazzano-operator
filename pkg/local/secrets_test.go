// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	fakek8s "k8s.io/client-go/kubernetes/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateGenericSecret(t *testing.T) {
	kubecli := fakek8s.NewSimpleClientset()
	sec := newSecret("v8o-test", "mySecret", "mySecret")
	sec.UID = "uid01"
	CreateGenericSecret(*sec, kubecli)
	get, err := GetSecret(sec.Name, sec.Namespace, kubecli)
	assert.Nil(t, err, "GetSecret error")
	assert.Equal(t, sec.UID, get.UID, "Expected Secret")

	get, b, err := GetSecretByUID(kubecli, sec.Namespace, string(sec.UID))
	assert.Nil(t, err, "GetSecretByUID error")
	assert.True(t, b, "Expected Secret found")
	assert.Equal(t, sec.UID, get.UID, "Expected Secret")
}

func TestCreateGenericSecretWithError(t *testing.T) {
	//kubecli := fakek8s.NewSimpleClientset()
	kubecli := testutil.MockError(fakek8s.NewSimpleClientset(),
		"create", "secrets", &corev1.Secret{})
	sec := newSecret("v8o-test", "mySecret", "mySecret")
	err := CreateGenericSecret(*sec, kubecli)
	assert.NotNil(t, err, "Expected CreateGenericSecret error")
}

func TestGetSecretWithError(t *testing.T) {
	sec := newSecret("v8o-test", "mySecret", "mySecret")
	kubecli := testutil.MockError(fakek8s.NewSimpleClientset(sec),
		"get", "secrets", &corev1.Secret{})
	_, err := GetSecret(sec.Name, sec.Namespace, kubecli)
	assert.NotNil(t, err, "Expected GetSecret error")
}

func TestGetSecretByUIDWithError(t *testing.T) {
	sec := newSecret("v8o-test", "mySecret", "mySecret")
	sec.UID = "uid01"
	kubecli := testutil.MockError(fakek8s.NewSimpleClientset(sec),
		"list", "secrets", &corev1.SecretList{})
	_, b, err := GetSecretByUID(kubecli, sec.Namespace, string(sec.UID))
	assert.NotNil(t, err, "Expected GetSecretByUID error")
	assert.False(t, b, "Expected Secret notfound")
}

func TestUpdateAcmeDNSSecret(t *testing.T) {
	ns := constants.VerrazzanoNamespace
	acmeDNSKey := constants.AcmeDNSSecretKey
	name := "TestUpdateAcmeDNSSecret"
	verrazzanoURI := "VerrazzanoURI"
	sec := newSecret(ns, name, name)
	data := map[string]acmeDNS{"acmeDNS": {}}
	b, _ := json.Marshal(data)
	sec.Data = map[string][]byte{acmeDNSKey: b}
	kubecli := fakek8s.NewSimpleClientset(sec)
	var secretLister = testutil.NewSecretLister(kubecli)
	var binding types.ResourceLocation
	binding.Name = "system"
	err := UpdateAcmeDNSSecret(&binding, kubecli, secretLister, name, verrazzanoURI)
	bindingDNSName := fmt.Sprintf("vmi.%s.%s", binding.Name, verrazzanoURI)
	assert.Nil(t, err, "UpdateAcmeDNSSecret error")
	get, _ := kubecli.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
	var dnsCred map[string]acmeDNS
	json.Unmarshal(get.Data[acmeDNSKey], &dnsCred)
	assert.Equal(t, 3, len(dnsCred), "Expected size of updated AcmeDNSSecret")
	assert.NotNil(t, dnsCred[bindingDNSName], "Expected content of updated AcmeDNSSecret")
}

func TestUpdateAcmeDNSSecretWithUpdateError(t *testing.T) {
	ns := constants.VerrazzanoNamespace
	acmeDNSKey := constants.AcmeDNSSecretKey
	name := "TestUpdateAcmeDNSSecretWithUpdateError"
	verrazzanoURI := "VerrazzanoURI"
	sec := newSecret(ns, name, name)
	b, _ := json.Marshal(map[string]acmeDNS{"acmeDNS": {}})
	sec.Data = map[string][]byte{acmeDNSKey: b}
	kubecli := testutil.MockError(fakek8s.NewSimpleClientset(sec),
		"update", "secrets", &corev1.Secret{})
	var secretLister = testutil.NewSecretLister(kubecli)
	var binding types.ResourceLocation
	binding.Name = "system"
	err := UpdateAcmeDNSSecret(&binding, kubecli, secretLister, name, verrazzanoURI)
	assert.NotNil(t, err, "Expected error in UpdateAcmeDNSSecret")
}

func newSecret(namespace, name, secret string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque, //corev1.SecretTypeOpaque
		StringData: map[string]string{
			"password": secret,
			"username": name,
		},
	}
}

func TestDeleteSecret(t *testing.T) {
	sec := newSecret("v8o-test", "TestDeleteSecret", "TestDeleteSecret")
	kubecli := fakek8s.NewSimpleClientset(sec)
	err := DeleteSecret(kubecli, sec)
	assert.Nil(t, err, "Expected nil error")
	sec, _ = kubecli.CoreV1().Secrets(sec.Namespace).Get(context.TODO(), sec.Name, metav1.GetOptions{})
	assert.Nil(t, sec, "Expected nil Secret")
}

func TestDeleteSecrets(t *testing.T) {
	sec := newSecret("v8o-test", "TestDeleteSecret", "TestDeleteSecret")
	kubecli := fakek8s.NewSimpleClientset(sec)
	var binding types.ResourceLocation
	binding.Name = "system"
	secretLister := testutil.NewSecretLister(kubecli)
	err := DeleteSecrets(&binding, kubecli, secretLister)
	assert.Nil(t, err, "Expected nil error")
	sec, _ = kubecli.CoreV1().Secrets(sec.Namespace).Get(context.TODO(), sec.Name, metav1.GetOptions{})
	assert.Nil(t, sec, "Expected nil Secret")
}

func TestUpdateSecret(t *testing.T) {
	sec := newSecret("v8o-test", "TestUpdateSecret", "TestUpdateSecret")
	kubecli := fakek8s.NewSimpleClientset(sec)
	var binding types.ResourceLocation
	binding.Name = "system"
	err := UpdateSecret(kubecli, sec)
	assert.Nil(t, err, "Expected nil error")
}

func TestUpdateAcmeDNSSecretWithErrors(t *testing.T) {
	var binding types.ResourceLocation
	binding.Name = "system"
	verrazzanoURI := "VerrazzanoURI"
	ns := constants.VerrazzanoNamespace
	type args struct {
		name    string
		secName string
		secData map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "NotFoundError",
			args: args{
				name:    "NotFoundError",
				secName: "mySec",
			},
			wantErr: false,
		},
		{
			name: "JsonError",
			args: args{
				name:    "TestJsonError",
				secName: "TestJsonError",
				secData: map[string][]byte{constants.AcmeDNSSecretKey: []byte("hello")},
			},
			wantErr: true,
		},
		{
			name: "DataContentError",
			args: args{
				name:    "DataContentError",
				secName: "DataContentError",
				secData: map[string][]byte{constants.AcmeDNSSecretKey: []byte("{}")},
			},
			wantErr: true,
		},
		{
			name: "DataNotFoundError",
			args: args{
				name:    "DataNotFoundError",
				secName: "DataNotFoundError",
				secData: map[string][]byte{"wrongKey": []byte("hello")},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := newSecret(ns, tt.args.secName, tt.args.secName)
			if tt.args.secData != nil {
				secret.Data = tt.args.secData
			}
			kubecli := fakek8s.NewSimpleClientset(secret)
			seclist := testutil.NewSecretLister(kubecli)
			if err := UpdateAcmeDNSSecret(&binding, kubecli, seclist, tt.args.name, verrazzanoURI); (err != nil) != tt.wantErr {
				t.Errorf("UpdateAcmeDNSSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteSecretsWithErrors(t *testing.T) {
	sec := newSecret("v8o-test-TestDeleteSecretsWithErrors", "TestDeleteSecretsWithErrors", "TestDeleteSecretsWithErrors")
	var binding types.ResourceLocation
	binding.Name = "system"
	tests := []struct {
		name    string
		kubecli kubernetes.Interface
		wantErr bool
	}{
		{
			name: "TestListError",
			kubecli: testutil.MockError(fakek8s.NewSimpleClientset(sec),
				"list", "secrets", &corev1.SecretList{}),
			wantErr: true,
		},
		{
			name: "TestDeleteError",
			kubecli: testutil.MockError(fakek8s.NewSimpleClientset(sec),
				"delete", "secrets", &corev1.Secret{}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretLister := testutil.NewSecretLister(tt.kubecli)
			if err := DeleteSecrets(&binding, tt.kubecli, secretLister); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecrets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
