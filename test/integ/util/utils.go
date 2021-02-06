// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// ReadManagedCluster reads/unmarshal's ManagedCluster yaml file into a ManagedCluster.
func ReadManagedCluster(path string) (*types.ManagedCluster, error) {
	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var vmc types.ManagedCluster
	err = yaml.Unmarshal(yamlFile, &vmc)
	return &vmc, err
}

// WriteYmal writes/marshalls the obj to a yaml file.
func WriteYmal(path string, obj interface{}) (string, error) {
	fileout, _ := filepath.Abs(path)
	bytes, err := ToYmal(obj)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(fileout, bytes, 0644)
	return fileout, err
}

// ToYmal marshalls the obj to a yaml
func ToYmal(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}
