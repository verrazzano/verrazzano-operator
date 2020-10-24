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

const filebeat = "filebeat"
const journalbeat = "journalbeat"
const genapp = "genapp"
const logging = "logging"
const monitoring = "monitoring"
const nodeExporter = "node-exporter"
const promPusherSystem = "prom-pusher-system"
const promPusherGenapp = "prom-pusher-genapp"
const system = "system"
const verrazzano = "verrazzano"
const verrazzanoOperator = "verrazzano-operator"
const verrazzanoSystem = "verrazzano-system"

var fewSeconds = 2 * time.Second
var tenSeconds = 10 * time.Second
var thirtySeconds = 30 * time.Second
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

var _ = Describe("Verrazzano namespace resources ", func() {
	It(fmt.Sprintf("Namespace %s exists", verrazzanoSystem), func() {
		Expect(K8sClient.DoesNamespaceExist(verrazzanoSystem)).To(BeTrue(),
			"The namespace should exist")
	})
	It(fmt.Sprintf("Secret %s exists", verrazzano), func() {
		Expect(K8sClient.DoesSecretExist(verrazzano, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator secret should exist")
	})
	It(fmt.Sprintf("ServiceAccount %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesServiceAccountExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator service account should exist")
	})
	It(fmt.Sprintf("Service %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesServiceExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator service should exist")
	})
	It(fmt.Sprintf("Deployment %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesDeploymentExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator doesn't doesn't exist")
	})
	It(fmt.Sprintf("Pod prefixed by %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesPodExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator pod doesn't exist")
	})
	It("VMI should exist ", func() {
		Eventually(vmiExists, tenSeconds).Should(BeTrue())
	})
	It("Logging namespace should exist ", func() {
		Eventually(loggingNamespaceExists, tenSeconds).Should(BeTrue())
	})
	It("Filebeat daemonset should exist ", func() {
		Eventually(filebeatDaemonsetExists, tenSeconds).Should(BeTrue())
	})
	It("Filebeat pod should exist ", func() {
		Eventually(filebeatPodExists, oneMinute).Should(BeTrue())
	})
	It("Jounalbeat daemonset should exist ", func() {
		Eventually(journalbeatDaemonsetExists, tenSeconds).Should(BeTrue())
	})
	It("Jounalbeat pod should exist ", func() {
		Eventually(journalbeatPodExists, oneMinute).Should(BeTrue())
	})
	It("Monitoring namespace should exist ", func() {
		Eventually(monitoringNamespaceExists, tenSeconds).Should(BeTrue())
	})
	It("Node exporter service should exist ", func() {
		Eventually(nodeExporterServiceExists, tenSeconds).Should(BeTrue())
	})
	It("Node exporter pod should exist ", func() {
		Eventually(nodeExporterPodExists, oneMinute).Should(BeTrue())
	})
	It("Node exporter daemonset should exist ", func() {
		Eventually(nodeExporterDaemonExists, tenSeconds).Should(BeTrue())
	})
	It("Prom pusher deployment should exist ", func() {
		Eventually(promPusherDeploymentExists, tenSeconds).Should(BeTrue())
	})
	It("Prom pusher pod should exist ", func() {
		Eventually(promPusherPodExists, oneMinute).Should(BeTrue())
	})
})

var _ = Describe("Logging namespace resources ", func() {
	It(logging+" exists", func() {
		Expect(K8sClient.DoesNamespaceExist(logging)).To(BeTrue(),
			"The namespace should exist")
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
		Eventually(genAppDeploymentExists, tenSeconds).Should(BeTrue())
	})
	It("genapp pod should exist ", func() {
		Eventually(genAppPodExists, oneMinute).Should(BeTrue())
	})
	It("genapp prom pusher deployment should exist ", func() {
		Eventually(genappPromPusherDeploymentExists, tenSeconds).Should(BeTrue())
	})
	It("genapp prom pusher pod should exist ", func() {
		Eventually(genappPromPusherPodExists, oneMinute).Should(BeTrue())
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
func loggingNamespaceExists() bool {
	return K8sClient.DoesNamespaceExist(logging)
}
func filebeatPodExists() bool {
	return K8sClient.DoesPodExist(filebeat, logging)
}
func filebeatDaemonsetExists() bool {
	return K8sClient.DoesDaemonsetExist(filebeat, logging)
}
func journalbeatPodExists() bool {
	return K8sClient.DoesPodExist(journalbeat, logging)
}
func journalbeatDaemonsetExists() bool {
	return K8sClient.DoesDaemonsetExist(journalbeat, logging)
}
func monitoringNamespaceExists() bool {
	return K8sClient.DoesNamespaceExist(monitoring)
}
func nodeExporterServiceExists() bool {
	return K8sClient.DoesServiceExist(nodeExporter, monitoring)
}
func nodeExporterPodExists() bool {
	return K8sClient.DoesPodExist(nodeExporter, monitoring)
}
func nodeExporterDaemonExists() bool {
	return K8sClient.DoesDaemonsetExist(nodeExporter, monitoring)
}
func promPusherDeploymentExists() bool {
	return K8sClient.DoesDeploymentExist(promPusherSystem, monitoring)
}
func promPusherPodExists() bool {
	return K8sClient.DoesPodExist(promPusherSystem, monitoring)
}
func vmiExists() bool {
	return K8sClient.DoesVmiExist(system)
}

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
	return K8sClient.DoesDeploymentExist(genapp, genapp)
}
func genAppPodExists() bool {
	return K8sClient.DoesPodExist(genapp, genapp)
}
func genappPromPusherDeploymentExists() bool {
	return K8sClient.DoesDeploymentExist(promPusherGenapp, monitoring)
}
func genappPromPusherPodExists() bool {
	return K8sClient.DoesPodExist(promPusherGenapp, monitoring)
}
