// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package managed

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8 "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	apicorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	testModelName                 = "my-model"
	testBindingname               = "my-binding"
	testClusterName               = "my-managed-1"
	testNamespace                 = "my-namespace"
	testComponentName             = "my-component"
	testComponentBindingName      = "my-app1"
	testKeepSourceLabels          = "__meta_kubernetes_pod_label_app"
	testKeepSourceLabelsRegex     = "my-component-label"
	testHelidonBindingName        = "my-helidon-binding"
	testCoherenceBindingName      = "my-coherence-binding"
	testWeblogicBindingName       = "my-weblogic-binding"
	testWeblogicCredentialsSecret = "my-weblogic-binding-weblogic-credentials"
	testWeblogicDomainUseranme    = "weblogic"
	testWeblogicDomainPassword    = "welcome1"
	testScrapeTargetJobName       = "my-job"
)

var testScrapeConfigInfo = ScrapeConfigInfo{
	PrometheusScrapeTargetJobName: testScrapeTargetJobName,
	Namespace:                     testNamespace,
	KeepSourceLabels:              testKeepSourceLabels,
	KeepSourceLabelsRegex:         testKeepSourceLabelsRegex,
	BindingName:                   testBindingname,
	ComponentBindingName:          testComponentBindingName,
	ManagedClusterName:            testClusterName,
	Username:                      testWeblogicDomainUseranme,
	Password:                      testWeblogicDomainPassword,
}

var testIstioCMPrometheusYml = map[string]string{PrometheusYml: `
global:
  scrape_interval: 15s
scrape_configs:
- job_name: dummy
`}

var testReplicas = int32(1)

// --------------
// | Unit Tests |
// --------------

func TestGetNewPrometheusConfigMap(t *testing.T) {
	confmap, err := getNewPrometheusConfigMap([]ScrapeConfigInfo{testScrapeConfigInfo}, testIstioCMPrometheusYml)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "prometheus", confmap.ObjectMeta.Name, "Wrong configmap name.")
	assert.Equal(t, "istio-system", confmap.ObjectMeta.Namespace, "Wrong configmap namespace.")
	validatePrometheusYml(t, confmap.Data[PrometheusYml])
}

func TestUpdatePrometheusYml(t *testing.T) {
	prometheusYml, err := updatePrometheusYml([]ScrapeConfigInfo{testScrapeConfigInfo}, testIstioCMPrometheusYml[PrometheusYml])
	if err != nil {
		t.Fatal(err)
	}
	validatePrometheusYml(t, prometheusYml)
}

func TestGetPrometheusScrapeConfig(t *testing.T) {
	componentBindingInfo := getTestGetComponentScrapeConfigInfo(t)
	scrapeConfig, err := getPrometheusScrapeConfig(componentBindingInfo)
	if err != nil {
		t.Fatal(err)
	}

	config := scrapeConfig.String()
	assert.True(t, strings.Contains(config, testKeepSourceLabels), "Missing expected replaced value.")
	assert.True(t, strings.Contains(config, testKeepSourceLabelsRegex), "Missing expected replaced value.")
	assert.False(t, strings.Contains(config, KeepSourceLabelsHolder), "Still have unwanted replacement holder(s).")
	assert.False(t, strings.Contains(config, KeepSourceLabelsRegexHolder), "Still have unwanted replacement holder(s).")

	jobName := scrapeConfig.S(PrometheusJobNameLabel).Data().(string)
	assert.Equal(t, testBindingname+"_"+testClusterName+"_"+testNamespace+"_"+testComponentName+"_"+testComponentBindingName, jobName, "Wrong job name.")
	namespace := scrapeConfig.S("kubernetes_sd_configs").Data().([]interface{})[0].(map[string]interface{})["namespaces"].(map[string]interface{})["names"].([]interface{})[0]
	assert.Equal(t, testNamespace, namespace, "Wrong namespace.")
	role := scrapeConfig.S("kubernetes_sd_configs").Data().([]interface{})[0].(map[string]interface{})["role"].(string)
	assert.Equal(t, "pod", role, "Wrong role.")
}

func TestGetHelidonScrapeConfigInfoList(t *testing.T) {
	pair := getTestModelBindingPair()
	addTestHelidonBinding(pair)
	addTestHelidonPlacement(pair)
	scrapeConfigInfoList, err := getComponentScrapeConfigInfoList(pair, createFakeSecretLister(), testClusterName)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(scrapeConfigInfoList), "Wrong list count for ScrapeConfigInfoList.")
	assert.Equal(t, testBindingname+"_"+testClusterName+"_"+testNamespace+"_"+HelidonName+"_"+testHelidonBindingName, scrapeConfigInfoList[0].PrometheusScrapeTargetJobName, "Wrong helidon jobname.")
}

func TestGetCoherenceScrapeConfigInfoList(t *testing.T) {
	pair := getTestModelBindingPair()
	addTestCoherenceBinding(pair)
	addTestCoherencePlacement(pair)
	scrapeConfigInfoList, err := getComponentScrapeConfigInfoList(pair, createFakeSecretLister(), testClusterName)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(scrapeConfigInfoList), "Wrong list count for ScrapeConfigInfoList.")
	assert.Equal(t, testBindingname+"_"+testClusterName+"_"+testNamespace+"_"+CoherenceName+"_"+testCoherenceBindingName, scrapeConfigInfoList[0].PrometheusScrapeTargetJobName, "Wrong Coherence jobname.")
	assert.Equal(t, testBindingname+"_"+testClusterName+"_"+testNamespace+"_"+CoherenceOperatorName, scrapeConfigInfoList[1].PrometheusScrapeTargetJobName, "Wrong Coherence jobname.")
}

func TestGetWeblogicScrapeConfigInfoList(t *testing.T) {
	pair := getTestModelBindingPair()
	addTestWeblogicBinding(pair)
	addTestWeblogicPlacement(pair)
	addTestWeblogicDomainToModel(pair)
	scrapeConfigInfoList, err := getComponentScrapeConfigInfoList(pair, createFakeSecretLister(), testClusterName)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(scrapeConfigInfoList), "Wrong list count for ScrapeConfigInfoList.")
	assert.Equal(t, testBindingname+"_"+testClusterName+"_"+testNamespace+"_"+WeblogicName+"_"+testWeblogicBindingName, scrapeConfigInfoList[0].PrometheusScrapeTargetJobName, "Wrong weblogic jobname.")
	assert.Equal(t, testBindingname+"_"+testClusterName+"_"+WeblogicOperatorName, scrapeConfigInfoList[1].PrometheusScrapeTargetJobName, "Wrong weblogic jobname.")
}

func TestGetComponentScrapeConfigInfo(t *testing.T) {
	componentBindingInfo := getTestGetComponentScrapeConfigInfo(t)
	assert.Equal(t, testBindingname, componentBindingInfo.BindingName, "Wrong binding name.")
	assert.Equal(t, testClusterName, componentBindingInfo.ManagedClusterName, "Wrong cluster name.")
	assert.Equal(t, testNamespace, componentBindingInfo.Namespace, "Wrong namespace.")
	assert.Equal(t, testKeepSourceLabels, componentBindingInfo.KeepSourceLabels, "Wrong keep source labels.")
	assert.Equal(t, testKeepSourceLabelsRegex, componentBindingInfo.KeepSourceLabelsRegex, "Wrong keep source labels regex.")
}

func TestGetWeblogicDomainCredentials(t *testing.T) {
	pair := getTestModelBindingPair()
	addTestWeblogicBinding(pair)
	addTestWeblogicDomainToModel(pair)
	username, password, err := getWeblogicDomainCredentials(pair, createFakeSecretLister(), testWeblogicBindingName)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, testWeblogicDomainUseranme, username, "Wrong weblogic username.")
	assert.Equal(t, testWeblogicDomainPassword, password, "Wrong weblogic password.")
}

func TestGetPrometheusScrapeConfigJobName(t *testing.T) {
	jobName := getPrometheusScrapeConfigJobName("bobs-books-binding", "my-vzno-managed-1", "bob", "weblogic", "bobs-bookstore")
	assert.Equal(t, "bobs-books-binding_my-vzno-managed-1_bob_weblogic_bobs-bookstore", jobName, "Wrong job name.")

	jobName = getPrometheusScrapeConfigJobName("bobs-books-binding", "my-vzno-managed-1", "bobby", "coherence-operator", "")
	assert.Equal(t, "bobs-books-binding_my-vzno-managed-1_bobby_coherence-operator", jobName, "Wrong job name.")

	jobName = getPrometheusScrapeConfigJobName("bobs-books-binding", "my-vzno-managed-1", "", "weblogic-operator", "")
	assert.Equal(t, "bobs-books-binding_my-vzno-managed-1_weblogic-operator", jobName, "Wrong job name.")
}

func TestFakeSecretNamespaceLister(t *testing.T) {
	secretNamespaceLister := createFakeSecretNamespaceLister()
	webLogicCredentialsSecret, _ := secretNamespaceLister.Get(testWeblogicCredentialsSecret)
	assert.Equal(t, testWeblogicDomainUseranme, string(webLogicCredentialsSecret.Data["username"]), "Wrong username.")
	assert.Equal(t, testWeblogicDomainPassword, string(webLogicCredentialsSecret.Data["password"]), "Wrong password.")
}

// ------------------------------
// | Unit Test helper functions |
// ------------------------------
func validatePrometheusYml(t *testing.T, prometheusYmlString string) {
	prometheusYmlJson, err := yaml.YAMLToJSON([]byte(prometheusYmlString))
	if err != nil {
		t.Fatal(err)
	}
	prometheusYml, err := gabs.ParseJSON(prometheusYmlJson)
	if err != nil {
		t.Fatal(err)
	}
	scrapeConfigs := prometheusYml.S(PrometheusScrapeConfigsLabel).Children()

	assert.Equal(t, "dummy", scrapeConfigs[0].Data().(map[string]interface{})[PrometheusJobNameLabel], "Original scrape job is missing.")

	newScrapeConfig := scrapeConfigs[1].Data().(map[string]interface{})
	assert.Equal(t, testScrapeTargetJobName, newScrapeConfig[PrometheusJobNameLabel], "New scrape job is missing.")
	assert.Equal(t, testWeblogicDomainUseranme, newScrapeConfig[BasicAuthLabel].(map[string]interface{})[UsernameLabel], "Weblogic username is missing.")
	assert.Equal(t, testWeblogicDomainPassword, newScrapeConfig[BasicAuthLabel].(map[string]interface{})[PasswordLabel], "Weblogic password is missing.")
	assert.Equal(t, testNamespace, newScrapeConfig["kubernetes_sd_configs"].([]interface{})[0].(map[string]interface{})["namespaces"].(map[string]interface{})["names"].([]interface{})[0], "New scrape job is missing.")
}

// -------------------------
// | Test helper functions |
// -------------------------
func getTestIstioPrometheusCM() *apicorev1.ConfigMap {
	return &apicorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      IstioPrometheusCMName,
			Namespace: IstioNamespace,
		},
		Data: map[string]string{PrometheusYml: testIstioCMPrometheusYml[PrometheusYml]},
	}
}

func getTestVznoModel(modelName string, namespace string) *v1beta1v8o.VerrazzanoModel {
	return &v1beta1v8o.VerrazzanoModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelName,
			Namespace: namespace,
		},
		Spec: v1beta1v8o.VerrazzanoModelSpec{
			Description: "test model",
		},
	}
}

func getTestVznoBinding(bindingName string, namespace string, modelName string, componentNameToCluster map[string]string) *v1beta1v8o.VerrazzanoBinding {
	vznoBinding := &v1beta1v8o.VerrazzanoBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bindingName,
			Namespace: namespace,
		},
		Spec: v1beta1v8o.VerrazzanoBindingSpec{
			ModelName: modelName,
		},
	}

	for componentName, clusterName := range componentNameToCluster {
		placement := v1beta1v8o.VerrazzanoPlacement{
			Name: clusterName,
			Namespaces: []v1beta1v8o.KubernetesNamespace{
				{
					Name: namespace,
					Components: []v1beta1v8o.BindingComponent{
						{
							Name: componentName,
						},
					},
				},
			},
		}
		vznoBinding.Spec.Placement = append(vznoBinding.Spec.Placement, placement)
	}
	return vznoBinding
}

func getTestModelBindingPair() *types.ModelBindingPair {
	vznoModel := getTestVznoModel(testModelName, testNamespace)
	vznoBinding := getTestVznoBinding(testBindingname, testNamespace, testModelName, map[string]string{testComponentBindingName: testClusterName})

	return &types.ModelBindingPair{
		Model:           vznoModel,
		Binding:         vznoBinding,
		ManagedClusters: map[string]*types.ManagedCluster{},
		Lock:            sync.RWMutex{},
	}
}

func addTestHelidonBinding(mbPair *types.ModelBindingPair) {
	helidonBinding := &v1beta1v8o.VerrazzanoHelidonBinding{
		Name:     testHelidonBindingName,
		Replicas: &testReplicas}
	mbPair.Binding.Spec.HelidonBindings = append(mbPair.Binding.Spec.HelidonBindings, *helidonBinding)
}

func addTestHelidonPlacement(mbPair *types.ModelBindingPair) {
	helidonPlacement := v1beta1v8o.VerrazzanoPlacement{
		Name: testClusterName,
		Namespaces: []v1beta1v8o.KubernetesNamespace{
			{
				Name: testNamespace,
				Components: []v1beta1v8o.BindingComponent{
					{
						Name: testHelidonBindingName,
					},
				},
			},
		},
	}
	mbPair.Binding.Spec.Placement = append(mbPair.Binding.Spec.Placement, helidonPlacement)
}

func addTestCoherenceBinding(mbPair *types.ModelBindingPair) {
	coherenceBinding := &v1beta1v8o.VerrazzanoCoherenceBinding{
		Name:     testCoherenceBindingName,
		Replicas: &testReplicas}
	mbPair.Binding.Spec.CoherenceBindings = append(mbPair.Binding.Spec.CoherenceBindings, *coherenceBinding)
}

func addTestCoherencePlacement(mbPair *types.ModelBindingPair) {
	coherencePlacement := v1beta1v8o.VerrazzanoPlacement{
		Name: testClusterName,
		Namespaces: []v1beta1v8o.KubernetesNamespace{
			{
				Name: testNamespace,
				Components: []v1beta1v8o.BindingComponent{
					{
						Name: testCoherenceBindingName,
					},
				},
			},
		},
	}
	mbPair.Binding.Spec.Placement = append(mbPair.Binding.Spec.Placement, coherencePlacement)
}

func addTestWeblogicBinding(mbPair *types.ModelBindingPair) {
	weblogicBinding := &v1beta1v8o.VerrazzanoWeblogicBinding{
		Name:     testWeblogicBindingName,
		Replicas: &testReplicas}
	mbPair.Binding.Spec.WeblogicBindings = append(mbPair.Binding.Spec.WeblogicBindings, *weblogicBinding)
}

func addTestWeblogicPlacement(mbPair *types.ModelBindingPair) {
	weblogicPlacement := v1beta1v8o.VerrazzanoPlacement{
		Name: testClusterName,
		Namespaces: []v1beta1v8o.KubernetesNamespace{
			{
				Name: testNamespace,
				Components: []v1beta1v8o.BindingComponent{
					{
						Name: testWeblogicBindingName,
					},
				},
			},
		},
	}
	mbPair.Binding.Spec.Placement = append(mbPair.Binding.Spec.Placement, weblogicPlacement)
}

func addTestWeblogicDomainToModel(mbPair *types.ModelBindingPair) {
	webLogicDomain := &v1beta1v8o.VerrazzanoWebLogicDomain{
		Name: testWeblogicBindingName,
		DomainCRValues: v8.DomainSpec{
			WebLogicCredentialsSecret: apicorev1.SecretReference{
				Name: testWeblogicCredentialsSecret,
			},
		},
	}
	mbPair.Model.Spec.WeblogicDomains = append(mbPair.Model.Spec.WeblogicDomains, *webLogicDomain)
}

func getTestGetComponentScrapeConfigInfo(t *testing.T) ScrapeConfigInfo {
	mbPair := getTestModelBindingPair()
	componentBindingInfo, err := getComponentScrapeConfigInfo(mbPair, testComponentBindingName, testClusterName, testComponentName,
		testKeepSourceLabels, testKeepSourceLabelsRegex)
	if err != nil {
		t.Fatal(err)
	}
	return componentBindingInfo
}

// -------------------------
// | Fake Kubernetes Objects |
// -------------------------

// Fake SecretNamespaceLister
type fakeSecretNamespaceLister struct{}

func (f fakeSecretNamespaceLister) List(selector labels.Selector) (ret []*apicorev1.Secret, err error) {
	panic("Not implemented!")
}

func (f fakeSecretNamespaceLister) Get(name string) (*apicorev1.Secret, error) {
	if name != testWeblogicCredentialsSecret {
		return nil, fmt.Errorf("Unexpected weblogic credential secret name. Expected %s, but got %s.", testWeblogicCredentialsSecret, name)
	}

	weblogicSecret := &apicorev1.Secret{
		Data: map[string][]byte{
			"username": []byte(testWeblogicDomainUseranme),
			"password": []byte(testWeblogicDomainPassword),
		},
	}
	return weblogicSecret, nil
}
func createFakeSecretNamespaceLister() corev1listers.SecretNamespaceLister {
	return fakeSecretNamespaceLister{}
}

// Fake SecretLister
type fakeSecretLister struct{}

func (f fakeSecretLister) List(selector labels.Selector) (ret []*apicorev1.Secret, err error) {
	panic("Not implemented!")
}

func (f fakeSecretLister) Secrets(namespace string) corev1listers.SecretNamespaceLister {
	return createFakeSecretNamespaceLister()
}

func createFakeSecretLister() corev1listers.SecretLister {
	return fakeSecretLister{}
}

// Fake ConfigMapLister
type fakeConfigMapLister struct{}

func (f fakeConfigMapLister) List(selector labels.Selector) (ret []*apicorev1.ConfigMap, err error) {
	panic("Not implemented!")
}

func (f fakeConfigMapLister) ConfigMaps(namespace string) corev1listers.ConfigMapNamespaceLister {
	return createFakeConfigMapNamespaceLister()
}

func createFakeConfigMapLister() corev1listers.ConfigMapLister {
	return fakeConfigMapLister{}
}

// Fake ConfigMapNamespaceLister
type fakeConfigMapNamespaceLister struct{}

func (f fakeConfigMapNamespaceLister) List(selector labels.Selector) (ret []*apicorev1.ConfigMap, err error) {
	panic("Not implemented")
}

func (f fakeConfigMapNamespaceLister) Get(name string) (*apicorev1.ConfigMap, error) {
	return getTestIstioPrometheusCM(), nil
}

func createFakeConfigMapNamespaceLister() corev1listers.ConfigMapNamespaceLister {
	return fakeConfigMapNamespaceLister{}
}
