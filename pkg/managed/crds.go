// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation of CRDs into Managed Clusters

package managed

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"
)

// CreateCrdDefinitions creates/updates custom resource definitions needed for each managed cluster.
func CreateCrdDefinitions(managedClusterConnection *util.ManagedClusterConnection, managedCluster *v1beta1v8o.VerrazzanoManagedCluster, manifest *util.Manifest) error {
	// Create log instance for creating crd definitions
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "CrdDefinitions").Str("name", "Creation").Logger()

	logger.Debug().Msgf("Creating/updating CRDs for ManagedCluster %s", managedCluster.Name)
	managedClusterConnection.Lock.RLock()
	managedClusterConnection.Lock.RUnlock()

	newCrds, err := newCrds(manifest)
	if err != nil {
		return err
	}

	for _, newCrd := range newCrds {
		existingCrd, err := managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), newCrd.Name, metav1.GetOptions{})
		if err == nil {
			specDiffs := diff.CompareIgnoreTargetEmpties(existingCrd, newCrd)
			if specDiffs != "" {
				logger.Debug().Msgf("CRD %s : Spec differences %s", newCrd.Name, specDiffs)
				if len(newCrd.ResourceVersion) == 0 {
					newCrd.ResourceVersion = existingCrd.ResourceVersion
				}
				logger.Info().Msgf("Updating CRD %s", newCrd.Name)
				_, err = managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Update(context.TODO(), newCrd, metav1.UpdateOptions{})
			}
		} else {
			logger.Info().Msgf("Creating CRD %s", newCrd.Name)
			_, err = managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), newCrd, metav1.CreateOptions{})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// Constructs the necessary CRD Definitions for managed clusters
func newCrds(manifest *util.Manifest) ([]*v1.CustomResourceDefinition, error) {
	yamlFiles := []string{manifest.WlsMicroOperatorCrd, manifest.HelidonAppOperatorCrd, manifest.CohClusterOperatorCrd}
	var crds []*v1.CustomResourceDefinition

	// Loop through the CRD's that need to be generated
	for fileIndex := range yamlFiles {

		// Read the yaml file that defines the CRD
		yamlFile, err := ioutil.ReadFile(yamlFiles[fileIndex])
		if err != nil {
			return nil, err
		}

		// Create a CRD struct
		var crd v1.CustomResourceDefinition
		err = yaml.Unmarshal(yamlFile, &crd)
		if err != nil {
			return nil, err
		}

		crds = append(crds, &crd)
	}

	return crds, nil
}
