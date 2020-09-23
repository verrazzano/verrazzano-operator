// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/stretchr/testify/assert"

	"crypto/rand"

	"gopkg.in/square/go-jose.v2/jwt"
)

type HandlerMock struct {
	res http.ResponseWriter
	req *http.Request
}

func (mock *HandlerMock) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	mock.res = res
	mock.req = req
	if res != nil {
		res.WriteHeader(200)
		res.Write([]byte("MockResponse"))
	}
}
func (mock *HandlerMock) assertNotCalled(t *testing.T) {
	assert.Nil(t, mock.res, "request should be nil")
	assert.Nil(t, mock.req, "response should be nil")
}
func (mock *HandlerMock) assertCalled(t *testing.T) {
	assert.NotNil(t, mock.res, "request should be set")
	assert.NotNil(t, mock.req, "response should be set")
	mock.req = nil
	mock.res = nil
}

type ResponseMock struct {
	header  map[string][]string
	status  int
	content string
}

func (mock *ResponseMock) Header() http.Header {
	if mock.header == nil {
		mock.header = make(map[string][]string)
	}
	return mock.header
}

func (mock *ResponseMock) Write(b []byte) (int, error) {
	mock.content = string(b)
	return len(b), nil
}

func (mock *ResponseMock) WriteHeader(statusCode int) {
	mock.status = statusCode
}

type KeyRepoMock struct {
	key *rsa.PrivateKey
	err error
}

func (mock *KeyRepoMock) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	pk := mock.key.Public()
	publicKey, ok := pk.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Invalid public key")
	}
	return publicKey, mock.err
}
func (mock *KeyRepoMock) GetPublicKeys() (bool, error) {
	return (mock.err == nil), mock.err
}

func (mock *KeyRepoMock) genToken(user string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	return genToken(user, expirationTime, mock.key)
}

func (mock *KeyRepoMock) expiredToken(user string) (string, error) {
	expirationTime := time.Unix(time.Now().Unix()-600, 0)
	return genToken(user, expirationTime, mock.key)
}

type Claims struct {
	PreferredUsername string `json:"preferred_username"`
	Expiry            int64  `json:"exp,omitempty"`
}

func genToken(user string, expirationTime time.Time, key *rsa.PrivateKey) (string, error) {
	return sign(user, expirationTime, key, jose.RS256)
}

func sign(user string, expirationTime time.Time, key *rsa.PrivateKey, algorithm jose.SignatureAlgorithm) (string, error) {
	var signingKey interface{}
	if algorithm == jose.RS256 || algorithm == jose.RS384 || algorithm == jose.RS512 {
		signingKey = key
	} else {
		signingKey = []byte("signingKey")
	}
	sig := jose.SigningKey{
		Algorithm: algorithm,
		Key:       signingKey,
	}
	signer, err := jose.NewSigner(sig, &jose.SignerOptions{})
	if err != nil {
		return "", err
	}
	builder := jwt.Signed(signer)
	claims := Claims{
		PreferredUsername: user,
		Expiry:            expirationTime.Unix(),
	}
	return builder.Claims(claims).CompactSerialize()
}

func TestNoBearerToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyRepo = &KeyRepoMock{key: key}

	mockHandler := HandlerMock{}
	authHandler := AuthHandler(&mockHandler)

	assert.HTTPError(t, authHandler.ServeHTTP, "GET", "/instance", nil)
	mockHandler.assertNotCalled(t)

	url, _ := urlpkg.Parse("/instance")
	res := ResponseMock{}
	req := http.Request{Method: "GET", URL: url}
	req.Header = map[string][]string{
		"Authorization": {"noBearerXxxx"},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 404, "NotAuthorizedOrNotFound", "")
	mockHandler.assertNotCalled(t)
}

func TestInvalidBearerToken(t *testing.T) {
	mockHandler := HandlerMock{}
	authHandler := AuthHandler(&mockHandler)
	url, _ := urlpkg.Parse("/instance")
	res := ResponseMock{}
	req := http.Request{Method: "GET", URL: url}
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyRepoMock := &KeyRepoMock{key: key}
	keyRepo = keyRepoMock
	keyRepoMock.err = nil

	req.Header = map[string][]string{
		"Authorization": {"polarBear xxx"},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 404, "NotAuthorizedOrNotFound", "")
	mockHandler.assertNotCalled(t)

	badToken := "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJXNk03ZUNuUVk3T01Zc1FHS1ZFcUtsdnlzSmlTWDFnUGhXLVZvZndlLU5vIn0.eyJleHAiOjE1ODg2MDg5OTQsImlhdCI6MTU4ODYwODY5NCwianRpIjoiN2EyODk2NzAtZDc5OC00NTFkLThiZGMtMWY3M2RlNzgxN2MwIiwiaXNzIjoiaHR0cHM6Ly9rZXljbG9hay5wYXVsMzkudjhvLm9yYWNsZWR4LmNvbS9hdXRoL3JlYWxtcy9kZXYiLCJhdWQiOiJhY2NvdW50Iiwic3ViIjoiMTViOWJkMzEtMTQyNC00OGM0LTllNjgtODYzYWE1NjgyYjdhIiwidHlwIjoiQmVhcmVyIiwiYXpwIjoiY3VybCIsInNlc3Npb25fc3RhdGUiOiI3OWMwYTMxOC0xNWNkLTRmYjMtYWY1Mi1kNTRmNGM0MjcwMzEiLCJhY3IiOiIxIiwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIm9mZmxpbmVfYWNjZXNzIiwidW1hX2F1dGhvcml6YXRpb24iXX0sInJlc291cmNlX2FjY2VzcyI6eyJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJzY29wZSI6ImVtYWlsIHByb2ZpbGUiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImNsaWVudEhvc3QiOiIxMC4yNDQuMC4wIiwiY2xpZW50SWQiOiJjdXJsIiwicHJlZmVycmVkX3VzZXJuYW1lIjoic2VydmljZS1hY2NvdW50LWN1cmwiLCJjbGllbnRBZGRyZXNzIjoiMTAuMjQ0LjAuMCJ9.jGIGM08BwVQKUmmpajVZI3sF4jW3k_aBddEnmFHxiGharscT7ghufge0-4lr7ZK4BE_J8s36ojXd59pw_ZYZCxBF87f1F0c00HmaV_trqxBu42SQCZYOinezsJOgU_HLMcWOxkZACVL-nv6jbwLUUFzlzTLvmpqg0rpr3UaYJAZtyaztopBW8Tk3oQ5X3DhOSpw0a13U1L4qskRBZXVFpEfsOkmSSJdAX_TcMPwMN4ywCaj5HCvaUtQPA3UMeizIlasYqsk63_gSPF2_lU3XRBzBFh4NZeB7dBg2gdqo09JTrTGx3KL2HJTOp9ih6xqwjj6Lg_PyvlgkOCQYwBF36g"
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", badToken)},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 404, "NotAuthorizedOrNotFound", "")
	mockHandler.assertNotCalled(t)

	expiredToken, _ := keyRepoMock.expiredToken("foo")
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", expiredToken)},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 404, "TokenExpired", "token is expired")
	mockHandler.assertNotCalled(t)
}

func TestSigningMethods(t *testing.T) {
	mockHandler := HandlerMock{}
	authHandler := AuthHandler(&mockHandler)
	url, _ := urlpkg.Parse("/instance")
	res := ResponseMock{}
	req := http.Request{Method: "GET", URL: url}
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyRepoMock := &KeyRepoMock{key: key}
	keyRepo = keyRepoMock

	wrongAlg, _ := sign("x", time.Now().Add(5*time.Minute), keyRepoMock.key, jose.HS256)
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", wrongAlg)},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 404, "NotAuthorizedOrNotFound", "algorithm")
	mockHandler.assertNotCalled(t)

	wrongAlg, _ = sign("y", time.Now().Add(5*time.Minute), keyRepoMock.key, jose.RS384)
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", wrongAlg)},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 404, "NotAuthorizedOrNotFound", "algorithm")
	mockHandler.assertNotCalled(t)

	goodToken, _ := sign("foo", time.Now().Add(5*time.Minute), keyRepoMock.key, jose.RS512)
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", goodToken)},
	}
	authHandler.ServeHTTP(&res, &req)
	assertOK(t, &res)
	mockHandler.assertCalled(t)
}

func TestKeyCloakUnavailable(t *testing.T) {
	mockHandler := HandlerMock{}
	authHandler := AuthHandler(&mockHandler)
	url, _ := urlpkg.Parse("/instance")
	res := ResponseMock{}
	req := http.Request{Method: "GET", URL: url}
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyRepoMock := &KeyRepoMock{key: key}
	keyRepo = keyRepoMock

	//keyCloak is unavailable
	keyRepoMock.err = errors.New("public key not found")
	req.Header = map[string][]string{
		"Authorization": {"Bearer xxx"},
	}
	authHandler.ServeHTTP(&res, &req)
	assertError(t, &res, 500, "InternalServerError", "public key not found")
	mockHandler.assertNotCalled(t)
	//keyCloak is up
	keyRepoMock.err = nil
	goodToken, _ := keyRepoMock.genToken("good")
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", goodToken)},
	}
	authHandler.ServeHTTP(&res, &req)
	assert.NotNil(t, req.Context().Value(keyBearerToken), "Expected parsed token")
	mockHandler.assertCalled(t)
	assertOK(t, &res)
}

func TestValidBearerToken(t *testing.T) {
	mockHandler := HandlerMock{}
	authHandler := AuthHandler(&mockHandler)
	url, _ := urlpkg.Parse("/instance")
	res := ResponseMock{}
	req := http.Request{Method: "GET", URL: url}

	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyRepoMock := &KeyRepoMock{key: key}
	keyRepo = keyRepoMock
	goodToken, _ := keyRepoMock.genToken("x")
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", goodToken)},
	}
	authHandler.ServeHTTP(&res, &req)
	assert.NotNil(t, req.Context().Value(keyBearerToken), "Expected parsed token")
	mockHandler.assertCalled(t)
	assertOK(t, &res)

	goodToken, _ = keyRepoMock.genToken("y")
	req.Header = map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer %v", goodToken)},
	}
	authHandler.ServeHTTP(&res, &req)
	assert.NotNil(t, req.Context().Value(keyBearerToken), "Expected parsed token")
	mockHandler.assertCalled(t)
	assertOK(t, &res)
}

func assertOK(t *testing.T, res *ResponseMock) {
	assert.Equal(t, 200, res.status, "Expected 200 OK")
}

func assertError(t *testing.T, res *ResponseMock, status int, code, message string) {
	assert.Equal(t, status, res.status, "Expected status")
	if code != "" {
		var err Error
		json.Unmarshal([]byte(res.content), &err)
		assert.Equal(t, code, err.Code, "Expected ErrorCode")
		assert.Contains(t, err.Message, message)
	}
}

func TestVerifyJsonWebToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyRepoMock := &KeyRepoMock{key: key}
	keyRepo = keyRepoMock
	goodToken, _ := keyRepoMock.genToken("goodJWT")
	token, err := verifyJSONWebToken(goodToken)
	assert.NotNil(t, token, "Expected good JWT")
	assert.Nil(t, err)

	expiredToken, _ := keyRepoMock.expiredToken("expiredToken")
	token, err = verifyJSONWebToken(expiredToken)
	assert.NotNil(t, token, "Expiredd JWT")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "expired")
}
