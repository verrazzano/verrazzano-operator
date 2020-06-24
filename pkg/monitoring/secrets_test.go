package monitoring

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/pbkdf2"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type MockSecrets struct {
	secrets map[string]*corev1.Secret
}

func (ms *MockSecrets) Get(name string) (*corev1.Secret, error) {
	return ms.secrets[name], nil
}

func (ms *MockSecrets) Create(newSecret *corev1.Secret) (*corev1.Secret, error) {
	ms.secrets[newSecret.Name] = newSecret
	return newSecret, nil
}

func (ms *MockSecrets) Update(newSecret *corev1.Secret) (*corev1.Secret, error) {
	ms.secrets[newSecret.Name] = newSecret
	return newSecret, nil
}

func (ms *MockSecrets) List(ns string, selector labels.Selector) (ret []*corev1.Secret, err error) {
	return []*corev1.Secret{}, nil
}

func (ms *MockSecrets) Delete(ns, name string) error {
	delete(ms.secrets, name)
	return nil
}

func (ms *MockSecrets) GetVmiPassword() (string, error) {
	return GetVmiPassword(ms)
}

func TestExistingVmiSecrets(t *testing.T) {
	binding := v1beta1v8o.VerrazzanoBinding{}
	binding.Name = "TestExistingVmiSecrets"
	existing := NewVmiSecret(&binding)
	secrets := MockSecrets{secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: existing,
	}}
	CreateVmiSecrets(&binding, &secrets)
	sec, _ := GetVmiPassword(&secrets)
	assert.Equal(t, string(existing.Data["password"]), sec, "existing vmi secret")
	assertSaltedHash(t, &secrets)
}

func assertSaltedHash(t *testing.T, secrets Secrets) {
	sec, _ := secrets.Get(constants.VmiSecretName)
	saltString := base64.StdEncoding.EncodeToString(sec.Data["salt"])
	salt, _ := base64.StdEncoding.DecodeString(saltString)
	hash := pbkdf2.Key(sec.Data["password"], salt, 27500, 64, sha256.New)
	hashString := base64.StdEncoding.EncodeToString(sec.Data["hash"])
	assert.Equal(t, hashString, base64.StdEncoding.EncodeToString(hash), "expected SaltedHash")
}

func TestNewVmiRandomPassword(t *testing.T) {
	binding := v1beta1v8o.VerrazzanoBinding{}
	binding.Name = "TestNewVmiRandomPassword"
	secrets := &MockSecrets{secrets: map[string]*corev1.Secret{}}
	CreateVmiSecrets(&binding, secrets)
	sec1, _ := GetVmiPassword(secrets)
	assertSaltedHash(t, secrets)
	secrets.Delete(constants.VerrazzanoNamespace, constants.VmiSecretName)

	CreateVmiSecrets(&binding, secrets)
	sec2, _ := GetVmiPassword(secrets)
	assertSaltedHash(t, secrets)
	secrets.Delete(constants.VerrazzanoNamespace, constants.VmiSecretName)

	CreateVmiSecrets(&binding, secrets)
	sec3, _ := GetVmiPassword(secrets)
	assertSaltedHash(t, secrets)
	secrets.Delete(constants.VerrazzanoNamespace, constants.VmiSecretName)

	assert.NotEqual(t, sec1, sec2, "new random password")
	assert.NotEqual(t, sec3, sec2, "new random password")
	assert.NotEqual(t, sec3, sec1, "new random password")
}
