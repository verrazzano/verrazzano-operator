// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"context"
	"reflect"
	"strings"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/assets"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

func UpdateConfigMaps(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	glog.V(6).Infof("Updating Local (Management Cluster) configMaps for VerrazzanoBinding %s", binding.Name)

	// Construct the set of expected configMap - this currently consists of the ConfigMap that contains the default Grafana dashboard definitions
	newConfigMap, err := newConfigMap(binding)
	if err != nil {
		return err
	}
	configMapNames := []string{newConfigMap.Name}
	existingConfigMap, err := configMapLister.ConfigMaps(newConfigMap.Namespace).Get(newConfigMap.Name)

	if existingConfigMap != nil {
		if !reflect.DeepEqual(existingConfigMap.Data, newConfigMap.Data) {
			// Don't mess with owner references or labels - VMO makes itself the "owner" of this object
			newConfigMap.OwnerReferences = existingConfigMap.OwnerReferences
			newConfigMap.Labels = existingConfigMap.Labels
			glog.V(4).Infof("Updating ConfigMap %s", newConfigMap.Name)
			_, err = kubeClientSet.CoreV1().ConfigMaps(newConfigMap.Namespace).Update(context.TODO(), newConfigMap, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// Delete ConfigMaps that shouldn't exist
	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})
	existingConfigMapsList, err := configMapLister.ConfigMaps(constants.VerrazzanoNamespace).List(selector)
	if err != nil {
		return err
	}
	for _, existingConfigMap := range existingConfigMapsList {
		if !util.Contains(configMapNames, existingConfigMap.Name) {
			glog.V(4).Infof("Deleting ConfigMap %s", existingConfigMap.Name)
			err := kubeClientSet.CoreV1().ConfigMaps(existingConfigMap.Namespace).Delete(context.TODO(), existingConfigMap.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func DeleteConfigMaps(binding *v1beta1v8o.VerrazzanoBinding, kubeClientSet kubernetes.Interface, configMapLister corev1listers.ConfigMapLister) error {
	glog.V(6).Infof("Deleting Local (Management Cluster) configMaps for VerrazzanoBinding %s", binding.Name)

	selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: binding.Name})
	existingConfigMapsList, err := configMapLister.ConfigMaps("").List(selector)
	if err != nil {
		return err
	}
	for _, existingConfigMap := range existingConfigMapsList {
		glog.V(4).Infof("Deleting configMap %s", existingConfigMap.Name)
		err := kubeClientSet.CoreV1().ConfigMaps(existingConfigMap.Namespace).Delete(context.TODO(), existingConfigMap.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// Constructs the necessary ConfigMaps for the given VerrazzanoBinding
func newConfigMap(binding *v1beta1v8o.VerrazzanoBinding) (*corev1.ConfigMap, error) {
	bindingLabels := util.GetLocalBindingLabels(binding)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetVmiNameForBinding(binding.Name) + "-dashboards",
			Namespace: constants.VerrazzanoNamespace,
			Labels:    bindingLabels,
		},
		Data: map[string]string{},
	}

	// check binding name and if system vmi, use system vmi dashboards
	var dashboards []string
	switch binding.Name {
	case constants.VmiSystemBindingName:
		// Add an entry in the ConfigMap for each of the system dashboards
		dashboards = constants.SystemDashboards
	default:
		// Add an entry in the ConfigMap for each of the default dashboards
		dashboards = constants.DefaultDashboards
	}

	for _, dashboard := range dashboards {
		content, err := assets.Asset(dashboard)
		if err != nil {
			return nil, err
		}
		configMap.Data[strings.Replace(dashboard, "/", "-", -1)] = string(content)
	}
	return configMap, nil
}
