// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testutil

import (
	"reflect"
	"testing"
)

// MockInterface defines Mock functionality.
type MockInterface interface {
	// SetupComplete sets the mock as being done expectation mode and now in record mode.
	// This needs to be called after setting expectations and before invoking the mock in the test.
	SetupComplete()

	// GetIsSetup returns true if in record mode and false if in expectation mode.
	GetIsSetup() bool

	// Record records the method invocation. This should be called once from every mock method that will be tracked.
	Record(funcName string, args map[string]interface{})

	// Verify verifies that the actual recorded invocations 'nicely' match the expectations
	// This verification does a non-strict verification which is defined by the following:
	// - if no expectation is set for the method, it will not fail if the method is invoked any number of times
	// - for every method expectation, there must be at least one recorded invocation
	// - if there are more method expectations than there are recorded invocations, verify will fail
	// - there may be more recorded method invocations than there are method expectations
	// - if there are more invocations than expectations, the last expectation is used for arg matching
	// - argument matching does a deep equal
	// - to match 'any' argument of a certain type, create a struct of the interface type and add testutil.Any
	// - all verification failures are logged in the case of a verify failure
	Verify(*testing.T)

	// VerifyStrict verifies that the actual recorded invocations 'strictly' match the expectations
	// This verification does a 'strict' verification which is defined by the following:
	// - there must be a method invocation for every method expectation
	// - if no expectations are set for the method, verifyStrict will fail if the method is invoked
	// - if the number of method invocations doesn't equal the number of method expectations, verifyStrict will fail
	// - argument matching does a deep equal
	// - to match 'any' argument of a certain type, create a struct of the interface type and add testutil.Any
	// - all verification failures are logged in the case of a verify failure
	VerifyStrict(*testing.T)
}

// Mock implementation which implements the MockInterface.
type Mock struct {
	MockInterface
	records map[string]*methodRecords
	isSetup bool
}

// SetupComplete sets the mock as being done expectation mode and now in record mode.
// This needs to be called after setting expectations and before invoking the mock in the test.
func (m *Mock) SetupComplete() {
	m.isSetup = true
}

// GetIsSetup returns true if in record mode and false if in expectation mode.
func (m *Mock) GetIsSetup() bool {
	return m.isSetup
}

// Record records the method invocation. This should be called once from every mock method that will be tracked.
func (m *Mock) Record(funcName string, args map[string]interface{}) {
	methodRecords, found := m.records[funcName]
	if !found {
		m.records[funcName] = newMethodRecords()
		methodRecords = m.records[funcName]
	}
	methodRecords.add(args, m.GetIsSetup())
}

// Verify verifies that the actual recorded invocations 'nicely' match the expectations
// This verification does a non-strict verification which is defined by the following:
// - ordering of invocations across different functions isn't considered
// - if no expectation is set for the method, it will not fail if the method is invoked any number of times
// - for every method expectation, there must be at least one recorded invocation
// - if there are more method expectations than there are recorded invocations, verify will fail
// - there may be more recorded method invocations than there are method expectations
// - if there are more invocations than expectations, the last expectation is used for arg matching
// - argument matching does a deep equal
// - to match 'any' argument of a certain type, create a struct of the interface type and add testutil.Any
// - all verification failures are logged in the case of a verify failure
func (m *Mock) Verify(t *testing.T) {
	for funcName, mRecords := range m.records {
		expectationCount := len(mRecords.expectedCalls)
		recordedCount := len(mRecords.recordedCalls)

		if recordedCount < expectationCount {
			t.Errorf("Non-strict (Nice) mock expected call '%s' at least %d times but was actually called %d times",
				funcName, expectationCount, recordedCount)
		} else {
			for i := 0; i < recordedCount; i++ {
				expectedArgs := map[string]interface{}{}
				if expectationCount != 0 {
					if i >= expectationCount {
						expectedArgs = mRecords.expectedCalls[expectationCount-1]
					} else {
						expectedArgs = mRecords.expectedCalls[i]
					}
				}
				m.compareArgs(t, funcName, expectedArgs, mRecords.recordedCalls[i])
			}
		}
	}
	m.reset()
}

// VerifyStrict verifies that the actual recorded invocations 'strictly' match the expectations
// This verification does a 'strict' verification which is defined by the following:
// - ordering of invocations across different functions isn't considered
// - there must be a method invocation for every method expectation
// - if no expectations are set for the method, verifyStrict will fail if the method is invoked
// - if the number of method invocations doesn't equal the number of method expectations, verifyStrict will fail
// - argument matching does a deep equal
// - to match 'any' argument of a certain type, create a struct of the interface type and add testutil.Any
// - all verification failures are logged in the case of a verify failure
func (m *Mock) VerifyStrict(t *testing.T) {
	for funcName, mRecords := range m.records {
		if len(mRecords.expectedCalls) != len(mRecords.recordedCalls) {
			t.Errorf("Strict mock expected call '%s' %d times but was actually called %d times",
				funcName, len(mRecords.expectedCalls), len(mRecords.recordedCalls))
		} else {
			for i := 0; i < len(mRecords.recordedCalls); i++ {
				m.compareArgs(t, funcName, mRecords.expectedCalls[i], mRecords.recordedCalls[i])
			}
		}
	}
	m.reset()
}

// reset resets the mock state
func (m *Mock) reset() {
	m.records = make(map[string]*methodRecords)
	m.isSetup = false
}

func (m *Mock) compareArgs(t *testing.T, funcName string, expectedArgs map[string]interface{}, recordedArgs map[string]interface{}) {
	for argName, expectedValue := range expectedArgs {
		recordedValue := recordedArgs[argName]
		switch expectedValue.(type) {
		case Any:
			if recordedValue == nil {
				t.Errorf("For the function call '%s', the value for arg '%s' doesn't match the expectation. Expected: %v but received %v",
					funcName, argName, expectedValue, recordedValue)
			}
		default:
			if !reflect.DeepEqual(expectedValue, recordedValue) {
				t.Errorf("For the function call '%s', the value for arg '%s' doesn't match the expectation. Expected: %v but received %v",
					funcName, argName, expectedValue, recordedValue)
			}
		}
	}
}

// NewMock returns a newly created Mock
func NewMock() *Mock {
	return &Mock{
		records: make(map[string]*methodRecords),
	}
}

// methodRecords is used to track the method expectations and invocations
type methodRecords struct {
	expectedCalls []map[string]interface{}
	recordedCalls []map[string]interface{}
}

// newMethodRecords returns a new methodRecords
func newMethodRecords() *methodRecords {
	return &methodRecords{recordedCalls: make([]map[string]interface{}, 0), expectedCalls: make([]map[string]interface{}, 0)}
}

// add adds a record for a function
func (m *methodRecords) add(args map[string]interface{}, isRecordedCall bool) {
	if isRecordedCall {
		m.recordedCalls = append(m.recordedCalls, args)
	} else {
		m.expectedCalls = append(m.expectedCalls, args)
	}
}

// Any is a 'marker' interface used to match any non-nil argument of the correct type
// To use, simple add the Any interface to a test struct which also includes the interface type of the ard
// For example, to match any FooInterface argument use the following struct in your expectation:
// type Foo struct {
//   FooInterface
//   Any
// }
type Any interface {
	any()
}
