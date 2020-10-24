// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano-operator/test/integ/k8s"
	"github.com/verrazzano/verrazzano-operator/test/integ/util"
	"time"
)

const genapp = "genapp"
const verrazzano = "verrazzano"
const verrazzanoOperator = "verrazzano-operator"
const verrazzanoSystem = "verrazzano-system"

var fewSeconds = 2 * time.Second
var tenSeconds = 10 * time.Second
var oneMinute = 60 * time.Second
var K8sClient k8s.K8sClient

var _ = BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewK8sClient()
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}
})

var _ = AfterSuite(func() {
})

var _ = Describe("Verrazzano cluster roles for verrazzano operator", func() {
	It("is deployed", func() {
		Expect(K8sClient.DoesClusterRoleExist(verrazzanoOperator)).To(BeTrue(),
			"The verrazzano-operator cluster rol should exist")
	})
})

var _ = Describe("Verrazzano cluster roles binding for verrazzano operator", func() {
	It("is deployed", func() {
		Expect(K8sClient.DoesClusterRoleBindingExist(verrazzanoOperator)).To(BeTrue(),
			"The verrazzano-operator cluster role binding should exist")
	})
})

var _ = Describe("Custom Resource Definition for clusters", func() {
	It("verrazzanomanagedclusters.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanomanagedclusters.verrazzano.io")).To(BeTrue(),
			"The verrazzanomanagedclusters.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Custom Resource Definition for models", func() {
	It("verrazzanomodels.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanomodels.verrazzano.io")).To(BeTrue(),
			"The verrazzanomodels.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Custom Resource Definition for bindings", func() {
	It("verrazzanobindings.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanobindings.verrazzano.io")).To(BeTrue(),
			"The verrazzanobindings.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Secret for verrazzano operator", func() {
	It("exists", func() {
		Expect(K8sClient.DoesSecretExist(verrazzano, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator secret should exist")
	})
})

var _ = Describe("Service account for verrazzano operator", func() {
	It("exists", func() {
		Expect(K8sClient.DoesServiceAccountExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator service account should exist")
	})
})

var _ = Describe("Service for verrazzano operator", func() {
	It("exists", func() {
		Expect(K8sClient.DoesServiceExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator service should exist")
	})
})

var _ = Describe("Deployment in verrazzano-system", func() {
	It("verrazzano-operator deployment should exist", func() {
		Expect(K8sClient.DoesDeployementExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator doesn't doesn't exist")
	})
})

var _ = Describe("Pod in verrazzano-system", func() {
	It("verrazzano-operator pod should exist", func() {
		Expect(K8sClient.DoesPodExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator pod doesn't exist")
	})
})

var _ = Describe("Testing generic app model/binding lifecycle", func() {
	It("apply model should result in a vm in default namespace", func() {
		_, stderr := util.RunCommand("kubectl apply -f testdata/gen-model.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(genAppModelExists, fewSeconds).Should(BeTrue())
	})
	It("apply binding should result in a vb in default namespace", func() {
		_, stderr := util.RunCommand("kubectl apply -f testdata/gen-binding.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(genAppBindingExists, fewSeconds).Should(BeTrue())
	})
	It("genapp namespace should exist", func() {
		Eventually(genAppNsExists, tenSeconds).Should(BeTrue())
	})
	It("genapp VMI should exist ", func() {
		Eventually(genAppVmiExists, tenSeconds).Should(BeTrue())
	})
	It("genapp deployment should exist ", func() {
		Eventually(genAppDeploymentExists(), tenSeconds).Should(BeTrue())
	})
	It("genapp pod should exist ", func() {
		Eventually(genAppPodExists(), oneMinute).Should(BeTrue())
	})
	It("deleting generic application binding", func() {
		_, stderr := util.RunCommand("kubectl delete -f testdata/gen-binding.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(genAppBindingExists, fewSeconds).Should(BeFalse())
	})
	It("deleting generic application model", func() {
		_, stderr := util.RunCommand("kubectl delete -f testdata/gen-model.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(genAppModelExists, fewSeconds).Should(BeFalse())
	})
})

// Helper functions
func genAppModelExists() bool {
	return K8sClient.DoesModelExist(genapp)
}
func genAppBindingExists() bool {
	return K8sClient.DoesBindingExist(genapp)
}
func genAppNsExists() bool {
	return K8sClient.DoesNamespaceExist(genapp)
}
func genAppVmiExists() bool {
	return K8sClient.DoesVmiExist(genapp)
}
func genAppDeploymentExists() bool {
	return K8sClient.DoesDeployementExist(genapp,genapp)
}
func genAppPodExists() bool {
	return K8sClient.DoesPodExist(genapp,genapp)
}
