// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of VMI CRs, based on a VerrazzanoBinding

package local

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/verrazzano/pkg/diff"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmoclientset "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var createInstanceFunc = createInstance

// CreateUpdateVmi creates/updates Verrazzano Monitoring Instances for a given binding.
func CreateUpdateVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error {
	zap.S().Debugf("Creating/updating Local (Management SynModel) VMI for VerrazzanoBinding %s", binding.Name)

	if util.SharedVMIDefault() && !util.IsSystemProfileBindingName(binding.Name) {
		zap.S().Infof("Using shared VMI for binding %s", binding.Name)
		return nil
	}

	// Construct the expected VMI
	newVmi, err := createInstanceFunc(binding, verrazzanoURI, enableMonitoringStorage)
	if err != nil {
		return err
	}

	// Create or update VMIs
	existingVmi, err := vmiLister.VerrazzanoMonitoringInstances(newVmi.Namespace).Get(newVmi.Name)
	if existingVmi != nil {
		newVmi.Spec.Grafana.Storage.PvcNames = existingVmi.Spec.Grafana.Storage.PvcNames
		newVmi.Spec.Prometheus.Storage.PvcNames = existingVmi.Spec.Prometheus.Storage.PvcNames
		newVmi.Spec.Elasticsearch.Storage.PvcNames = existingVmi.Spec.Elasticsearch.Storage.PvcNames
		specDiffs := diff.Diff(existingVmi, newVmi)
		if specDiffs != "" {
			zap.S().Debugf("VMI %s : Spec differences %s", newVmi.Name, specDiffs)
			zap.S().Infof("Updating VMI %s/%s", newVmi.Namespace, newVmi.Name)
			newVmi.ResourceVersion = existingVmi.ResourceVersion
			_, err = vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(newVmi.Namespace).Update(context.TODO(), newVmi, metav1.UpdateOptions{})
		}
	} else {
		zap.S().Infof("Creating VMI %s%s", newVmi.Namespace, newVmi.Name)
		_, err = vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(newVmi.Namespace).Create(context.TODO(), newVmi, metav1.CreateOptions{})
	}
	if err != nil {
		zap.S().Infof("Failed to create/update VMI %s/%s: %v", newVmi.Namespace, newVmi.Name, err)
		return err
	}
	return nil
}

// DeleteVmi deletes Verrazzano Monitoring Instances for a given binding.
func DeleteVmi(binding *types.SyntheticBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {
	zap.S().Infof("Deleting Local (Management SynModel) VMIs for VerrazzanoBinding %s", binding.Name)

	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})

	existingVMIsList, err := vmiLister.VerrazzanoMonitoringInstances("").List(selector)
	if err != nil {
		return err
	}
	for _, existingVmi := range existingVMIsList {
		zap.S().Infof("Deleting VMI %s", existingVmi.Name)
		err := vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(existingVmi.Namespace).Delete(context.TODO(), existingVmi.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// Create a storage setting based on the specified value if monitoring storage is enabled
func createStorageOption(envSetting string, enableMonitoringStorageEnvFlag string) vmov1.Storage {
	storageSetting := vmov1.Storage{
		Size: "",
	}
	monitoringStorageEnabled, err := strconv.ParseBool(enableMonitoringStorageEnvFlag)
	if err != nil {
		zap.S().Errorf("Failed, invalid storage setting: %s", enableMonitoringStorageEnvFlag)
	} else if monitoringStorageEnabled && len(envSetting) > 0 {
		storageSetting = vmov1.Storage{
			Size: envSetting,
		}
	}
	return storageSetting
}

// Constructs the necessary VerrazzanoMonitoringInstance for the given VerrazzanoBinding
func createInstance(binding *types.SyntheticBinding, verrazzanoURI string, enableMonitoringStorage string) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if verrazzanoURI == "" {
		return nil, errors.New("verrazzanoURI must not be empty")
	}

	bindingLabels := util.GetLocalBindingLabels(binding)

	bindingName := util.GetProfileBindingName(binding.Name)

	return &vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetVmiNameForBinding(binding.Name),
			Namespace: constants.VerrazzanoNamespace,
			Labels:    bindingLabels,
		},
		Spec: vmov1.VerrazzanoMonitoringInstanceSpec{
			URI:             util.GetVmiURI(bindingName, verrazzanoURI),
			AutoSecret:      true,
			SecretsName:     constants.VmiSecretName,
			CascadingDelete: true,
			AlertManager: vmov1.AlertManager{
				Enabled: true,
				Resources: vmov1.Resources{
					RequestMemory: util.GetPrometheusRequestMemory(),
				},
			},
			Grafana: vmov1.Grafana{
				Enabled:             util.GetGrafanaEnabled(),
				Storage:             createStorageOption(util.GetGrafanaDataStorageSize(), enableMonitoringStorage),
				DashboardsConfigMap: util.GetVmiNameForBinding(binding.Name) + "-dashboards",
				Resources: vmov1.Resources{
					RequestMemory: util.GetGrafanaRequestMemory(),
				},
			},
			IngressTargetDNSName: fmt.Sprintf("verrazzano-ingress.%s", verrazzanoURI),
			Prometheus: vmov1.Prometheus{
				Enabled: util.GetPrometheusEnabled(),
				Storage: createStorageOption(util.GetPrometheusDataStorageSize(), enableMonitoringStorage),
				Resources: vmov1.Resources{
					RequestMemory: util.GetPrometheusRequestMemory(),
				},
			},
			Elasticsearch: vmov1.Elasticsearch{
				Enabled: util.GetElasticsearchEnabled(),
				Storage: createStorageOption(util.GetElasticsearchDataStorageSize(), enableMonitoringStorage),
				IngestNode: vmov1.ElasticsearchNode{
					Replicas: util.GetElasticsearchIngestNodeReplicas(),
					Resources: vmov1.Resources{
						RequestMemory: util.GetElasticsearchIngestNodeRequestMemory(),
					},
				},
				MasterNode: vmov1.ElasticsearchNode{
					Replicas: util.GetElasticsearchMasterNodeReplicas(),
					Resources: vmov1.Resources{
						RequestMemory: util.GetElasticsearchMasterNodeRequestMemory(),
					},
				},
				DataNode: vmov1.ElasticsearchNode{
					Replicas: util.GetElasticsearchDataNodeReplicas(),
					Resources: vmov1.Resources{
						RequestMemory: util.GetElasticsearchDataNodeRequestMemory(),
					},
				},
			},
			Kibana: vmov1.Kibana{
				Enabled: util.GetKibanaEnabled(),
				Resources: vmov1.Resources{
					RequestMemory: util.GetKibanaRequestMemory(),
				},
			},
			ServiceType: "ClusterIP",
		},
	}, nil
}
