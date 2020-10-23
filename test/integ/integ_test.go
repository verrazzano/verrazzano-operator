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

const verrazzano = "verrazzano"
const verrazzanoOperator = "verrazzano-operator"

var vzK8sClient k8s.VerrazzanoK8sClient

var _ = BeforeSuite(func() {
	var err error
	vzK8sClient, err = k8s.NewVzK8sClient()
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}
})

var _ = AfterSuite(func() {

})

//var _ = Describe("Verrazzano Operator", func() {
//	It("is deployed", func() {
//		deployment, err := k8s.GetClientSet().AppsV1().Deployments("verrazzano-system").Get(context.Background(), "verrazzano-operator", metav1.GetOptions{})
//		Expect(err).To(BeNil(), "Should not have received an error when trying to get the verrazzano-operator deployment")
//		Expect(deployment.Spec.Template.Spec.Containers[0].Name).To(Equal("verrazzano-operator"),
//			"Should have a container with the name verrazzano-operator")
//	})
//
//	It("is running (within 1m)", func() {
//		isPodRunningYet := func() bool {
//			return k8s.IsPodRunning("verrazzano-operator", "verrazzano-system")
//		}
//		Eventually(isPodRunningYet, "1m", "5s").Should(BeTrue(),
//			"The verrazzano-operator pod should be in the Running state")
//	})
//})
//
//var _ = Describe("Verrazzano secret for verrazzano operator", func() {
//	It("is deployed", func() {
//		_, err := k8s.GetClientSet().CoreV1().Secrets("verrazzano-system").Get(context.Background(), verrazzano, metav1.GetOptions{})
//		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s secret", verrazzano))
//	})
//})
//
//var _ = Describe("Verrazzano service account for verrazzano operator", func() {
//	It("is deployed", func() {
//		_, err := k8s.GetClientSet().CoreV1().ServiceAccounts("verrazzano-system").Get(context.Background(), verrazzanoOperator, metav1.GetOptions{})
//		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s service account", verrazzanoOperator))
//	})
//})
//
//var _ = Describe("Verrazzano service for verrazzano operator", func() {
//	It("is deployed", func() {
//		_, err := k8s.GetClientSet().CoreV1().Services("verrazzano-system").Get(context.Background(), verrazzanoOperator, metav1.GetOptions{})
//		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s service", verrazzanoOperator))
//	})
//})
//
//var _ = Describe("Verrazzano cluster roles for verrazzano operator", func() {
//	It("is deployed", func() {
//		_, err := k8s.GetClientSet().RbacV1().ClusterRoles().Get(context.Background(), verrazzanoOperator, metav1.GetOptions{})
//		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s cluster roles", verrazzanoOperator))
//	})
//})
//
//var _ = Describe("Verrazzano cluster roles binding for verrazzano operator", func() {
//	It("is deployed", func() {
//		_, err := k8s.GetClientSet().RbacV1().ClusterRoleBindings().Get(context.Background(), verrazzanoOperator, metav1.GetOptions{})
//		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s cluster roles binding", verrazzanoOperator))
//	})
//})
//var _ = Describe("Custom Resource Definition for clusters", func() {
//	It("verrazzanomanagedclusters.verrazzano.io exists", func() {
//		Expect(k8s.DoesCRDExist("verrazzanomanagedclusters.verrazzano.io")).To(BeTrue(),
//			"The verrazzanomanagedclusters.verrazzano.io CRD should exist")
//	})
//})
//
//var _ = Describe("Custom Resource Definition for models", func() {
//	It("verrazzanomodels.verrazzano.io exists", func() {
//		Expect(k8s.DoesCRDExist("verrazzanomodels.verrazzano.io")).To(BeTrue(),
//			"The verrazzanomodels.verrazzano.io CRD should exist")
//	})
//})
//
//var _ = Describe("Custom Resource Definition for bindings", func() {
//	It("verrazzanobindings.verrazzano.io exists", func() {
//		Expect(k8s.DoesCRDExist("verrazzanobindings.verrazzano.io")).To(BeTrue(),
//			"The verrazzanobindings.verrazzano.io CRD should exist")
//	})
//})

var fewSeconds = 2 * time.Second

var _ = Describe("Testing generic app model/binding lifecycle", func() {
	It("applying generic application model", func() {
		_, stderr := util.RunCommand("kubectl apply -f testdata/gen-model.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(vzK8sClient.DoesModelExist("genapp"),fewSeconds).Should(BeTrue())
	})
	It("applying generic application binding", func() {
		_, stderr := util.RunCommand("kubectl apply -f testdata/gen-binding.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(vzK8sClient.DoesBindingExist("genapp"),fewSeconds).Should(BeTrue())
	})
	It("deleting generic application binding", func() {
		_, stderr := util.RunCommand("kubectl delete -f testdata/gen-binding.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(vzK8sClient.DoesBindingExist("genapp"),fewSeconds).Should(BeFalse())
	})
	It("deleting generic application model", func() {
		_, stderr := util.RunCommand("kubectl delete -f testdata/gen-model.yaml")
		Expect(stderr).To(Equal(""))
		Eventually(vzK8sClient.DoesModelExist("genapp"),fewSeconds).Should(BeFalse())
	})
})
