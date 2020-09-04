// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Handles creation/deletion of VMI CRs, based on a VerrazzanoBinding
package local

import (
	"context"
	"fmt"
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

func CreateVmis(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister, verrazzanoUri string, enableMonitoringStorage string) error {

	glog.V(6).Infof("Creating/updating Local (Management Cluster) VMI for VerrazzanoBinding %s", binding.Name)

	// Construct the set of expected VMIs
	newVMIs := newVMIs(binding, verrazzanoUri, enableMonitoringStorage)

	// Create or update VMIs
	var vmiNames = []string{}
	for _, newVmi := range newVMIs {
		vmiNames = append(vmiNames, newVmi.Name)
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
	}

	// Delete VMIs that shouldn't exist
	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})
	existingVMIsList, err := vmiLister.VerrazzanoMonitoringInstances("").List(selector)
	if err != nil {
		return err
	}
	for _, existingVmi := range existingVMIsList {
		if !util.Contains(vmiNames, existingVmi.Name) {
			glog.V(4).Infof("Deleting VMI %s", existingVmi.Name)
			err := vmoClientSet.VerrazzanoV1().VerrazzanoMonitoringInstances(existingVmi.Namespace).Delete(context.TODO(), existingVmi.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func DeleteVmis(binding *v1beta1v8o.VerrazzanoBinding, vmoClientSet vmoclientset.Interface, vmiLister vmolisters.VerrazzanoMonitoringInstanceLister) error {

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

// Constructs the necessary VMI for the given VerrazzanoBinding
func newVMIs(binding *v1beta1v8o.VerrazzanoBinding, verrazzanoUri string, enableMonitoringStorage string) []*vmov1.VerrazzanoMonitoringInstance {
	bindingLabels := util.GetLocalBindingLabels(binding)

	if verrazzanoUri == "" {
		verrazzanoUri = "my.verrazano.com"
	}
	storageOption := createStorageOption(enableMonitoringStorage)
	return []*vmov1.VerrazzanoMonitoringInstance{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.GetVmiNameForBinding(binding.Name),
				Namespace: constants.VerrazzanoNamespace,
				Labels:    bindingLabels,
			},
			Spec: vmov1.VerrazzanoMonitoringInstanceSpec{
				URI:             util.GetVmiUri(binding.Name, verrazzanoUri),
				AutoSecret:      true,
				SecretsName:     constants.VmiSecretName,
				CascadingDelete: true,
				Grafana: vmov1.Grafana{
					Enabled:             true,
					Storage:             storageOption,
					DashboardsConfigMap: util.GetVmiNameForBinding(binding.Name) + "-dashboards",
					Resources: vmov1.Resources{
						RequestMemory: getGrafanaRequestMemory(),
					},
				},
				IngressTargetDNSName: fmt.Sprintf("verrazzano-ingress.%s", verrazzanoUri),
				Prometheus: vmov1.Prometheus{
					Enabled: true,
					Storage: storageOption,
					Resources: vmov1.Resources{
						RequestMemory: getPrometheusRequestMemory(),
					},
				},
				Elasticsearch: vmov1.Elasticsearch{
					Enabled: true,
					Storage: storageOption,
					IngestNode: vmov1.ElasticsearchNode{
						Replicas: 1,
						Resources: vmov1.Resources{
							RequestMemory: getElasticsearchIngestNodeRequestMemory(),
						},
					},
					MasterNode: vmov1.ElasticsearchNode{
						Replicas: 3,
						Resources: vmov1.Resources{
							RequestMemory: getElasticsearchMasterNodeRequestMemory(),
						},
					},
					DataNode: vmov1.ElasticsearchNode{
						Replicas: 2,
						Resources: vmov1.Resources{
							RequestMemory: getElasticsearchDataNodeRequestMemory(),
						},
					},
				},
				Kibana: vmov1.Kibana{
					Enabled: true,
					Resources: vmov1.Resources{
						RequestMemory: getKibanaRequestMemory(),
					},
				},
				ServiceType: "ClusterIP",
			},
		},
	}
}

func getElasticsearchMasterNodeRequestMemory() string {
	return "1.9Gi"
}

func getElasticsearchIngestNodeRequestMemory() string {
	return "3.4Gi"
}

func getElasticsearchDataNodeRequestMemory() string {
	return "6.5Gi"
}

func getGrafanaRequestMemory() string {
	return "48Mi"
}

func getPrometheusRequestMemory() string {
	return "72Mi"
}

func getKibanaRequestMemory() string {
	return "192Mi"
}
