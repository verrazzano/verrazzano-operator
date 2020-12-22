// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/verrazzano/verrazzano-operator/pkg/api/instance"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

var (
	listerSet controller.Listers
)

// Init initializes set of Listers for API server.
func Init(listers controller.Listers) {
	listerSet = listers
}

// This is global for the apiServer
var apiServerRealm string
var keyRepo KeyRepo

// SetRealm sets the the realm for API server.
func SetRealm(realm string) {
	apiServerRealm = realm
	if apiServerRealm != "" {
		keyRepo = NewKeyCloak(instance.GetKeyCloakURL(), apiServerRealm)
	} else {
		keyRepo = nil
	}
}

type key int

const (
	keyBearerToken key = iota
)

// AuthHandler setups handler for API server HTTP requests.
func AuthHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		auth, err := getAuthBearer(r)
		if keyRepo != nil {
			if err != nil {
				errMsg := fmt.Sprintf("Error getting Authorization Header: %v", err)
				zap.S().Errorw(errMsg)
				invalidTokenError(w, err.Error())
				return
			}
			//Getting public key from KeyCloak
			ok, err := keyRepo.GetPublicKeys()
			if !ok || err != nil {
				errMsg := fmt.Sprintf("Error getting public key from KeyCloak: %v", err)
				zap.S().Errorw(errMsg)
				InternalServerError(w, errMsg)
				return
			}
			token, err := verifyJSONWebToken(auth)
			if err != nil {
				zap.S().Infof("%v error verifying token: %v", r.URL.Path, err)
				invalidTokenError(w, err.Error())
				return
			}
			if token == nil {
				errMsg := fmt.Sprintf("%v missing Bearer token", r.URL.Path)
				zap.S().Infow(errMsg)
				invalidTokenError(w, errMsg)
				return
			}
			verifiedRequest := r.WithContext(context.WithValue(r.Context(), keyBearerToken, token))
			*r = *verifiedRequest
		}
		h.ServeHTTP(w, r)
	})
}

// PublicKeys represents json public keys.
type PublicKeys struct {
	Keys []JSONWebKey `json:"keys"`
}

// JSONWebKey represents a cryptographic key.
//https://tools.ietf.org/html/rfc7517#section-4.7 JsonWebKey
type JSONWebKey struct {
	KeyID            string   `json:"kid"`
	KeyType          string   `json:"kty"`
	Algorithm        string   `json:"alg"`
	CertificateChain []string `json:"x5c"`
}

// KeyRepo defines interface to get public keys.
type KeyRepo interface {
	GetPublicKey(kid string) (*rsa.PublicKey, error)
	GetPublicKeys() (bool, error)
}

// KeyCloak represents a KeyCloak instance.
type KeyCloak struct {
	Endpoint string
	Realm    string
	keyCache map[string]*JSONWebKey
}

// NewKeyCloak creates a KeyCloak struct.
func NewKeyCloak(url, realm string) KeyRepo {
	return &KeyCloak{Endpoint: url, Realm: realm}
}

const certTemp = `
-----BEGIN CERTIFICATE-----
%v
-----END CERTIFICATE-----`

// GetPublicKeys returns boolean whether public keys exist or not.
func (kc *KeyCloak) GetPublicKeys() (bool, error) {
	if kc.keyCache == nil {
		err := kc.refreshKeyCache()
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// GetPublicKey returns a public key.
func (kc *KeyCloak) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	key := kc.keyCache[kid]
	if key == nil {
		errMsg := "Public Key not found"
		err := kc.refreshKeyCache()
		if err != nil {
			errMsg = err.Error()
		}
		if err != nil || len(kc.keyCache) == 0 {
			return nil, errors.New(errMsg)
		}
		key = kc.keyCache[kid]
		if key == nil || key.CertificateChain == nil || len(key.CertificateChain) == 0 {
			return nil, errors.New(errMsg)
		}
	}
	block, _ := pem.Decode([]byte(fmt.Sprintf(certTemp, key.CertificateChain[0])))
	if block == nil {
		return nil, errors.New("Invalid public key: key must be PEM encoded")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err == nil {
		publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("Invalid public key: %T is not a valid RSA public key", cert.PublicKey)
		}
		return publicKey, nil
	}

	return nil, err
}

const certsURL = "%v/auth/realms/%v/protocol/openid-connect/certs"

func (kc *KeyCloak) refreshKeyCache() error {
	cert, err := kc.getPublicKeys()
	if err != nil || cert.Keys == nil || len(cert.Keys) == 0 {
		return err
	}
	kc.keyCache = make(map[string]*JSONWebKey)
	for _, k := range cert.Keys {
		if k.CertificateChain != nil && len(k.CertificateChain) > 0 && k.KeyID != "" {
			kc.keyCache[k.KeyID] = &k
		}
	}
	return nil
}

// Call Keycloak to get the public keys.  This is only called
// when the public key id specified by the JWT token is not
// cached, which should not happen often.
func (kc *KeyCloak) getPublicKeys() (*PublicKeys, error) {
	url := fmt.Sprintf(certsURL, kc.Endpoint, kc.Realm)
	httpClient := getKeyCloakClient()
	httpClient.RetryMax = 10
	zap.S().Infow(fmt.Sprintf("Calling KeyCloak to get the public keys at url " + url))
	res, err := httpClient.Get(url)
	if err != nil {
		zap.S().Errorw(fmt.Sprintf("Error %s calling %s to get certs ", err.Error(), url))
		return &PublicKeys{}, err
	}
	if res.StatusCode == http.StatusNotFound {
		zap.S().Errorw(fmt.Sprintf("HTTP StatusNotFound calling %s to get certs ", url))
		return &PublicKeys{}, fmt.Errorf("Failed retrieving Public Key from %s", url)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var publicKeys PublicKeys
	err = json.Unmarshal(body, &publicKeys)
	return &publicKeys, err
}

func invalidTokenError(w http.ResponseWriter, message string) {
	errorCode := "NotAuthorizedOrNotFound"
	if strings.Contains(strings.ToLower(message), "expired") {
		errorCode = "TokenExpired"
	}
	HTTPError(w, http.StatusNotFound, errorCode, message)
}

// Get client used to call keycloak
func getKeyCloakClient() *retryablehttp.Client {
	// Get the cert
	caData := getCACert(*listerSet.KubeClientSet)

	// Create Transport object
	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{RootCAs: rootCertPool(caData)},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	setupHTTPResolve(tr)

	// Replace inner http client with client that uses the Transport object
	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{Transport: tr, Timeout: 300 * time.Second}
	return client
}

func setupHTTPResolve(tr *http.Transport) {
	host := util.GetRancherHost()
	if host != "" {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			if strings.Contains(addr, "127.0.0.1") && strings.Contains(addr, ":443") {
				zap.S().Debugf("address modified from %s to %s \n", addr, host+":443")
				addr = host + ":443"
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}
}

// get the ca.crt from secret "<vz-env-name>-secret" in namespace "verrazzano-system"
func getCACert(kubeClientSet kubernetes.Interface) []byte {
	secretResName := instance.GetVerrazzanoName() + "-secret"
	certSecret, err := kubeClientSet.CoreV1().Secrets(constants.VerrazzanoSystem).Get(context.TODO(), secretResName, metav1.GetOptions{})
	if err != nil {
		zap.S().Warnf("Error getting secret %s/%s in management cluster: %s", constants.VerrazzanoSystem, secretResName, err.Error())
		return []byte{}
	}
	if certSecret == nil {
		zap.S().Warnf("Secret %s/%s not found in management cluster", constants.VerrazzanoSystem, secretResName)
		return []byte{}
	}
	return certSecret.Data["ca.crt"]
}

func rootCertPool(caData []byte) *x509.CertPool {
	if len(caData) == 0 {
		return nil
	}
	// if we have caData, use it
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	return certPool
}

func getAuthBearer(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header with Bearer {token} is required")
	}
	return getBearerToken(authHeader)
}

func getBearerToken(authHeader string) (string, error) {
	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || !strings.EqualFold(authHeaderParts[0], "Bearer") {
		return "", errors.New("Authorization header format must be Bearer {token}")
	}
	return authHeaderParts[1], nil
}

var validSigningAlgorithms = []string{"RS256", "RS512"}

func validateSigningAlgorithm(algorithm string) bool {
	for _, alg := range validSigningAlgorithms {
		if algorithm == alg {
			return true
		}
	}
	return false
}

func verifyJSONWebToken(auth string) (*jwt.JSONWebToken, error) {
	joseToken, err := jwt.ParseSigned(auth)
	if err != nil {
		return nil, err
	}
	var header *jose.Header = nil
	if joseToken != nil && joseToken.Headers != nil {
		for _, jhr := range joseToken.Headers {
			if !validateSigningAlgorithm(jhr.Algorithm) {
				return nil, fmt.Errorf("Invalid signing algorithm %s", jhr.Algorithm)
			}
			header = &jhr
		}
	}
	if header == nil {
		return nil, errors.New("Invalid JSONWebToken headers")
	}
	publicKey, err := keyRepo.GetPublicKey(header.KeyID)
	if err != nil {
		return nil, err
	}
	var claims jwt.Claims
	err = joseToken.Claims(publicKey, &claims)
	if err == nil {
		err = claims.Validate(jwt.Expected{Time: time.Now()})
	}
	return joseToken, err
}
