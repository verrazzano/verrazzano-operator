// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

type MockDeployment struct {
	namespace string
	deleted   string
}

func (mock *MockDeployment) DeleteDeployment(namespace, name string) error {
	mock.namespace = namespace
	mock.deleted = name
	return nil
}

func TestCreateDeployment(t *testing.T) {
	mock := &MockDeployment{}
	app := "hello"
	pomPusher := pomPusherName(app)
	DeletePomPusher(app, mock)
	assert.Equal(t, constants.MonitoringNamespace, mock.namespace, "namespace")
	assert.Equal(t, pomPusher, mock.deleted, "deployment deleted")
}

func TestGetSystemDeployments(t *testing.T) {
	mock := &MockDeployment{}
	app := "hello"
	pomPusher := pomPusherName(app)
	DeletePomPusher(app, mock)
	assert.Equal(t, constants.MonitoringNamespace, mock.namespace, "namespace")
	assert.Equal(t, pomPusher, mock.deleted, "deployment deleted")
}
