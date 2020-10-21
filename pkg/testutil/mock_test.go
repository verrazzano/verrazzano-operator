// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"testing"
)

// GIVEN a mock with a single no-arg function expectation and matching invocation
// WHEN Verify() is called on the mock
// THEN the mock verifies
func TestVerifyNoArg(t *testing.T) {
	mock := newTestMock()

	// set expectation
	mock.foo()

	mock.SetupComplete()

	// invoke method
	mock.foo()

	mock.Verify(t)
}

// GIVEN a mock with a single expectation and multiple matching invocations
// WHEN Verify() is called on the mock
// THEN the mock verifies
func TestVerifySingleExpectationMultipleInvocations(t *testing.T) {
	mock := newTestMock()

	// set expectation
	mock.foo()

	mock.SetupComplete()

	// invoke method
	mock.foo()
	mock.foo()
	mock.foo()

	mock.Verify(t)
}

// GIVEN a mock with a function expectation that has a map arg and a matching invocation
// WHEN Verify() is called on the mock
// THEN the mock verifies
func TestVerifyMapArg(t *testing.T) {
	testMap := map[string]string{"test": "it"}
	s := "test-string"

	mock := newTestMock()

	// set expectation
	mock.bar(s, testMap)

	mock.SetupComplete()

	mock.bar(s, map[string]string{"test": "it"})

	mock.Verify(t)
}

// GIVEN a mock with a function expectation that has an ANY arg and a matching invocation
// WHEN Verify() is called on the mock
// THEN the mock verifies
func TestVerifyAnyArg(t *testing.T) {
	mock := newTestMock()

	// set expectation with an 'Any' arg
	// this should match any Secrets
	mock.foobar(AnySecrets{})

	mock.SetupComplete()

	mock.foobar(&controller.KubeSecrets{})

	mock.Verify(t)
}

// GIVEN a mock with multiple function expectations and matching invocations
// WHEN Verify() is called on the mock
// THEN the mock verifies
func TestVerifyMultipleCalls(t *testing.T) {
	mock := newTestMock()

	mock.foo()
	mock.bar("test", map[string]string{"expectedKey": "expectedValue"})
	mock.foo()
	mock.foobar(AnySecrets{})

	mock.SetupComplete()

	mock.foo()
	mock.bar("test", map[string]string{"expectedKey": "expectedValue"})
	mock.foo()
	mock.foobar(&controller.KubeSecrets{})

	mock.Verify(t)
}

// GIVEN a mock with an invocation and no expectations
// WHEN Verify() is called on the mock
// THEN the mock verifies
func TestVerifyCallsWithNoExpectation(t *testing.T) {
	mock := newTestMock()
	// no expectations
	mock.SetupComplete()
	// invocation with no expectation is ok with 'nice' verify
	mock.foo()

	mock.Verify(t)
}

// GIVEN a mock with multiple function expectations and matching invocations
// WHEN VerifyStrict() is called on the mock
// THEN the mock verifies
func TestVerifyStrict(t *testing.T) {
	mock := newTestMock()

	mock.foo()
	mock.bar("test", map[string]string{"expectedKey": "expectedValue"})

	mock.SetupComplete()

	mock.foo()
	mock.bar("test", map[string]string{"expectedKey": "expectedValue"})

	mock.VerifyStrict(t)
}

// GIVEN a mock with a single function expectation and no matching invocations
// WHEN VerifyStrict() is called on the mock
// THEN verification fails
func TestVerifyNegativeMissingInvocation(t *testing.T) {
	mock := newTestMock()

	// set expectation
	mock.foo()

	mock.SetupComplete()

	tt := &testing.T{}
	// call Verify without invoking foo() should result in Verify failing
	mock.Verify(tt)

	assert.True(t, tt.Failed())
}

// GIVEN a mock with a single function expectation and an invocation with args that don't match
// WHEN Verify() is called on the mock
// THEN verification fails
func TestVerifyNegativeArgMismatch(t *testing.T) {
	mock := newTestMock()

	// set expectation on mock
	mock.bar("test", map[string]string{"expectedKey": "expectedValue"})

	mock.SetupComplete()

	// invoke bar with non-equal map arg
	mock.bar("test", map[string]string{"expectedKey": "definitelyNotExpectedValue"})
	tt := &testing.T{}
	// non-matching map arg should result in Verify failing
	mock.Verify(tt)

	assert.True(t, tt.Failed())
}

// GIVEN a mock with a single function expectation and an invocation with args that don't match
// WHEN VerifyStrict() is called on the mock
// THEN verification fails
func TestVerifyStrictNegativeArgMismatch(t *testing.T) {
	mock := newTestMock()

	// set expectation on mock
	mock.bar("test", map[string]string{"expectedKey": "expectedValue"})

	mock.SetupComplete()

	// invoke bar with non-equal map arg
	mock.bar("test", map[string]string{"expectedKey": "definitelyNotExpectedValue"})
	tt := &testing.T{}
	// non-matching map arg should result in Verify failing
	mock.Verify(tt)

	assert.True(t, tt.Failed())
}

// GIVEN a mock with a single expectation and multiple matching invocations
// WHEN VerifyStrict() is called on the mock
// THEN mock verification fails
func TestVerifyStrictSingleExpectationMultipleInvocations(t *testing.T) {
	mock := newTestMock()

	mock.foo()

	mock.SetupComplete()

	mock.foo()
	mock.foo()

	tt := &testing.T{}
	// too many invocations of foo() should result in VerifyStrict failing
	mock.VerifyStrict(tt)

	assert.True(t, tt.Failed())
}

// GIVEN a mock with a single expectation and no matching invocations
// WHEN VerifyStrict() is called on the mock
// THEN mock verification fails
func TestVerifyStrictNegativeNoInvocation(t *testing.T) {
	mock := newTestMock()

	mock.foo()

	mock.SetupComplete()

	tt := &testing.T{}
	mock.VerifyStrict(tt)

	assert.True(t, tt.Failed())
}

// GIVEN a mock with an invocation and no expectation
// WHEN VerifyStrict() is called on the mock
// THEN mock verification fails
func TestVerifyStrictNegativeInvocationWithNoExpectation(t *testing.T) {
	mock := newTestMock()

	mock.SetupComplete()

	mock.foo()

	tt := &testing.T{}
	// invocation with no expectation should result in verifyStrict failure
	mock.VerifyStrict(tt)

	assert.True(t, tt.Failed())
}

// GIVEN a mock with multiple a single expectation and a matching invocation
//   AND another invocation with no matching expectation
// WHEN VerifyStrict() is called on the mock
// THEN mock verification fails
func TestVerifyStrictNegativeInvocationWithNoMatchingExpectation(t *testing.T) {
	mock := newTestMock()

	mock.foo()
	mock.SetupComplete()

	mock.foo()
	mock.bar("test", map[string]string{})

	tt := &testing.T{}
	// invocation with no matching expectation should result in verifyStrict failure
	mock.VerifyStrict(tt)

	assert.True(t, tt.Failed())
}

// fooBarInterface is an example interface used for mock testing
type fooBarInterface interface {
	foo()
	bar(s string, m map[string]string)
	foobar(mc monitoring.Secrets) string
}

// fooBarMock is a mock for testing the fooBar interface
type fooBarMock struct {
	fooBarInterface
	Mock
}

func (t *fooBarMock) foo() {
	t.Record("foo", nil)
}

func newTestMock() *fooBarMock {
	return &fooBarMock{
		Mock: *NewMock(),
	}
}

func (t *fooBarMock) bar(s string, m map[string]string) {
	t.Record("bar", map[string]interface{}{"s": s, "m": m})
}

func (t *fooBarMock) foobar(secrets monitoring.Secrets) string {
	t.Record("foobar", map[string]interface{}{"secrets": secrets})

	return "foobar"
}

// AnySecrets is used to match any Secrets in mock testing
type AnySecrets struct {
	monitoring.Secrets
	Any
}
