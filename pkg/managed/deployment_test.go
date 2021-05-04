// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/monitoring"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

var origGetEnvFunc = util.GetEnvFunc

// TestCreateDeploymentsVmiSystem tests the creation of deployments when the binding name is 'system'.
// GIVEN a cluster which does not have any deployments
//  WHEN I call CreateDeployments with a binding named 'system'
//  THEN there should be an expected set of system deployments created
func TestCreateDeploymentsVmiSystem(t *testing.T) {
	assert := assert.New(t)

	SyntheticModelBinding := testutil.GetSyntheticModelBinding()
	clusterConnections := testutil.GetManagedClusterConnections()
	vmiSecret := monitoring.NewVmiSecret(SyntheticModelBinding.SynBinding)
	secrets := &testutil.FakeSecrets{Secrets: map[string]*corev1.Secret{
		constants.VmiSecretName: vmiSecret,
	}}

	// temporarily set util.GetEnvFunc to mock response
	util.GetEnvFunc = getenv
	defer func() { util.GetEnvFunc = origGetEnvFunc }()

	SyntheticModelBinding.SynBinding.Name = constants.VmiSystemBindingName
	err := CreateDeployments(SyntheticModelBinding, clusterConnections, "testURI", secrets)
	assert.Nil(err, "got an error from CreateDeployments: %v", err)
}

// getenv returns a mocked response for keys used by these tests
func getenv(key string) string {
	return origGetEnvFunc(key)
}
