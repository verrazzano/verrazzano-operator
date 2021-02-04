// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano-operator/test/integ/k8s"
	"time"
)

const filebeat = "filebeat"
const journalbeat = "journalbeat"
const logging = "logging"
const monitoring = "monitoring"
const nodeExporter = "node-exporter"
const promPusherSystem = "prom-pusher-system"
const system = "system"
const verrazzano = "verrazzano"
const verrazzanoOperator = "verrazzano-operator"
const verrazzanoSystem = "verrazzano-system"

var fewSeconds = 2 * time.Second
var tenSeconds = 10 * time.Second
var thirtySeconds = 30 * time.Second
var oneMinute = 60 * time.Second
var K8sClient k8s.Client

var _ = BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient()
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}
})

var _ = AfterSuite(func() {
})

var _ = Describe("Verrazzano cluster roles for verrazzano operator", func() {
	It("is deployed", func() {
		Expect(K8sClient.DoesClusterRoleExist(verrazzanoOperator)).To(BeTrue(),
			"The verrazzano-operator cluster role should exist")
	})
})

var _ = Describe("Verrazzano cluster roles binding for verrazzano operator", func() {
	It("is deployed", func() {
		Expect(K8sClient.DoesClusterRoleBindingExist(verrazzanoOperator)).To(BeTrue(),
			"The verrazzano-operator cluster role binding should exist")
	})
})

var _ = Describe("verrazzano-system namespace resources ", func() {
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
			"The verrazzano operator doesn't exist")
	})
	It(fmt.Sprintf("Pod prefixed by %s exists", verrazzanoOperator), func() {
		Expect(K8sClient.DoesPodExist(verrazzanoOperator, verrazzanoSystem)).To(BeTrue(),
			"The verrazzano operator pod doesn't exist")
	})
	It("VMI should exist ", func() {
		Eventually(vmiExists, tenSeconds).Should(BeTrue())
	})
})

var _ = Describe("Logging namespace resources ", func() {
	It("logging namespace should exist ", func() {
		Eventually(loggingNamespaceExists, oneMinute).Should(BeTrue())
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
})

var _ = Describe("Monitoring namespace resources ", func() {
	It("monitoring namespace should exist ", func() {
		Eventually(monitoringNamespaceExists, oneMinute).Should(BeTrue())
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
