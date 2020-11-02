// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of VMI CRs, based on a VerrazzanoBinding

package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmoclientset "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	vmolisters "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/listers/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CreateUpdateVmi creates/updates Verrazzano Monitoring Instances for a given binding.
func CreateUpdateVmi(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoURI string, enableMonitoringStorage string) error {

	glog.V(6).Infof("Creating/updating Local (Management Cluster) VMI for VerrazzanoBinding %s", binding.Name)

	// Construct the expected VMI
	newVmi, err := createInstance(binding, verrazzanoURI, enableMonitoringStorage)
	if err != nil {
		return err
	}

	// Create or update VMIs
	existingVmi, err := vmiLister.VerrazzanoMonitoringInstances(newVmi.Namespace).Get(newVmi.Name)
	if existingVmi != nil {
		newVmi.Spec.Grafana.Storage.PvcNames = existingVmi.Spec.Grafana.Storage.PvcNames
		newVmi.Spec.Prometheus.Storage.PvcNames = existingVmi.Spec.Prometheus.Storage.PvcNames
		newVmi.Spec.Elasticsearch.Storage.PvcNames = existingVmi.Spec.Elasticsearch.Storage.PvcNames
		specDiffs := diff.CompareIgnoreTargetEmpties(existingVmi, newVmi)
		if specDiffs != "" {
			glog.V(6).Infof("VMI %s : Spec differences %s", newVmi.Name, specDiffs)
			glog.V(4).Infof("Updating VMI %s", newVmi.Name)
			newVmi.ResourceVersion = existingVmi.ResourceVersion
			_, err = vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(newVmi.Namespace).Update(context.TODO(), newVmi, metav1.UpdateOptions{})
		}
	} else {
		glog.V(4).Infof("Creating VMI %s", newVmi.Name)
		_, err = vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(newVmi.Namespace).Create(context.TODO(), newVmi, metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}
	return nil
}

// DeleteVmi deletes Verrazzano Monitoring Instances for a given binding.
func DeleteVmi(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {

	glog.V(4).Infof("Deleting Local (Management Cluster) VMIs for VerrazzanoBinding %s", binding.Name)

	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})

	existingVMIsList, err := vmiLister.VerrazzanoMonitoringInstances("").List(selector)
	if err != nil {
		return err
	}
	for _, existingVmi := range existingVMIsList {
		glog.V(4).Infof("Deleting VMI %s", existingVmi.Name)
		err := vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(existingVmi.Namespace).Delete(context.TODO(), existingVmi.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func createStorageOption(enableMonitoringStorage string) vmov1.Storage {
	if strings.ToLower(enableMonitoringStorage) == "false" {
		return vmov1.Storage{
			Size: "",
		}
	}
	return vmov1.Storage{
		Size: "50Gi",
	}
}

// Constructs the necessary VerrazzanoMonitoringInstance for the given VerrazzanoBinding
func createInstance(binding *v1beta1v8o.VerrazzanoBinding, verrazzanoURI string, enableMonitoringStorage string) (*vmov1.VerrazzanoMonitoringInstance, error) {
	if verrazzanoURI == "" {
		return nil, errors.New("verrazzanoURI must not be empty")
	}

	bindingLabels := util.GetLocalBindingLabels(binding)

	storageOption := createStorageOption(enableMonitoringStorage)

	_, present := os.LookupEnv("singleSystemVMI")
	glog.Infof("CDD Env var SingleSystemVMI present? %v", present)
	var bindingName string
	if present {
		bindingName = constants.VmiSystemBindingName
	} else {
		bindingName = binding.Name
	}

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
			Grafana: vmov1.Grafana{
				Enabled:             true,
				Storage:             storageOption,
				DashboardsConfigMap: util.GetVmiNameForBinding(bindingName) + "-dashboards",
				Resources: vmov1.Resources{
					RequestMemory: util.GetGrafanaRequestMemory(),
				},
			},
			IngressTargetDNSName: fmt.Sprintf("verrazzano-ingress.%s", verrazzanoURI),
			Prometheus: vmov1.Prometheus{
				Enabled: true,
				Storage: storageOption,
				Resources: vmov1.Resources{
					RequestMemory: util.GetPrometheusRequestMemory(),
				},
			},
			Elasticsearch: vmov1.Elasticsearch{
				Enabled: true,
				Storage: storageOption,
				IngestNode: vmov1.ElasticsearchNode{
					Replicas: 1,
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
					Replicas: 2,
					Resources: vmov1.Resources{
						RequestMemory: util.GetElasticsearchDataNodeRequestMemory(),
					},
				},
			},
			Kibana: vmov1.Kibana{
				Enabled: true,
				Resources: vmov1.Resources{
					RequestMemory: util.GetKibanaRequestMemory(),
				},
			},
			ServiceType: "ClusterIP",
		},
	}, nil
}
