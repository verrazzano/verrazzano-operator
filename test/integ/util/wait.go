// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// DefaultRetry is the default backoff for e2e tests.
var DefaultRetry = wait.Backoff{
	Steps:    150,
	Duration: 4 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}

// Retry executes the provided function repeatedly, retrying until the function
// returns done = true, errors, or exceeds the given timeout.
func Retry(backoff wait.Backoff, fn wait.ConditionFunc) error {
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		done, err := fn()
		if err != nil {
			lastErr = err
		}
		return done, err
	})
	if err == wait.ErrWaitTimeout {
		if lastErr != nil {
			err = lastErr
		}
	}
	return err
}

// WaitForDeploymentAvailable waits for the given deployment to reach the given number of available replicas.
func WaitForDeploymentAvailable(namespace string, deploymentName string, availableReplicas int, kubeClient kubernetes.Interface) error {
	var err error
	fmt.Printf("Waiting for deployment '%s' to reach %d available and total replicas...\n", deploymentName, availableReplicas)
	err = Retry(DefaultRetry, func() (bool, error) {
		deployments, err := kubeClient.ExtensionsV1beta1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		for _, deployment := range deployments.Items {
			if deployment.Name == deploymentName {
				if deployment.Status.AvailableReplicas == int32(availableReplicas) && deployment.Status.Replicas == int32(availableReplicas) {
					return true, nil
				}
			}
		}
		return false, nil
	})
	return err
}

// WaitForServiceAccountAvailable waits for the given ServiceAccount to become available.
func WaitForServiceAccountAvailable(namespace string, serviceAccount string, kubeClient kubernetes.Interface) error {
	var err error
	fmt.Printf("Waiting for ServiceAccount '%s' to become available...\n", serviceAccount)
	err = Retry(DefaultRetry, func() (bool, error) {
		_, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), serviceAccount, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error getting ServiceAccount '%s'\n", err)
			return false, nil
		}
		return true, nil
	})
	return err
}
