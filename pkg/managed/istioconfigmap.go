// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/golang/glog"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/yaml"
)

type ScrapeConfigInfo struct {
	PrometheusScrapeTargetJobName string
	Namespace                     string
	KeepSourceLabels              string
	KeepSourceLabelsRegex         string
	BindingName                   string
	ComponentBindingName          string
	ManagedClusterName            string
	Username                      string
	Password                      string
}

// Updating the istio prometheus scrape configs to include all the node exporter endpoints in monitoring namespace
func UpdateIstioPrometheusConfigMaps(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {

	if mbPair.Binding.Name == constants.VmiSystemBindingName {
		UpdateIstioPrometheusConfigMapsForNodeExporter(mbPair, availableManagedClusterConnections)
	} else {
		UpdateIstioPrometheusConfigMapsForBinding(mbPair, secretLister, availableManagedClusterConnections)
	}
	return nil
}

// Updating the istio prometheus scrape configs to include all the node exporter endpoints in monitoring namespace
func UpdateIstioPrometheusConfigMapsForNodeExporter(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	glog.V(6).Infof("Add/Update node export scrape target in Istio Prometheus configmaps for all clusters.")

	// Construct prometheus.yml configMaps for each managed cluster
	filteredConnections, err := GetFilteredConnections(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct services for each ManagedCluster
	for clusterName := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		istioPrometheusCM, err := managedClusterConnection.ConfigMapLister.ConfigMaps(IstioNamespace).Get(IstioPrometheusCMName)
		if err != nil {
			return err
		}

		// Construct the expected prometheus config map
		scrapeConfigInfoList := []ScrapeConfigInfo{}
		scrapeConfigInfoList = append(scrapeConfigInfoList, ScrapeConfigInfo{
			PrometheusScrapeTargetJobName: NodeExporterName,
			Namespace:                     NodeExporterNamespace,
			KeepSourceLabels:              NodeExporterKeepSourceLabels,
			KeepSourceLabelsRegex:         NodeExporterKeepSourceLabelsRegex,
			ManagedClusterName:            clusterName,
		})

		// Create/Update istio prometheus config map
		if err := updateIstioPrometheusConfigMap(clusterName, managedClusterConnection, scrapeConfigInfoList, istioPrometheusCM.Data); err != nil {
			return err
		}
	}
	return nil
}

// Updating the istio prometheus scrape configs to include all the component endpoints from a given verrazzano model binding pair.
// Different scrape targets/jobs will be added to differentiate the binding/cluster/namespace/component/componentBinding combinations.
// job_name: <binding_name>_<cluster_name>_<namespace>_<component_name>_<component_binding_name>
//
// For examples:
//    # weblogic binding
//    myvb_poc-managed-1_weblogic-operator (one per managed cluster per verrazzano binding if needed)
//    myvb_poc-managed-1_myapps_weblogic_my-bookstore-ui
//    myvb_poc-managed-1_myapps_weblogic_my-bookstore-backend
//
//    # helidon binding
//    myvb_poc-managed-1_myapps_helidon_my-stock-app
//
//    # coherence binding
//    myvb_poc-managed-1_myapps_coherence-operator (one per namespace if needed)
//    myvb_poc-managed-1_myapps_coherence_my-cache-app
//
func UpdateIstioPrometheusConfigMapsForBinding(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, clusterConnections map[string]*util.ManagedClusterConnection) error {
	glog.V(6).Infof("Create/Update Istio Prometheus configmap [binding]%s [model]%s", mbPair.Binding.Name, mbPair.Model.Name)

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, clusterConnections)
	if err != nil {
		return err
	}

	// Construct prometheus.yml configMaps for each managed cluster
	for clusterName := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		istioPrometheusCM, err := managedClusterConnection.ConfigMapLister.ConfigMaps(IstioNamespace).Get(IstioPrometheusCMName)
		if err != nil {
			return err
		}

		// Retrieve all component bindings info from a specific managed cluster
		scrapeConfigInfoList, err := getComponentScrapeConfigInfoList(mbPair, secretLister, clusterName)
		if err != nil {
			return err
		}

		// Create/Update istio prometheus config map
		if err := updateIstioPrometheusConfigMap(clusterName, managedClusterConnection, scrapeConfigInfoList, istioPrometheusCM.Data); err != nil {
			return err
		}
	}
	return nil
}

// Creating/Updating prometheus config map
func updateIstioPrometheusConfigMap(clusterName string, managedClusterConnection *util.ManagedClusterConnection,
	scrapeConfigInfoList []ScrapeConfigInfo, istioPrometheusCMData map[string]string) error {
	// Construct the expected prometheus config map
	newPrometheusConfigMap, err := getNewPrometheusConfigMap(scrapeConfigInfoList, istioPrometheusCMData)
	if err != nil {
		return err
	}

	// Create a new istio prometheus configmap if necessary
	existingcm, err := managedClusterConnection.ConfigMapLister.ConfigMaps(IstioNamespace).Get(IstioPrometheusCMName)
	if err != nil {
		return err
	}
	if existingcm == nil {
		glog.V(4).Infof("Creating Configmaps %s in cluster %s", newPrometheusConfigMap.Name, clusterName)
		_, err = managedClusterConnection.KubeClient.CoreV1().ConfigMaps(newPrometheusConfigMap.Namespace).Create(context.TODO(), newPrometheusConfigMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	// Update the existing istio prometheus configmap if necessary
	specDiffs := diff.CompareIgnoreTargetEmpties(existingcm, newPrometheusConfigMap)
	if specDiffs != "" {
		glog.V(6).Infof("ConfigMap %s : Spec differences %s", newPrometheusConfigMap.Name, specDiffs)
		glog.V(4).Infof("Updating ConfigMap %s in cluster %s", newPrometheusConfigMap.Name, clusterName)
		_, err = managedClusterConnection.KubeClient.CoreV1().ConfigMaps(newPrometheusConfigMap.Namespace).Update(context.TODO(), newPrometheusConfigMap, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		// Bounce Prometheus Pod
		istioPodName, err := managedClusterConnection.KubeClient.CoreV1().Pods(IstioNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: PrometheusLabel})
		for _, pod := range istioPodName.Items {
			glog.V(4).Infof("Deleting pod %s in namespace %s in cluster %s", pod.Name, IstioNamespace, clusterName)
			err = managedClusterConnection.KubeClient.CoreV1().Pods(IstioNamespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Constructing a new prometheus configmap with all the component bindings within a given cluster defined in the VerrazzanoBinding
func getNewPrometheusConfigMap(scrapeConfigInfoList []ScrapeConfigInfo, cmdata map[string]string) (*corev1.ConfigMap, error) {
	// Get the existing istio Prometheus config
	currPrometheusYml := cmdata[PrometheusYml]

	// Prepare the new expected Prometheus config
	newPrometheusYml, err := updatePrometheusYml(scrapeConfigInfoList, currPrometheusYml)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      IstioPrometheusCMName,
			Namespace: IstioNamespace,
		},
		Data: map[string]string{PrometheusYml: newPrometheusYml},
	}, nil
}

// Updating prometheus.yml with all the component bindings within a given cluster
func updatePrometheusYml(scrapeConfigInfoList []ScrapeConfigInfo, currPrometheusYml string) (string, error) {
	// Parse current Prometheus config
	prometheusYmlJson, err := yaml.YAMLToJSON([]byte(currPrometheusYml))
	if err != nil {
		return "", err
	}
	prometheusYml, err := gabs.ParseJSON(prometheusYmlJson)
	if err != nil {
		return "", err
	}

	// Add Scrape Config for each component binding
	for _, scrapeConfigInfo := range scrapeConfigInfoList {
		scrapeConfig, err := getPrometheusScrapeConfig(scrapeConfigInfo)
		if err != nil {
			return "", err
		}

		// Preserve/Update/Add scrape configs
		existingScrapeConfigs := prometheusYml.S(PrometheusScrapeConfigsLabel).Children()
		prometheusYml.Array(PrometheusScrapeConfigsLabel) // zero out the array
		existingFound := false
		for _, scrapeConfig := range existingScrapeConfigs {
			jobName := scrapeConfig.S(PrometheusJobNameLabel).Data()
			if jobName == scrapeConfigInfo.PrometheusScrapeTargetJobName && !existingFound {
				prometheusYml.ArrayAppendP(scrapeConfig.Data(), PrometheusScrapeConfigsLabel)
				existingFound = true
			} else {
				prometheusYml.ArrayAppendP(scrapeConfig.Data(), PrometheusScrapeConfigsLabel)
			}
		}
		if !existingFound {
			prometheusYml.ArrayAppendP(scrapeConfig.Data(), PrometheusScrapeConfigsLabel)
		}
	}

	// Return prometheus.yml in string format
	prometheusYmlString, err := yaml.JSONToYAML(prometheusYml.Bytes())
	if err != nil {
		return "", err
	}
	return string(prometheusYmlString), nil
}

// Getting prometheus scrape config for a given component binding
func getPrometheusScrapeConfig(scrapeConfigInfo ScrapeConfigInfo) (*gabs.Container, error) {
	// Replace all the place holders from the prometheus scrape config template
	scrapeConfigTemplate := PrometheusScrapeConfigTemplate
	scrapeConfigTemplate = strings.Replace(scrapeConfigTemplate, JobNameHolder, scrapeConfigInfo.PrometheusScrapeTargetJobName, -1)
	scrapeConfigTemplate = strings.Replace(scrapeConfigTemplate, NamespaceHolder, scrapeConfigInfo.Namespace, -1)
	scrapeConfigTemplate = strings.Replace(scrapeConfigTemplate, KeepSourceLabelsHolder, scrapeConfigInfo.KeepSourceLabels, -1)
	scrapeConfigTemplate = strings.Replace(scrapeConfigTemplate, KeepSourceLabelsRegexHolder, scrapeConfigInfo.KeepSourceLabelsRegex, -1)

	// Parse out the prometheus scrape config
	scrapeConfigJson, err := yaml.YAMLToJSON([]byte(scrapeConfigTemplate))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failure to parse built-in Prometheus job template: %v", err))
	}
	scrapeConfig, err := gabs.ParseJSON(scrapeConfigJson)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failure to parse built-in Prometheus job template: %v", err))
	}

	// Add basic auth
	if scrapeConfigInfo.Username != "" && scrapeConfigInfo.Password != "" {
		scrapeConfig.Set(scrapeConfigInfo.Username, BasicAuthLabel, UsernameLabel)
		scrapeConfig.Set(scrapeConfigInfo.Password, BasicAuthLabel, PasswordLabel)
	}

	return scrapeConfig, nil
}

// Getting a list of component scrape config info for a given cluster
func getComponentScrapeConfigInfoList(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, clusterName string) ([]ScrapeConfigInfo, error) {
	scrapeConfigInfoList := []ScrapeConfigInfo{}

	// Get all weblogic bindings info
	hasWeblogicBinding := false
	for _, component := range mbPair.Model.Spec.WeblogicDomains {
		// Get the common binding info
		componnentBindingInfo, err := getComponentScrapeConfigInfo(mbPair, component.Name, clusterName, WeblogicName, WeblogicKeepSourceLabels,
			strings.Replace(WeblogicKeepSourceLabelsRegex, ComponentBindingNameHolder, component.Name, -1))
		if err != nil {
			return nil, err
		}

		if componnentBindingInfo.BindingName != "" {
			// Get username and password for the weblogic doamin
			username, password, err := getWeblogicDomainCredentials(mbPair, secretLister, component.Name)
			if err != nil {
				return nil, err
			}
			componnentBindingInfo.Username = username
			componnentBindingInfo.Password = password

			scrapeConfigInfoList = append(scrapeConfigInfoList, componnentBindingInfo)
			hasWeblogicBinding = true
		}
	}

	// Add weblogic operator binding info by managed cluster for each verrazzano binding
	// A new weblogic Operator is deployed to cluster when one or more weblocgic component(s) is placed in the managed cluster
	if hasWeblogicBinding {
		scrapeConfigInfoList = append(scrapeConfigInfoList, ScrapeConfigInfo{
			PrometheusScrapeTargetJobName: getPrometheusScrapeConfigJobName(mbPair.Binding.Name, clusterName, "", WeblogicOperatorName, ""),
			Namespace:                     "verrazzano-" + mbPair.Binding.Name,
			KeepSourceLabels:              WeblogicOperatorKeepSourceLabels,
			KeepSourceLabelsRegex:         WeblogicOperatorKeepSourceLabelsRegex,
			BindingName:                   mbPair.Binding.Name,
			ManagedClusterName:            clusterName,
		})
	}

	// Get all coherence bindings info
	var coherenceNamespaces []string
	for _, component := range mbPair.Model.Spec.CoherenceClusters {
		// Get the common binding info
		componnentBindingInfo, err := getComponentScrapeConfigInfo(mbPair, component.Name, clusterName, CoherenceName, CoherenceKeepSourceLabels,
			strings.Replace(CoherenceKeepSourceLabelsRegex, ComponentBindingNameHolder, component.Name, -1))
		if err != nil {
			return nil, err
		}

		if componnentBindingInfo.BindingName != "" {
			scrapeConfigInfoList = append(scrapeConfigInfoList, componnentBindingInfo)

			namespaceAlreadyExist := false
			for _, namespace := range coherenceNamespaces {
				if namespace == componnentBindingInfo.Namespace {
					namespaceAlreadyExist = true
					break
				}
			}
			if !namespaceAlreadyExist {
				coherenceNamespaces = append(coherenceNamespaces, componnentBindingInfo.Namespace)
			}
		}
	}

	// Add coherence operator binding info by namespace
	// Coherence Operator is deployed to all namespaces where one or more coherence component(s) is placed
	if len(coherenceNamespaces) > 0 {
		for _, namespace := range coherenceNamespaces {
			scrapeConfigInfoList = append(scrapeConfigInfoList, ScrapeConfigInfo{
				PrometheusScrapeTargetJobName: getPrometheusScrapeConfigJobName(mbPair.Binding.Name, clusterName, namespace, CoherenceOperatorName, ""),
				Namespace:                     namespace,
				KeepSourceLabels:              CoherenceOperatorKeepSourceLabels,
				KeepSourceLabelsRegex:         CoherenceOperatorKeepSourceLabelsRegex,
				BindingName:                   mbPair.Binding.Name,
				ManagedClusterName:            clusterName,
			})
		}
	}

	// Get all helidon bindings info
	for _, component := range mbPair.Model.Spec.HelidonApplications {
		// Get the common binding info
		componnentBindingInfo, err := getComponentScrapeConfigInfo(mbPair, component.Name, clusterName, HelidonName, HelidonKeepSourceLabels,
			strings.Replace(HelidonKeepSourceLabelsRegex, ComponentBindingNameHolder, component.Name, -1))
		if err != nil {
			return nil, err
		}

		if componnentBindingInfo.BindingName != "" {
			scrapeConfigInfoList = append(scrapeConfigInfoList, componnentBindingInfo)
		}
	}

	return scrapeConfigInfoList, nil
}

// Getting the component binding info for a given component binding
func getComponentScrapeConfigInfo(mbPair *types.ModelBindingPair, componentBindingName string, clusterName string, componentName string,
	keepSourceLabels string, keepSourceLabelsRegex string) (ScrapeConfigInfo, error) {
	for _, placement := range mbPair.Binding.Spec.Placement {
		if placement.Name == clusterName {
			for _, namespace := range placement.Namespaces {
				for _, component := range namespace.Components {
					if component.Name == componentBindingName {
						return ScrapeConfigInfo{
							PrometheusScrapeTargetJobName: getPrometheusScrapeConfigJobName(mbPair.Binding.Name, clusterName, namespace.Name, componentName, componentBindingName),
							Namespace:                     namespace.Name,
							KeepSourceLabels:              keepSourceLabels,
							KeepSourceLabelsRegex:         keepSourceLabelsRegex,
							BindingName:                   mbPair.Binding.Name,
							ComponentBindingName:          componentBindingName,
							ManagedClusterName:            clusterName,
						}, nil
					}
				}
			}
		}
	}
	return ScrapeConfigInfo{}, nil
}

// Getting the weblogic domain credentials for a given weblogic binding
func getWeblogicDomainCredentials(mbPair *types.ModelBindingPair, secretLister corev1listers.SecretLister, componentBindingName string) (string, string, error) {
	var webLogicCredentialsSecretName string
	weblogicDomains := mbPair.Model.Spec.WeblogicDomains
	if weblogicDomains != nil {
		for _, weblogicDomain := range weblogicDomains {
			if weblogicDomain.Name == componentBindingName {
				webLogicCredentialsSecretName = weblogicDomain.DomainCRValues.WebLogicCredentialsSecret.Name
				break
			}
		}
	}

	webLogicCredentialsSecret, err := secretLister.Secrets(constants.DefaultNamespace).Get(webLogicCredentialsSecretName)
	if err != nil {
		return "", "", fmt.Errorf("Failure to retrieve secret [%q]: %v", webLogicCredentialsSecretName, err)
	}
	return string(webLogicCredentialsSecret.Data["username"]), string(webLogicCredentialsSecret.Data["password"]), nil
}

// Getting the prometheus scrape config job name for a given component binding within certain cluster and namespace
func getPrometheusScrapeConfigJobName(bindingName string, clusterName string, namespace string, componentName string, componentBindingName string) string {
	jobName := bindingName + "_" + clusterName
	if namespace != "" {
		jobName = jobName + "_" + namespace
	}
	jobName = jobName + "_" + componentName
	if componentBindingName != "" {
		jobName = jobName + "_" + componentBindingName
	}
	return jobName
}
