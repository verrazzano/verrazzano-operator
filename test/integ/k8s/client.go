// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	vzclient "github.com/verrazzano/verrazzano-crd-generator/pkg/client/clientset/versioned"
	vmiclient "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"

	"github.com/verrazzano/verrazzano-operator/test/integ/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	// Client to access the Kubernetes API
	clientset *kubernetes.Clientset

	// Client to access the Kubernetes API for extensions
	apixClient *apixv1beta1client.ApiextensionsV1beta1Client

	// Client to access Kubernetes API for Verrazzano objects
	vzClient *vzclient.Clientset

	// Client to access Kubernetes API for Verrazzano objects
	vmiClient *vmiclient.Clientset
}

// Get a new client that calls the Kubernetes API server to access the Verrazzano API Objects
func NewK8sClient() (K8sClient, error) {
	kubeconfig := util.GetKubeconfig()

	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return K8sClient{}, err
	}

	// Client to access the Kubernetes API
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return K8sClient{}, err
	}

	// Client to access the Kubernetes API for extensions
	apixcli, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		return K8sClient{}, err
	}

	// Client to access Kubernetes API for Verrazzano objects
	vzcli, err := vzclient.NewForConfig(config)
	if err != nil {
		return K8sClient{}, err
	}

	// Client to access Kubernetes API for Verrazzano VMI objects
	vmicli, err := vmiclient.NewForConfig(config)
	if err != nil {
		return K8sClient{}, err
	}

	return K8sClient{
		clientset: clientset,
		apixClient: apixcli,
		vmiClient: vmicli,
		vzClient: vzcli,
	}, err
}

