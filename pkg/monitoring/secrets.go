// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Secrets defines the interface for handling secrets.
type Secrets interface {
	Get(name string) (*corev1.Secret, error)
	Create(*corev1.Secret) (*corev1.Secret, error)
	Update(*corev1.Secret) (*corev1.Secret, error)
	List(ns string, selector labels.Selector) (ret []*corev1.Secret, err error)
	Delete(ns, name string) error
	GetVmiPassword() (string, error)
}

// NewVmiSecret creates the necessary Secrets for the given VerrazzanoBinding.
func NewVmiSecret(binding *types.SyntheticBinding) *corev1.Secret {
	bindingLabels := util.GetLocalBindingLabels(binding)
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VmiSecretName,
			Namespace: constants.VerrazzanoNamespace,
			Labels:    bindingLabels,
		},
		Data: map[string][]byte{
			"username": []byte(constants.VmiUsername),
			"password": []byte(genPassword(16)),
		},
	}
	return sec
}

// GetVmiPassword returns the password for the VMI secret.
func GetVmiPassword(secrets Secrets) (string, error) {
	sec, err := secrets.Get(constants.VmiSecretName)
	pw := ""
	if sec != nil {
		bytes := sec.Data["password"]
		if bytes != nil {
			pw = string(bytes)
		} else {
			err = fmt.Errorf("Failed to retrieve %s password", constants.VmiUsername)
		}
	}
	return pw, err
}

var passwordChars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genPassword(passSize int) string {
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < passSize; i++ {
		b.WriteRune(passwordChars[rand.Intn(len(passwordChars))])
	}
	return b.String()
}

// CreateVmiSecrets creates/updates a VMI secret.
func CreateVmiSecrets(binding *types.SyntheticBinding, secrets Secrets) error {
	vmiSecret, _ := secrets.Get(constants.VmiSecretName)

	if vmiSecret == nil {
		vmiSecret = NewVmiSecret(binding)
		_, err := secrets.Create(vmiSecret)
		if err != nil {
			zap.S().Errorf("Failed to create VMI secret %s/%s: %v", vmiSecret.Namespace, vmiSecret.Name, err)
			return err
		}
		zap.S().Infof("Succesfully created VMI secret %s/%s", vmiSecret.Namespace, vmiSecret.Name)
	} else {
		if vmiSecret.Data["username"] == nil || vmiSecret.Data["password"] == nil {
			vmiSecret = NewVmiSecret(binding)
			_, err := secrets.Update(vmiSecret)
			if err != nil {
				zap.S().Errorf("Failed to update VMI secret %s/%s: %v", vmiSecret.Namespace, vmiSecret.Name, err)
				return err
			}
			zap.S().Infof("Succesfully updated VMI secret %s/%s", vmiSecret.Namespace, vmiSecret.Name)

		}
	}

	// Delete Secrets that shouldn't exist
	secretNames := []string{constants.VmiUsername, util.GetVmiNameForBinding(binding.Name)}
	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})
	existingSecretsList, err := secrets.List("", selector)
	if err != nil {
		return err
	}
	for _, existingSecret := range existingSecretsList {
		if !util.Contains(secretNames, existingSecret.Name) {
			zap.S().Infof("Deleting Secret %s", existingSecret.Name)
			err := secrets.Delete(existingSecret.Namespace, existingSecret.Name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
