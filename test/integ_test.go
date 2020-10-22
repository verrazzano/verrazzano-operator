// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const verrazzanoValidation = "verrazzano-validation"

// Randomly generated password used by test
var testPwd = generateRandomString()

var _ = BeforeSuite(func() {
	_, stderr := runCommand("kubectl apply -f testdata/local-cluster.yaml")
	Expect(stderr).To(Equal(""))
})

var _ = AfterSuite(func() {
	_, stderr := runCommand("kubectl delete -f testdata/local-cluster.yaml")
	Expect(stderr).To(Equal(""))
})

var _ = Describe("Verrazzano admission controller", func() {
	It("is deployed", func() {
		deployment, err := getClientSet().AppsV1().Deployments("verrazzano-system").Get(context.Background(), "verrazzano-admission-controller", metav1.GetOptions{})
		Expect(err).To(BeNil(), "Should not have received an error when trying to get the verrazzano-admission-controller deployment")
		Expect(deployment.Spec.Template.Spec.Containers[0].Name).To(Equal("webhook"),
			"Should have a container with the name webhook")
	})

	It("is running (within 1m)", func() {
		isPodRunningYet := func() bool {
			return isPodRunning("verrazzano-admission-controller", "verrazzano-system")
		}
		Eventually(isPodRunningYet, "1m", "5s").Should(BeTrue(),
			"The verrazzano-admission-controller pod should be in the Running state")
	})
})

var _ = Describe("Verrazzano secret for admission controller", func() {
	It("is deployed", func() {
		_, err := getClientSet().CoreV1().Secrets("verrazzano-system").Get(context.Background(), verrazzanoValidation, metav1.GetOptions{})
		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s secret", verrazzanoValidation))
	})
})

var _ = Describe("Verrazzano validatingWebhookConfiguration", func() {
	It("is deployed", func() {
		_, err := getClientSet().AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(context.Background(), verrazzanoValidation, metav1.GetOptions{})
		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s validatingWebhookConfiguration", verrazzanoValidation))
	})
})

var _ = Describe("Verrazzano service account for admission controller", func() {
	It("is deployed", func() {
		_, err := getClientSet().CoreV1().ServiceAccounts("verrazzano-system").Get(context.Background(), verrazzanoValidation, metav1.GetOptions{})
		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s service account", verrazzanoValidation))
	})
})

var _ = Describe("Verrazzano service for admission controller", func() {
	It("is deployed", func() {
		_, err := getClientSet().CoreV1().Services("verrazzano-system").Get(context.Background(), verrazzanoValidation, metav1.GetOptions{})
		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s service", verrazzanoValidation))
	})
})

var _ = Describe("Verrazzano cluster roles for admission controller", func() {
	It("is deployed", func() {
		_, err := getClientSet().RbacV1().ClusterRoles().Get(context.Background(), verrazzanoValidation, metav1.GetOptions{})
		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s cluster roles", verrazzanoValidation))
	})
})

var _ = Describe("Verrazzano cluster roles binding for admission controller", func() {
	It("is deployed", func() {
		_, err := getClientSet().RbacV1().ClusterRoleBindings().Get(context.Background(), verrazzanoValidation, metav1.GetOptions{})
		Expect(err).To(BeNil(), fmt.Sprintf("Should not have received an error when trying to get the %s cluster roles binding", verrazzanoValidation))
	})
})
var _ = Describe("Custom Resource Definition for clusters", func() {
	It("verrazzanomanagedclusters.verrazzano.io exists", func() {
		Expect(doesCRDExist("verrazzanomanagedclusters.verrazzano.io")).To(BeTrue(),
			"The verrazzanomanagedclusters.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Custom Resource Definition for models", func() {
	It("verrazzanomodels.verrazzano.io exists", func() {
		Expect(doesCRDExist("verrazzanomodels.verrazzano.io")).To(BeTrue(),
			"The verrazzanomodels.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Custom Resource Definition for bindings", func() {
	It("verrazzanobindings.verrazzano.io exists", func() {
		Expect(doesCRDExist("verrazzanobindings.verrazzano.io")).To(BeTrue(),
			"The verrazzanobindings.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Apply binding", func() {
	It("with invalid ingressBindings", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/invalid-ingress-binding.yaml")
		Expect(stderr).To(ContainSubstring(" Invalid DNS name"))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply binding", func() {
	It("with invalid components", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/invalid-components-binding.yaml")
		Expect(stderr).To(ContainSubstring(" Invalid Component"))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Delete model", func() {
	It("is not found", func() {
		_, stderr := runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(ContainSubstring("error when deleting \"testdata/min-model.yaml\": verrazzanomodels.verrazzano.io \"min-model\" not found"),
			"The model should not exist")
	})
})

var _ = Describe("Delete binding", func() {
	It("is not found", func() {
		_, stderr := runCommand("kubectl delete -f testdata/min-binding.yaml")
		Expect(stderr).To(ContainSubstring("error when deleting \"testdata/min-binding.yaml\": verrazzanobindings.verrazzano.io \"min-binding\" not found"),
			"The binding should not exist")
	})
})

var _ = Describe("Apply binding", func() {
	It("with no matching model", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-binding.yaml")
		Expect(stderr).To(ContainSubstring("binding is referencing model min-model that does not exist in namespace default"),
			"The matching model must exist")
	})
})

var _ = Describe("Delete model", func() {
	It("before delete of binding", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/min-binding.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(ContainSubstring("model cannot be deleted before binding min-binding is deleted in namespace default"))
		_, stderr = runCommand("kubectl delete -f testdata/min-binding.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply binding", func() {
	It("with no matching placement(s)", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/invalid-1placement-binding.yaml")
		Expect(stderr).To(ContainSubstring("binding references cluster(s) \"env-managed-1\" that do not exist in namespace default"))
		_, stderr = runCommand("kubectl apply -f testdata/invalid-2placements-binding.yaml")
		Expect(stderr).To(ContainSubstring("binding references cluster(s) \"env-managed-1,env-managed-2\" that do not exist in namespace default"))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply binding", func() {
	It("with a placement using the default namespace", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/default-namespace-in-placement-binding.yaml")
		Expect(stderr).To(ContainSubstring("default namespace is not allowed in placements of binding"))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply model", func() {
	It("with WebLogic domain containing more than cluster", func() {
		_, stderr := runCommand("kubectl apply -f testdata/domain-with-multiple-clusters-model.yaml")
		Expect(stderr).To(ContainSubstring("More than one WebLogic cluster is not allowed for WebLogic domain weblogic-domain"))
	})
})

var _ = Describe("Apply model", func() {
	It("with missing Helidon application imagePullSecret", func() {
		_, stderr := runCommand("kubectl apply -f testdata/missing-helidon-secret-model.yaml")
		Expect(stderr).To(ContainSubstring("model references helidonApplications.imagePullSecret \"ocr\" for component helidon-application.  This secret must be created in the default namespace before proceeding."))
	})
})

var _ = Describe("Apply model", func() {
	It("with missing Coherence imagePullSecret", func() {
		_, stderr := runCommand("kubectl apply -f testdata/missing-coherence-secret-model.yaml")
		Expect(stderr).To(ContainSubstring("model references coherenceClusters.imagePullSecret \"ocr\" for component coherence-application.  This secret must be created in the default namespace before proceeding."))
	})
})

var _ = Describe("Apply model", func() {
	It("with missing WebLogic imagePullSecret", func() {
		_, stderr := runCommand("kubectl apply -f testdata/missing-weblogic-secret-model.yaml")
		Expect(stderr).To(ContainSubstring("model references weblogicDomains.domainCRValues.imagePullSecret \"ocr\" for component weblogic-domain.  This secret must be created in the default namespace before proceeding."))
	})
})

var _ = Describe("Apply model", func() {
	It("with missing webLogicCredentialsSecret", func() {
		_, stderr := runCommand("kubectl create secret docker-registry ocr --docker-username=user-id --docker-password=" + testPwd + " --docker-server=container-registry.oracle.com")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/missing-weblogic-credentials-secret-model.yaml")
		Expect(stderr).To(ContainSubstring("model references weblogicDomains.domainCRValues.webLogicCredentialsSecret \"domain-credentials\" for component weblogic-domain.  This secret must be created in the default namespace before proceeding."))
		_, stderr = runCommand("kubectl delete secret ocr")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply model", func() {
	It("with missing configOverrideSecrets", func() {
		_, stderr := runCommand("kubectl create secret docker-registry ocr --docker-username=user-id --docker-password=" + testPwd + " --docker-server=container-registry.oracle.com")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl create secret generic domain-credentials --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/missing-config-override-secrets-model.yaml")
		Expect(stderr).To(ContainSubstring("model references weblogicDomains.domainCRValues.configOverrideSecrets \"config-secret\" for component weblogic-domain.  This secret must be created in the default namespace before proceeding."))
		_, stderr = runCommand("kubectl delete secret domain-credentials")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete secret ocr")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply binding", func() {
	It("with missing databaseBinding credentials", func() {
		_, stderr := runCommand("kubectl create secret generic found-credentials --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/missing-database-credentials-secret-binding.yaml")
		Expect(stderr).To(ContainSubstring("binding references databaseBindings.credentials \"missing-credentials\" for mysql2.  This secret must be created in the default namespace before proceeding."))
		_, stderr = runCommand("kubectl delete secret found-credentials")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply binding", func() {
	It("with very long binding name", func() {
		_, stderr := runCommand("kubectl apply -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/very-long-name-binding.yaml")
		Expect(stderr).To(ContainSubstring("the VMI domain name is greater than 64 characters: *.vmi.very-long-name-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-binding"))
		_, stderr = runCommand("kubectl delete -f testdata/min-model.yaml")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Apply model", func() {
	It("with invalid environmentVariableForHost in rest connection", func() {
		_, stderr := runCommand("kubectl apply -f testdata/invalid-conn-rest-env-vars-coherence-model.yaml")
		Expect(stderr).To(ContainSubstring("a valid environment variable name must consist of alphabetic characters"))
	})
})

var _ = Describe("Apply model", func() {
	It("with invalid environmentVariableForPort in rest connection", func() {
		_, stderr := runCommand("kubectl create secret docker-registry ocr --docker-username=user-id --docker-password=" + testPwd + " --docker-server=container-registry.oracle.com")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl create secret generic domain-credentials --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/invalid-conn-rest-env-vars-weblogic-model.yaml")
		Expect(stderr).To(ContainSubstring("a valid environment variable name must consist of alphabetic characters"))
		_, stderr = runCommand("kubectl delete secret domain-credentials")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete secret ocr")
		Expect(stderr).To(Equal(""))

	})
})

var _ = Describe("Apply model", func() {
	It("with invalid environmentVariableForHost and environmentVariableForPort in rest connection", func() {
		_, stderr := runCommand("kubectl apply -f testdata/invalid-conn-rest-env-vars-helidon-model.yaml")
		Expect(stderr).To(ContainSubstring("uses the same environment variable for host and port"))
	})
})

var _ = Describe("Apply model", func() {
	It("Helidon with invalid ports", func() {
		_, stderr := runCommand("kubectl apply -f testdata/invalid-ports-helidon-model.yaml")
		Expect(stderr).To(ContainSubstring("cannot unmarshal number -1 into Go struct field VerrazzanoHelidon.spec.helidonApplications.targetPort of type uint"))
	})
	It("WebLogic with invalid ports", func() {
		_, stderr := runCommand("kubectl create secret generic domain-credentials --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/invalid-ports-weblogic-model.yaml")
		Expect(stderr).To(ContainSubstring("Port -1 is not valid. must be between 1 and 65535, inclusive"))
		Expect(stderr).To(ContainSubstring("AdminPort and T3Port in WebLogic domain weblogic-domain have the same value"))
		_, stderr = runCommand("kubectl delete secret domain-credentials")
		Expect(stderr).To(Equal(""))
	})
	It("Helidon with non-default ports", func() {
		_, stderr := runCommand("kubectl apply -f testdata/non-default-ports-helidon-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/non-default-ports-helidon-model.yaml")
		Expect(stderr).To(Equal(""))
	})
	It("WebLogic with non-default ports", func() {
		_, stderr := runCommand("kubectl create secret generic domain-credentials --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/non-default-ports-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/non-default-ports-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete secret domain-credentials")
		Expect(stderr).To(Equal(""))
	})
	It("Helidon with default ports", func() {
		_, stderr := runCommand("kubectl apply -f testdata/default-ports-helidon-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/default-ports-helidon-model.yaml")
		Expect(stderr).To(Equal(""))
	})
	It("WebLogic with default ports", func() {
		_, stderr := runCommand("kubectl create secret generic domain-credentials --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/default-ports-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/default-ports-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete secret domain-credentials")
		Expect(stderr).To(Equal(""))
	})
	It("WebLogic with missing clusters", func() {
		_, stderr := runCommand("kubectl create secret generic domain-credentials-missing-clusters --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/missing-clusters-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/missing-clusters-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete secret domain-credentials-missing-clusters")
		Expect(stderr).To(Equal(""))
	})
	It("WebLogic with empty array clusters", func() {
		_, stderr := runCommand("kubectl create secret generic domain-credentials-empty-array-clusters --from-literal=username=user-id --from-literal=password=" + testPwd)
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/empty-array-clusters-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/empty-array-clusters-weblogic-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete secret domain-credentials-empty-array-clusters")
		Expect(stderr).To(Equal(""))
	})
})

var _ = Describe("Namespace", func() {
	It("is created", func() {
		stdout, stderr := runCommand("kubectl create namespace foo")
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal("namespace/foo created\n"))
	})

	It("is deleted", func() {
		stdout, stderr := runCommand("kubectl delete namespace foo")
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal("namespace \"foo\" deleted\n"))
	})

})

var _ = Describe("Apply model with GenericComponents", func() {
	It("with invalid initContainer port", func() {
		_, stderr := runCommand("kubectl apply -f testdata/generic-components-model-invalidPort.yaml")
		Expect(stderr).To(ContainSubstring("Port -1 is not valid. must be between 1 and 65535"))
	})
	It("with missing initContainer env reference", func() {
		createSecret("ocr")
		_, stderr := runCommand("kubectl apply -f testdata/generic-components-model.yaml")
		Expect(stderr).To(ContainSubstring("model references genericComponents.Deployment.InitContainers.Env \"init-credentials\""))
		deleteSecret("ocr")
	})
	It("with missing container env reference", func() {
		createSecret("ocr")
		createSecret("init-credentials")
		_, stderr := runCommand("kubectl apply -f testdata/generic-components-model.yaml")
		Expect(stderr).To(ContainSubstring("model references genericComponents.Deployment.Containers.Env \"mysql-credentials\""))
		deleteSecret("ocr")
		deleteSecret("init-credentials")
	})
	It("with all secrets", func() {
		createSecret("ocr")
		createSecret("init-credentials")
		createSecret("mysql-credentials")
		_, stderr := runCommand("kubectl apply -f testdata/generic-components-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/generic-components-binding.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/generic-components-binding.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl delete -f testdata/generic-components-model.yaml")
		Expect(stderr).To(Equal(""))
		deleteSecret("mysql-credentials")
		deleteSecret("init-credentials")
		deleteSecret("ocr")
	})
	It("with invalid binding", func() {
		createSecret("ocr")
		createSecret("init-credentials")
		createSecret("mysql-credentials")
		_, stderr := runCommand("kubectl apply -f testdata/generic-components-model.yaml")
		Expect(stderr).To(Equal(""))
		_, stderr = runCommand("kubectl apply -f testdata/generic-components-binding-invalid.yaml")
		Expect(stderr).To(ContainSubstring("Multiple occurrence of component across placement namespaces. Invalid Component: [mysql]"))
		_, stderr = runCommand("kubectl delete -f testdata/generic-components-model.yaml")
		Expect(stderr).To(Equal(""))
		deleteSecret("mysql-credentials")
		deleteSecret("init-credentials")
		deleteSecret("ocr")
	})
})

func createSecret(name string) string {
	cmd := fmt.Sprintf("kubectl create secret generic %s --from-literal=username=%s --from-literal=password=%s", name, name, name)
	_, stderr := runCommand(cmd)
	return stderr
}
func deleteSecret(name string) string {
	cmd := fmt.Sprintf("kubectl delete secret %s", name)
	_, stderr := runCommand(cmd)
	return stderr
}

// ---------------------------   helper functions ------------------------------------

func getKubeconfig() string {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		Fail("Could not get kubeconfig")
	}
	return kubeconfig
}

func getClientSet() *kubernetes.Clientset {
	kubeconfig := getKubeconfig()

	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		Fail("Could not get current context from kubeconfig " + kubeconfig)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		Fail("Could not get clientset from config")
	}

	return clientset
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}

func isPodRunning(name string, namespace string) bool {
	GinkgoWriter.Write([]byte("[DEBUG] checking if there is a running pod named " + name + "* in namespace " + namespace + "\n"))
	clientset := getClientSet()
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		Fail("Could not get list of pods")
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			conditions := pods.Items[i].Status.Conditions
			for j := range conditions {
				if conditions[j].Type == "Ready" {
					if conditions[j].Status == "True" {
						return true
					}
				}
			}
		}
	}
	return false
}

func doesCRDExist(crdName string) bool {
	kubeconfig := getKubeconfig()

	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		Fail("Could not get current context from kubeconfig " + kubeconfig)
	}

	apixClient, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		Fail("Could not get apix client")
	}

	crds, err := apixClient.CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		Fail("Failed to get list of CustomResourceDefinitions")
	}

	for i := range crds.Items {
		if strings.Compare(crds.Items[i].ObjectMeta.Name, crdName) == 0 {
			return true
		}
	}

	return false
}

// RunCommand runs an external process, captures the stdout
// and stderr, as well as streaming them to the real output
// streams in real time
func runCommand(commandLine string) (string, string) {
	GinkgoWriter.Write([]byte("[DEBUG] RunCommand: " + commandLine + "\n"))
	parts := strings.Split(commandLine, " ")
	var cmd *exec.Cmd
	if len(parts) < 1 {
		Fail("No command provided")
	} else if len(parts) == 1 {
		cmd = exec.Command(parts[0], "")
	} else {
		cmd = exec.Command(parts[0], parts[1:]...)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		Fail("cmd.Start() failed with " + err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	_, errStderr = io.Copy(stderr, stderrIn)
	wg.Wait()

	cmd.Wait()
	if errStdout != nil || errStderr != nil {
		Fail("failed to capture stdout or stderr")
	}
	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	return outStr, errStr
}

// generateRandomString returns a base64 encoded generated random string.
func generateRandomString() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
