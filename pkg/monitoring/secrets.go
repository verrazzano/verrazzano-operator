// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"golang.org/x/crypto/pbkdf2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type Secrets interface {
	Get(name string) (*corev1.Secret, error)
	Create(*corev1.Secret) (*corev1.Secret, error)
	Update(*corev1.Secret) (*corev1.Secret, error)
	List(ns string, selector labels.Selector) (ret []*corev1.Secret, err error)
	Delete(ns, name string) error
	GetVmiPassword() (string, error)
}

// Constructs the necessary Secrets for the given VerrazzanoBinding
func NewVmiSecret(binding *v1beta1v8o.VerrazzanoBinding) *corev1.Secret {
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

func GetVmiPassword(secrets Secrets) (string, error) {
	sec, err := secrets.Get(constants.VmiSecretName)
	pw := ""
	if sec != nil {
		bytes := sec.Data["password"]
		if bytes != nil {
			pw = string(bytes)
		} else {
			err = errors.New(fmt.Sprintf("Failed to retrieve %s password", constants.VmiUsername))
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
	// Create log instance for getting salted hash
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Secret").Str("name", sec.Name).Logger()

	salt := make([]byte, 16, 16+sha1.Size)
	io.ReadFull(crand.Reader, salt)
	pw := sec.Data["password"]
	sec.Data["salt"] = salt
	sec.Data["hash"] = pbkdf2.Key(pw, salt, 27500, 64, sha256.New)
	logger.Debug().Msgf("Creating/updating %s secret %s", sec.Namespace, sec.Name)
	return sec
}

func CreateVmiSecrets(binding *v1beta1v8o.VerrazzanoBinding, secrets Secrets) error {
	// Create log instance for creating vmi secrets
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Binding").Str("name", binding.Name).Logger()

	vmiSecret, err := secrets.Get(constants.VmiSecretName)

	if vmiSecret == nil {
		vmiSecret = NewVmiSecret(binding)
		vmiSecret, err = secrets.Create(vmiSecret)
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
			vmiSecret, err = secrets.Update(vmiSecret)
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
			logger.Info().Msgf("Deleting Secret %s", existingSecret.Name)
			err := secrets.Delete(existingSecret.Namespace, existingSecret.Name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Create all the secrets needed by Filebeats and Journalbeats in all the managed cluster
func GetSystemSecrets(sec Secrets) []*corev1.Secret {
	// Create log instance for getting system secrets
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Secrets").Str("name", "Retrieve").Logger()

	var secrets []*corev1.Secret
	password, err := sec.GetVmiPassword()
	if err != nil {
		logger.Error().Msgf("Failed to retrieve secret %v", err)
	}
	fileabeatSecret, err := createLoggingSecret(constants.LoggingNamespace, constants.FilebeatName, password)
	if err != nil {
		logger.Debug().Msgf("New logging secret %s is giving error %s", constants.FilebeatName, err)
	}
	journalbeatSecret, err := createLoggingSecret(constants.LoggingNamespace, constants.JournalbeatName, password)
	if err != nil {
		logger.Debug().Msgf("New logging secret %s is giving error %s", constants.JournalbeatName, err)
	}
	secrets = append(secrets, fileabeatSecret, journalbeatSecret)
	return secrets
}

// Constructs the necessary secrets for logging
func createLoggingSecret(namespace, name, password string) (*corev1.Secret, error) {
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
	return loggingSecret, nil
}
