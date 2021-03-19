// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"
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
			"password": []byte(genPassword(10)),
		},
	}
	return saltedHash(sec)
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

func saltedHash(sec *corev1.Secret) *corev1.Secret {
	salt := make([]byte, 16, 16+sha1.Size)
	io.ReadFull(crand.Reader, salt)
	pw := sec.Data["password"]
	sec.Data["salt"] = salt
	sec.Data["hash"] = pbkdf2.Key(pw, salt, 27500, 64, sha256.New)
	zap.S().Debugf("Creating/updating %s secret %s", sec.Namespace, sec.Name)
	return sec
}

// CreateVmiSecrets creates/updates a VMI secret.
func CreateVmiSecrets(binding *types.SyntheticBinding, secrets Secrets) error {
	vmiSecret, _ := secrets.Get(constants.VmiSecretName)

	if vmiSecret == nil {
		vmiSecret = NewVmiSecret(binding)
		_, err := secrets.Create(vmiSecret)
		if err != nil {
			return err
		}
	} else {
		updated := false
		if vmiSecret.Data["username"] == nil || vmiSecret.Data["password"] == nil {
			vmiSecret = NewVmiSecret(binding)
			updated = true
		}
		if vmiSecret.Data["salt"] == nil || vmiSecret.Data["hash"] == nil {
			vmiSecret = saltedHash(vmiSecret)
			updated = true
		}
		if updated {
			_, err := secrets.Update(vmiSecret)
			if err != nil {
				return err
			}
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

// GetSystemSecrets the secrets to be created for Filebeats and Journalbeats.
func GetSystemSecrets(sec Secrets, clusterInfo ClusterInfo) []*corev1.Secret {
	var secrets []*corev1.Secret
	if isManagedCluster(clusterInfo) {
		fileabeatSecret := createAdminLoggingSecret(constants.LoggingNamespace, constants.FilebeatName, clusterInfo)
		journalbeatSecret := createAdminLoggingSecret(constants.LoggingNamespace, constants.JournalbeatName, clusterInfo)
		secrets = append(secrets, fileabeatSecret, journalbeatSecret)
	} else {
		password, err := sec.GetVmiPassword()
		if err != nil {
			zap.S().Errorf("Failed to retrieve secret %v", err)
		}
		fileabeatSecret := createLocalLoggingSecret(constants.LoggingNamespace, constants.FilebeatName, password)
		journalbeatSecret := createLocalLoggingSecret(constants.LoggingNamespace, constants.JournalbeatName, password)
		secrets = append(secrets, fileabeatSecret, journalbeatSecret)
	}
	return secrets
}

func createLocalLoggingSecret(namespace, name, password string) *corev1.Secret {
	loggingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-secret",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"username": []byte(constants.VmiUsername),
			"password": []byte(password),
		},
	}
	return loggingSecret
}

func createAdminLoggingSecret(namespace, name string, clusterInfo ClusterInfo) *corev1.Secret {
	loggingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-secret",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"username":  []byte(clusterInfo.ElasticsearchUsername),
			"password":  []byte(clusterInfo.ElasticsearchPassword),
			"es-url":    []byte(clusterInfo.ElasticsearchURL),
			"ca-bundle": clusterInfo.ElasticsearchCABundle,
		},
	}
	return loggingSecret
}
