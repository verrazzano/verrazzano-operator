// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dgrijalva/jwt-go"
	jwtreq "github.com/dgrijalva/jwt-go/request"
	"github.com/golang/glog"
	"github.com/verrazzano/verrazzano-operator/pkg/api/instance"

	"github.com/hashicorp/go-retryablehttp"
)

var (
	listerSet controller.Listers
)

func Init(listers controller.Listers) {
	listerSet = listers
}

// This is global for the apiServer
var apiServerRealm string
var keyRepo KeyRepo

func SetRealm(realm string) {
	apiServerRealm = realm
	if apiServerRealm != "" {
		keyRepo = NewKeyCloak(instance.GetKeyCloakUrl(), apiServerRealm)
	} else {
		keyRepo = nil
	}
	jwt.TimeFunc = func() time.Time {
		return time.Now().Add(time.Second)
	}
}

//context property name
const BearerToken = "BearerToken"

var parser = jwt.Parser{ValidMethods: []string{"RS256"}}

var parserOpt = jwtreq.WithParser(&parser)

func AuthHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var keyFunc jwt.Keyfunc = nil
		var err error
		//Getting public key from KeyCloak
		if keyRepo != nil {
			keyFunc, err = keyRepo.KeyFunc()
			if err != nil {
				errMsg := fmt.Sprintf("Error getting public key from KeyCloak: %v", err)
				glog.Error(errMsg)
				InternalServerError(w, errMsg)
				return
			}
		}
		token, err := jwtreq.ParseFromRequest(r, jwtreq.AuthorizationHeaderExtractor, keyFunc, parserOpt)
		if keyRepo != nil {
			if err != nil {
				glog.V(5).Infof("%v error verifying token: %v", r.URL.Path, err)
				InvalidTokenError(w, err.Error())
				return
			}
			if token == nil {
				errMsg := fmt.Sprintf("%v missing Bearer token", r.URL.Path)
				glog.V(5).Infoln(errMsg)
				InvalidTokenError(w, errMsg)
				return
			}
			verifiedRequest := r.WithContext(context.WithValue(r.Context(), BearerToken, token))
			*r = *verifiedRequest
		}
		h.ServeHTTP(w, r)
	})
}

type PublicKeys struct {
	Keys []JsonWebKey `json:"keys"`
}

//https://tools.ietf.org/html/rfc7517#section-4.7 JsonWebKey
type JsonWebKey struct {
	KeyId            string   `json:"kid"`
	KeyType          string   `json:"kty"`
	Algorithm        string   `json:"alg"`
	CertificateChain []string `json:"x5c"`
}

type KeyRepo interface {
	KeyFunc() (jwt.Keyfunc, error)
}

type KeyCloak struct {
	Endpoint string
	Realm    string
	keyCache map[string]*JsonWebKey
}

func NewKeyCloak(url, realm string) KeyRepo {
	return &KeyCloak{Endpoint: url, Realm: realm}
}

const certTemp = `
-----BEGIN CERTIFICATE-----
%v
-----END CERTIFICATE-----`

func (kc *KeyCloak) KeyFunc() (jwt.Keyfunc, error) {
	if kc.keyCache == nil {
		kc.keyCache = make(map[string]*JsonWebKey)
	}
	return func(token *jwt.Token) (interface{}, error) {
		kid := token.Header["kid"]
		key := kc.keyCache[kid.(string)]
		if key == nil {
			errMsg := "Public Key not found"
			err := kc.refreshKeyCache()
			if err != nil {
				errMsg = err.Error()
			}
			if err != nil || kc.keyCache == nil || len(kc.keyCache) == 0 {
				return nil, errors.New(errMsg)
			}
			key = kc.keyCache[kid.(string)]
			if key == nil {
				return nil, errors.New(errMsg)
			}
		}
		return jwt.ParseRSAPublicKeyFromPEM([]byte(fmt.Sprintf(certTemp, key.CertificateChain[0])))
	}, nil
}

const certsUrl = "%v/auth/realms/%v/protocol/openid-connect/certs"

func (kc *KeyCloak) refreshKeyCache() error {
	cert, err := kc.getPublicKeys()
	if err != nil || cert.Keys == nil || len(cert.Keys) == 0 {
		return err
	} else {
		kc.keyCache = make(map[string]*JsonWebKey)
		for _, k := range cert.Keys {
			if k.CertificateChain != nil && len(k.CertificateChain) > 0 && k.KeyId != "" {
				kc.keyCache[k.KeyId] = &k
			}
		}
		return nil
	}
}

// Call Keycloak to get the public keys.  This is only called
// when the public key id specified by the JWT token is not
// cached, which should not happen often.
func (kc *KeyCloak) getPublicKeys() (*PublicKeys, error) {
	url := fmt.Sprintf(certsUrl, kc.Endpoint, kc.Realm)
	httpClient := getKeyCloakClient()
	httpClient.RetryMax = 10
	glog.Info(fmt.Sprintf("Calling KeyCloak to get the public keys at url " + url))
	res, err := httpClient.Get(url)
	if err != nil {
		glog.Error(fmt.Sprintf("Error %s calling %s to get certs ", err.Error(), url))
		return &PublicKeys{}, err
	}
	if res.StatusCode == http.StatusNotFound {
		glog.Error(fmt.Sprintf("HTTP StatusNotFound calling %s to get certs ", url))
		return &PublicKeys{}, errors.New(fmt.Sprintf("Failed retrieving Public Key from %s", url))
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return &PublicKeys{}, err
	}
	var publicKeys PublicKeys
	err = json.Unmarshal(body, &publicKeys)
	return &publicKeys, err
}

func InvalidTokenError(w http.ResponseWriter, message string) {
	errorCode := "NotAuthorizedOrNotFound"
	if strings.Contains(strings.ToLower(message), "expired") {
		errorCode = "TokenExpired"
	}
	HttpError(w, http.StatusNotFound, errorCode, message)
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

	// Replace inner http client with client that uses the Transport object
	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{Transport: tr, Timeout: 300 * time.Second}
	return client
}

// get the ca.crt from secret "<vz-env-name>-secret" in namespace "verrazzano-system"
func getCACert(kubeClientSet kubernetes.Interface) []byte {
	secretResName := instance.GetVerrazzanoName() + "-secret"
	certSecret, err := kubeClientSet.CoreV1().Secrets(constants.VerrazzanoSystem).Get(context.TODO(), secretResName, metav1.GetOptions{})
	if err != nil {
		glog.Warningf("Error getting secret %s/%s in management cluster: %s", constants.VerrazzanoSystem, secretResName, err.Error())
		return []byte{}
	}
	if certSecret == nil {
		glog.Warningf("Secret %s/%s not found in management cluster", constants.VerrazzanoSystem, secretResName)
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
