// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"flag"
	"io/ioutil"
	"math/rand"
	"time"

	clientset "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Global framework.
var Global *Framework

// Framework handles communication with the kube cluster in e2e tests.
type Framework struct {
	KubeConfigContents       string
	KubeClient               kubernetes.Interface
	VerrazzanoOperatorClient clientset.Interface
	Namespace                string
	RunID                    string
}

// Setup sets up a test framework and initialises framework.Global.
func Setup() error {
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file with authorization and master location information.")
	namespace := flag.String("namespace", "default", "Integration test namespace")
	runid := flag.String("runid", "", "Optional string that will be used to uniquely identify this test run.")
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	verrazzanoOperatorClientSet, err := clientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(*kubeconfig)
	if err != nil {
		return err
	}

	if *runid == "" {
		runIdString := "test-" + generateRandomID(3)
		runid = &runIdString
	}

	Global = &Framework{
		KubeConfigContents:       string(data),
		KubeClient:               kubeClient,
		VerrazzanoOperatorClient: verrazzanoOperatorClientSet,
		Namespace:                *namespace,
		RunID:                    *runid,
	}

	return nil
}

// Teardown shuts down the test framework and cleans up.
func Teardown() error {
	Global = nil
	return nil
}

func generateRandomID(n int) string {
	rand.Seed(time.Now().Unix())
	var letter = []rune("abcdefghijklmnopqrstuvwxyz")

	id := make([]rune, n)
	for i := range id {
		id[i] = letter[rand.Intn(len(letter))]
	}
	return string(id)
}
