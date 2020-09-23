// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	cohoprtypes "github.com/verrazzano/verrazzano-coh-cluster-operator/pkg/apis/verrazzano/v1beta1"
	cohclutypes "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/coherence/v1"
	domaintypes "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	listers "github.com/verrazzano/verrazzano-crd-generator/pkg/client/listers/verrazzano/v1beta1"
	cohcluinformers "github.com/verrazzano/verrazzano-crd-generator/pkg/clientcoherence/informers/externalversions"
	dominformers "github.com/verrazzano/verrazzano-crd-generator/pkg/clientwks/informers/externalversions"
	helidontypes "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/cohcluster"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	wlsoprtypes "github.com/verrazzano/verrazzano-wko-operator/pkg/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// CreateCustomResources creates/updates custome resources needed for each managed cluster.
func CreateCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, stopCh <-chan struct{}, vbLister listers.VerrazzanoBindingLister) error {

	glog.V(6).Infof("Creating/updating CustomResources for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)

	// In case of System binding, skip creating Custom resources
	if mbPair.Binding.Name == constants.VmiSystemBindingName {
		glog.V(6).Infof("Skip creating CustomResources for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)
		return nil
	}

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Construct CustomResources for each ManagedCluster
	for clusterName, mc := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Create a CR for WlsOperator?
		if mc.WlsOperator != nil {
			existingCR, err := managedClusterConnection.WlsOperatorLister.WlsOperators(mc.WlsOperator.Namespace).Get(mc.WlsOperator.Name)
			if existingCR != nil {
				// Make sure the Kind and APIVersion are set since they are not always set when returned from k8s
				existingCR.TypeMeta.Kind = "WlsOperator"
				existingCR.TypeMeta.APIVersion = "verrazzano.io/v1beta1"

				// Make a copy to be used for changes
				wlsOperatorCopy := mc.WlsOperator
				// Don't try to update the Status field, it is managed by the downstream micro-operator and we don't
				// want to overwrite it.
				wlsOperatorCopy.Status = wlsoprtypes.WlsOperatorStatus{}
				// Don't try to update the Finalizers field, it is managed by the downstream micro-operator
				wlsOperatorCopy.Finalizers = existingCR.Finalizers

				specDiffs := diff.CompareIgnoreTargetEmpties(existingCR, wlsOperatorCopy)
				if specDiffs != "" {
					glog.V(6).Infof("WlsOperator Custom Resource %s : Spec differences %s", wlsOperatorCopy.Name, specDiffs)
					glog.V(4).Infof("Updating custom resource %s:%s in cluster %s", wlsOperatorCopy.Namespace, wlsOperatorCopy.Name, clusterName)
					if len(wlsOperatorCopy.ResourceVersion) == 0 {
						wlsOperatorCopy.ResourceVersion = existingCR.ResourceVersion
					}
					_, err = managedClusterConnection.WlsOprClientSet.VerrazzanoV1beta1().WlsOperators(wlsOperatorCopy.Namespace).Update(context.TODO(), wlsOperatorCopy, metav1.UpdateOptions{})
				}

				// Retain the current status so it can be reported through the UI
				mc.WlsOperator.Status = existingCR.Status
			} else {
				glog.V(4).Infof("Creating WlsOperator custom resource %s:%s in cluster %s", mc.WlsOperator.Namespace, mc.WlsOperator.Name, clusterName)
				_, err = managedClusterConnection.WlsOprClientSet.VerrazzanoV1beta1().WlsOperators(mc.WlsOperator.Namespace).Create(context.TODO(), mc.WlsOperator, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}

			// Create the informer/lister for the wls domain now since the weblogic operator creates the
			// domains.weblogic.oracle CRD.  If we do it before creating the WLS operator then many
			// "Failed to list *v2.Domain" messages are logged.
			if managedClusterConnection.DomainLister == nil {
				// Wait up to 10 seconds for domains.weblogic.oracle CRD to be created.  If the CRD is not created within
				// this time period we will try again the next resource checking interval.
				found := false
				for i := 0; i < 10; i++ {
					_, err := managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "domains.weblogic.oracle", metav1.GetOptions{})
					if err != nil && strings.Contains(err.Error(), "\"domains.weblogic.oracle\" not found") {
						glog.V(4).Infof("Waiting for domains.weblogic.oracle CRD to be created in cluster %s", clusterName)
						time.Sleep(time.Second)
						continue
					}
					if err != nil {
						return nil
					}
					found = true
					break
				}
				if found {
					domOperatorInformerFactory := dominformers.NewSharedInformerFactory(managedClusterConnection.DomainClientSet, constants.ResyncPeriod)
					domainInformer := domOperatorInformerFactory.Weblogic().V8().Domains()
					managedClusterConnection.DomainInformer = domainInformer.Informer()
					managedClusterConnection.DomainLister = domainInformer.Lister()
					go domOperatorInformerFactory.Start(stopCh)
					// We need to sync the cache to discover existing resources if we were restarted
					if ok := cache.WaitForCacheSync(stopCh, managedClusterConnection.DomainInformer.HasSynced); !ok {
						glog.V(4).Info("Failed to wait for caches to sync")
					}
				}
			}
		}

		// Create any domain CRs?
		if mc.WlsDomainCRs != nil && managedClusterConnection.DomainLister != nil {
			for _, domainCR := range mc.WlsDomainCRs {
				existingCR, err := managedClusterConnection.DomainLister.Domains(domainCR.Namespace).Get(domainCR.Name)
				if existingCR != nil {
					// Don't try to update the Status field, it ia managed by the downstream operator and we don't
					// want to overwrite it.
					domainCR.Status = domaintypes.DomainStatus{}
					specDiffs := diff.CompareIgnoreTargetEmpties(existingCR, domainCR)
					// Explicitly check the replicas which handles the case when target replicas are changed to zero or omitted
					if specDiffs != "" || !isReplicasEqual(existingCR.Spec.Replicas, domainCR.Spec.Replicas) {
						if specDiffs != "" {
							glog.V(6).Infof("Domain Custom Resource %s : Spec differences %s", domainCR.Name, specDiffs)
						}
						glog.V(4).Infof("Patching Domain custom resource %s:%s in cluster %s", domainCR.Namespace, domainCR.Name, clusterName)
						domainCRjson, err := json.Marshal(domainCR)
						if err != nil {
							return err
						}
						_, err = managedClusterConnection.DomainClientSet.WeblogicV8().Domains(domainCR.Namespace).Patch(context.TODO(), domainCR.Name, k8sTypes.MergePatchType, domainCRjson, metav1.PatchOptions{})
						if err != nil {
							return err
						}
					}

					// Retain the current status so it can be reported through the UI
					domainCR.Status = existingCR.Status
				} else {
					glog.V(4).Infof("Creating Domain custom resource %s:%s in cluster %s", domainCR.Namespace, domainCR.Name, clusterName)
					_, err = managedClusterConnection.DomainClientSet.WeblogicV8().Domains(domainCR.Namespace).Create(context.TODO(), domainCR, metav1.CreateOptions{})
				}
				if err != nil {
					return err
				}
			}
		}

		// Create any Coherence operator CRs?
		// A Coherence operator is created for each namespace that needs a coherence cluster
		if mc.CohOperatorCRs != nil {
			for _, operatorCR := range mc.CohOperatorCRs {

				existingCR, err := managedClusterConnection.CohOperatorLister.CohClusters(operatorCR.Namespace).Get(operatorCR.Name)
				if existingCR != nil {
					// Make sure the Kind and APIVersion are set since they are not always set when returned from k8s
					existingCR.TypeMeta.Kind = "CohCluster"
					existingCR.TypeMeta.APIVersion = "verrazzano.io/v1beta1"

					// Don't try to update the Status field, it ia managed by the downstream operator and we don't
					// want to overwrite it.
					operatorCR.Status = cohoprtypes.CohClusterStatus{}
					// Don't try to update the Finalizers field, it is managed by the downstream micro-operator
					operatorCR.Finalizers = existingCR.Finalizers

					specDiffs := diff.CompareIgnoreTargetEmpties(existingCR, operatorCR)
					if specDiffs != "" {
						glog.V(6).Infof("CohCluster custom resource %s : Spec differences %s", operatorCR.Name, specDiffs)
						glog.V(4).Infof("Updating CohCluster custom resource %s:%s in cluster %s", operatorCR.Namespace, operatorCR.Name, clusterName)
						if len(operatorCR.ResourceVersion) == 0 {
							operatorCR.ResourceVersion = existingCR.ResourceVersion
						}
						_, err = managedClusterConnection.CohOprClientSet.VerrazzanoV1beta1().CohClusters(operatorCR.Namespace).Update(context.TODO(), operatorCR, metav1.UpdateOptions{})
					}
				} else {
					glog.V(4).Infof("Creating CohCluster custom resource %s:%s in cluster %s", operatorCR.Namespace, operatorCR.Name, clusterName)
					_, err = managedClusterConnection.CohOprClientSet.VerrazzanoV1beta1().CohClusters(operatorCR.Namespace).Create(context.TODO(), operatorCR, metav1.CreateOptions{})
				}
				if err != nil {
					return err
				}
			}

			// Create the informer/lister for Coherence cluster now since the Coherence operator creates the
			// Coherence CRDs.  If we do it before creating the Coherence operator then many
			// "Failed to list *v1.CoherenceCluster" messages are logged.
			if managedClusterConnection.CohClusterLister == nil {
				// Wait up to 10 seconds for Coherence CRDs to be created.  If the CRD is not created within
				// this time period we will try again the next resource checking interval.
				crd1 := false
				crd2 := false
				crd3 := false
				for i := 0; i < 10; i++ {
					if !crd1 {
						_, err := managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coherenceclusters.coherence.oracle.com", metav1.GetOptions{})
						if err != nil && strings.Contains(err.Error(), "\"coherenceclusters.coherence.oracle.com\" not found") {
							glog.V(4).Infof("Waiting for coherenceclusters.coherence.oracle.com CRD to be created in cluster %s", clusterName)
							time.Sleep(time.Second)
							continue
						}
						if err != nil {
							return nil
						}
						crd1 = true
					}
					if !crd2 {
						_, err := managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coherenceroles.coherence.oracle.com", metav1.GetOptions{})
						if err != nil && strings.Contains(err.Error(), "\"coherenceroles.coherence.oracle.com\" not found") {
							glog.V(4).Infof("Waiting for coherenceroles.coherence.oracle.com CRD to be created in cluster %s", clusterName)
							time.Sleep(time.Second)
							continue
						}
						if err != nil {
							return nil
						}
						crd2 = true
					}
					_, err := managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coherenceinternals.coherence.oracle.com", metav1.GetOptions{})
					if err != nil && strings.Contains(err.Error(), "\"coherenceinternals.coherence.oracle.com\" not found") {
						glog.V(4).Infof("Waiting for coherenceinternals.coherence.oracle.com CRD to be created in cluster %s", clusterName)
						time.Sleep(time.Second)
						continue
					}
					if err != nil {
						return nil
					}
					crd3 = true
					break
				}

				if crd1 && crd2 && crd3 {
					cohInformerFactory := cohcluinformers.NewSharedInformerFactory(managedClusterConnection.CohClusterClientSet, constants.ResyncPeriod)
					cohInformer := cohInformerFactory.Coherence().V1().CoherenceClusters()
					managedClusterConnection.CohClusterInformer = cohInformer.Informer()
					managedClusterConnection.CohClusterLister = cohInformer.Lister()
					go cohInformerFactory.Start(stopCh)
					// We need to sync the cache to discover existing resources if we were restarted
					if ok := cache.WaitForCacheSync(stopCh, managedClusterConnection.CohClusterInformer.HasSynced); !ok {
						glog.V(4).Info("Failed to wait for caches to sync")
					}
				}
			}
		}

		// Create any Coherence cluster CRs?
		if mc.CohClusterCRs != nil && managedClusterConnection.CohClusterLister != nil {
			for _, clusterCR := range mc.CohClusterCRs {
				existingCR, err := managedClusterConnection.CohClusterLister.CoherenceClusters(clusterCR.Namespace).Get(clusterCR.Name)
				if existingCR != nil {
					// Make sure the Kind and APIVersion are set since they are not always set when returned from k8s
					existingCR.TypeMeta.Kind = cohcluster.Kind
					existingCR.TypeMeta.APIVersion = cohcluster.ApiVersion

					// Don't try to update the Status field, it ia managed by the downstream operator and we don't
					// want to overwrite it.
					clusterCR.Status = cohclutypes.CoherenceClusterStatus{}

					specDiffs := diff.CompareIgnoreTargetEmpties(existingCR, clusterCR)
					// Explicitly check the replicas which handles the case when target replicas are changed to zero or omitted
					if specDiffs != "" || !isReplicasEqual(existingCR.Spec.Replicas, clusterCR.Spec.Replicas) {
						if specDiffs != "" {
							glog.V(6).Infof("CoherenceCluster custom resource %s : Spec differences %s", clusterCR.Name, specDiffs)
						}
						glog.V(4).Infof("Updating CoherenceCluster custom resource %s:%s in cluster %s", clusterCR.Namespace, clusterCR.Name, clusterName)
						// resourceVersion field cannot be empty on an update
						if len(clusterCR.ResourceVersion) == 0 {
							clusterCR.ResourceVersion = existingCR.ResourceVersion
						}
						_, err = managedClusterConnection.CohClusterClientSet.CoherenceV1().CoherenceClusters(clusterCR.Namespace).Update(context.TODO(), clusterCR, metav1.UpdateOptions{})
					}

					// Retain the current status so it can be reported through the UI
					clusterCR.Status = existingCR.Status
				} else {
					glog.V(4).Infof("Creating CoherenceCluster custom resource %s:%s in cluster %s", clusterCR.Namespace, clusterCR.Name, clusterName)
					_, err = managedClusterConnection.CohClusterClientSet.CoherenceV1().CoherenceClusters(clusterCR.Namespace).Create(context.TODO(), clusterCR, metav1.CreateOptions{})
				}
				if err != nil {
					return err
				}
			}
		}

		// Create any Helidon App CRs?
		if mc.HelidonApps != nil {
			for _, helidonCR := range mc.HelidonApps {
				existingCR, err := managedClusterConnection.HelidonLister.HelidonApps(helidonCR.Namespace).Get(helidonCR.Name)
				if existingCR != nil {
					// Make sure the Kind and APIVersion are set since they are not always set when returned from k8s
					existingCR.TypeMeta.Kind = "HelidonApp"
					existingCR.TypeMeta.APIVersion = "verrazzano.io/v1beta1"

					// Don't try to update the Status field, it ia managed by the downstream operator and we don't
					// want to overwrite it.
					helidonCR.Status = helidontypes.HelidonAppStatus{}

					specDiffs := diff.CompareIgnoreTargetEmpties(existingCR, helidonCR)
					// Explicitly check the cases where target resources are changed to zero or omitted
					if specDiffs != "" || !isReplicasEqual(existingCR.Spec.Replicas, helidonCR.Spec.Replicas) ||
						!isContainersEqual(existingCR.Spec.Containers, helidonCR.Spec.Containers) ||
						!isVolumesEqual(existingCR.Spec.Volumes, helidonCR.Spec.Volumes) {
						if specDiffs != "" {
							glog.V(6).Infof("HelidonApp Custom Resource %s : Spec differences %s", helidonCR.Name, specDiffs)
						}
						glog.V(4).Infof("Updating HelidonApp custom resource %s:%s in cluster %s", helidonCR.Namespace, helidonCR.Name, clusterName)
						// resourceVersion field cannot be empty on an update
						if len(helidonCR.ResourceVersion) == 0 {
							helidonCR.ResourceVersion = existingCR.ResourceVersion
						}
						_, err = managedClusterConnection.HelidonClientSet.VerrazzanoV1beta1().HelidonApps(helidonCR.Namespace).Update(context.TODO(), helidonCR, metav1.UpdateOptions{})

					}

					// Retain the current status so it can be reported through the UI
					helidonCR.Status = existingCR.Status
				} else {
					glog.V(4).Infof("Creating HelidonApp custom resource %s:%s in cluster %s", helidonCR.Namespace, helidonCR.Name, clusterName)
					_, err = managedClusterConnection.HelidonClientSet.VerrazzanoV1beta1().HelidonApps(helidonCR.Namespace).Create(context.TODO(), helidonCR, metav1.CreateOptions{})
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// DeleteCustomResources deletes custom resources for a given binding.
func DeleteCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	glog.V(6).Infof("Deleting ServiceAccount for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	managedClusters, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	// Delete custom resources associated with the given VerrazzanoApplicationBinding (based on labels selectors)
	for clusterName, mc := range managedClusters {
		mc.Lock.RLock()
		defer mc.Lock.RUnlock()

		// Domain Custom Resources
		// Don't use domain lister since during a restart of the operator the lister may not be available.
		_, err = mc.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "domains.weblogic.oracle", metav1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil
		}

		if err == nil {
			existingDomainList, err := mc.DomainClientSet.WeblogicV8().Domains("").List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: constants.VerrazzanoBinding + "=" + mbPair.Binding.Name})
			if err != nil {
				return err
			}
			for _, domain := range existingDomainList.Items {
				glog.V(4).Infof("Deleting Domain custom resource %s:%s in cluster %s", domain.Namespace, domain.Name, clusterName)
				err = mc.DomainClientSet.WeblogicV8().Domains(domain.Namespace).Delete(context.TODO(), domain.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForCRDeletion(mc, "domain", domain.Name, domain.Namespace, clusterName, 2*time.Minute, 2*time.Second)
				if err != nil {
					return err
				}
			}
		}

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name})

		// WlsOperator Custom Resources
		existingWlsOprList, err := mc.WlsOperatorLister.WlsOperators("").List(selector)
		if err != nil {
			return err
		}
		for _, wlsopr := range existingWlsOprList {
			glog.V(4).Infof("Deleting WlsOperator custom resource %s:%s in cluster %s", wlsopr.Namespace, wlsopr.Name, clusterName)
			err = mc.WlsOprClientSet.VerrazzanoV1beta1().WlsOperators(wlsopr.Namespace).Delete(context.TODO(), wlsopr.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			err = waitForCRDeletion(mc, "wlsoperator", wlsopr.Name, wlsopr.Namespace, clusterName, 1*time.Minute, 5*time.Second)
			if err != nil {
				return err
			}
		}

		// Helidon App Custom Resources
		existingAppList, err := mc.HelidonLister.HelidonApps("").List(selector)
		if err != nil {
			return err
		}
		for _, app := range existingAppList {
			glog.V(4).Infof("Deleting HelidonApp custom resource %s:%s in cluster %s", app.Namespace, app.Name, clusterName)
			err = mc.HelidonClientSet.VerrazzanoV1beta1().HelidonApps(app.Namespace).Delete(context.TODO(), app.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			err = waitForCRDeletion(mc, "helidonapp", app.Name, app.Namespace, clusterName, 1*time.Minute, 5*time.Second)
			if err != nil {
				return err
			}
		}

		// Coherence cluster Custom Resources
		// Don't use Coherence cluster lister since during a restart of the operator the lister may not be available.
		_, err = mc.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coherenceclusters.coherence.oracle.com", metav1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil
		}

		if err == nil {
			existingClusterList, err := mc.CohClusterClientSet.CoherenceV1().CoherenceClusters("").List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: constants.VerrazzanoBinding + "=" + mbPair.Binding.Name})
			if err != nil {
				return err
			}
			for _, cluster := range existingClusterList.Items {
				glog.V(4).Infof("Deleting CoherenceCluster custom resource %s:%s in cluster %s", cluster.Namespace, cluster.Name, clusterName)
				err = mc.CohClusterClientSet.CoherenceV1().CoherenceClusters(cluster.Namespace).Delete(context.TODO(), cluster.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = cleanupCoherenceCustomResources(mc, cluster.Name, cluster.Namespace, clusterName)
				if err != nil {
					return err
				}
			}
		}

		// Coherence operator Custom Resources
		existingCohList, err := mc.CohOperatorLister.CohClusters("").List(selector)
		if err != nil {
			return err
		}
		for _, cohCluster := range existingCohList {
			glog.V(4).Infof("Deleting CohCluster custom resource %s:%s in cluster %s", cohCluster.Namespace, cohCluster.Name, clusterName)
			err = mc.CohOprClientSet.VerrazzanoV1beta1().CohClusters(cohCluster.Namespace).Delete(context.TODO(), cohCluster.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			err = waitForCRDeletion(mc, "cohcluster", cohCluster.Name, cohCluster.Namespace, clusterName, 1*time.Minute, 5*time.Second)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func cleanupCoherenceCustomResources(mc *util.ManagedClusterConnection, name string, namespace string, clusterName string) error {

	// First, delete the coherencecluster resource
	err := waitForCRDeletion(mc, "coherencecluster", name, namespace, clusterName, 1*time.Minute, 5*time.Second)
	if err != nil {
		return err
	}

	// Let make sure the coherenceinternals resource is also gone. This resource is created as a result of creating the coherencecluster
	// resource.  This resource must be deleted before deleting the coherence operator otherwise we could have a coherenceinternals resource
	// with a finalizer that has not been run.
	coherenceInternalsName := name + "-storage"
	glog.V(4).Infof("Deleting CoherenceInternals custom resource %s:%s in cluster %s", namespace, coherenceInternalsName, clusterName)
	err = waitForCRDeletion(mc, "coherenceinternals", coherenceInternalsName, namespace, clusterName, 1*time.Minute, 2*time.Second)
	if err != nil {
		return err
	}

	return nil
}

func containsHelidonApp(apps []*helidontypes.HelidonApp, inName string, inNamespace string) bool {
	for _, app := range apps {
		if inName == app.Name && inNamespace == app.Namespace {
			return true
		}
	}
	return false
}

func containsCohClusterCRs(clusters []*cohclutypes.CoherenceCluster, inName string, inNamespace string) bool {
	for _, clu := range clusters {
		if inName == clu.Name && inNamespace == clu.Namespace {
			return true
		}
	}
	return false
}

func containsCohOperatorCRs(oprs []*cohoprtypes.CohCluster, inName string, inNamespace string) bool {
	for _, opr := range oprs {
		if inName == opr.Name && inNamespace == opr.Namespace {
			return true
		}
	}
	return false
}

func containsWlsDomainCRs(domains []*domaintypes.Domain, inName string, inNamespace string) bool {
	for _, dom := range domains {
		if inName == dom.Name && inNamespace == dom.Namespace {
			return true
		}
	}
	return false
}

func containsWlsOperator(opr *wlsoprtypes.WlsOperator, inName string, inNamespace string) bool {
	if opr != nil && inName == opr.Name && inNamespace == opr.Namespace {
		return true
	}
	return false
}

// CleanupOrphanedCustomResources deletes custom resources that have been orphaned.
func CleanupOrphanedCustomResources(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, stopCh <-chan struct{}) error {
	glog.V(6).Infof("Cleaning up orphaned CustomResources for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)

	// Get the managed clusters that this binding applies to
	matchedClusters, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return nil
	}

	for clusterName, mc := range mbPair.ManagedClusters {
		managedClusterConnection := matchedClusters[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// Make sure the cache is up-to-date before we get the resource list
		if ok := cache.WaitForCacheSync(stopCh, managedClusterConnection.HelidonInformer.HasSynced); !ok {
			glog.V(4).Info("Failed to wait for caches to sync")
		}

		// Get list of Helidon apps for this cluster and given binding
		existingCRList, err := managedClusterConnection.HelidonLister.HelidonApps("").List(selector)
		if err != nil {
			return err
		}

		// Delete any Helidon apps not expected on this cluster
		for _, app := range existingCRList {
			if !containsHelidonApp(mc.HelidonApps, app.Name, app.Namespace) {
				glog.V(4).Infof("Deleting HelidonApp custom resource %s:%s in cluster %s", app.Namespace, app.Name, clusterName)
				err := managedClusterConnection.HelidonClientSet.VerrazzanoV1beta1().HelidonApps(app.Namespace).Delete(context.TODO(), app.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForCRDeletion(managedClusterConnection, "helidonapp", app.Name, app.Namespace, clusterName, 1*time.Minute, 5*time.Second)
				if err != nil {
					return err
				}
			}
		}

		// Get list of Coherence Clusters for this cluster and given binding.
		// Don't use Coherence cluster lister since during a restart of the operator the lister may not be available.
		_, err = managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coherenceclusters.coherence.oracle.com", metav1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil
		}

		if err == nil {
			existingClusterList, err := managedClusterConnection.CohClusterClientSet.CoherenceV1().CoherenceClusters("").List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: constants.VerrazzanoBinding + "=" + mbPair.Binding.Name})
			if err != nil {
				return err
			}

			// Delete any Coherence Clusters not expected on this cluster
			for _, cohCluster := range existingClusterList.Items {
				if !containsCohClusterCRs(mc.CohClusterCRs, cohCluster.Name, cohCluster.Namespace) {
					glog.V(4).Infof("Deleting CoherenceCluster custom resource %s:%s in cluster %s", cohCluster.Namespace, cohCluster.Name, clusterName)
					err := managedClusterConnection.CohClusterClientSet.CoherenceV1().CoherenceClusters(cohCluster.Namespace).Delete(context.TODO(), cohCluster.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}

					err = cleanupCoherenceCustomResources(managedClusterConnection, cohCluster.Name, cohCluster.Namespace, clusterName)
					if err != nil {
						return err
					}
				}
			}
		}

		// Make sure the cache is up-to-date before we get the resource list
		if ok := cache.WaitForCacheSync(stopCh, managedClusterConnection.CohOperatorInformer.HasSynced); !ok {
			glog.V(4).Info("Failed to wait for caches to sync")
		}

		// Get list of Coherence Operators for this cluster and given binding
		existingCohClustersCRList, err := managedClusterConnection.CohOperatorLister.CohClusters("").List(selector)
		if err != nil {
			return err
		}

		// Delete any Coherence Operators not expected on this cluster
		for _, cohOperator := range existingCohClustersCRList {
			if !containsCohOperatorCRs(mc.CohOperatorCRs, cohOperator.Name, cohOperator.Namespace) {
				glog.V(4).Infof("Deleting CohCluster custom resource %s:%s on cluster %s", cohOperator.Namespace, cohOperator.Name, clusterName)
				err := managedClusterConnection.CohOprClientSet.VerrazzanoV1beta1().CohClusters(cohOperator.Namespace).Delete(context.TODO(), cohOperator.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForCRDeletion(managedClusterConnection, "cohcluster", cohOperator.Name, cohOperator.Namespace, clusterName, 1*time.Minute, 5*time.Second)
				if err != nil {
					return err
				}
			}
		}

		// Get list of WLS Domains for this cluster and given binding.
		// Don't use domain lister since during a restart of the operator the lister may not be available.
		_, err = managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "domains.weblogic.oracle", metav1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil
		}

		if err == nil {
			existingDomainList, err := managedClusterConnection.DomainClientSet.WeblogicV8().Domains("").List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: constants.VerrazzanoBinding + "=" + mbPair.Binding.Name})
			if err != nil {
				return err
			}

			// Delete any WLS Domains not expected on this cluster
			for _, wlsDomain := range existingDomainList.Items {
				if !containsWlsDomainCRs(mc.WlsDomainCRs, wlsDomain.Name, wlsDomain.Namespace) {
					glog.V(4).Infof("Deleting Domain custom resource %s:%s in cluster %s", wlsDomain.Namespace, wlsDomain.Name, clusterName)
					err := managedClusterConnection.DomainClientSet.WeblogicV8().Domains(wlsDomain.Namespace).Delete(context.TODO(), wlsDomain.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}

					err = waitForCRDeletion(managedClusterConnection, "domain", wlsDomain.Name, wlsDomain.Namespace, clusterName, 2*time.Minute, 2*time.Second)
					if err != nil {
						return err
					}
				}
			}
		}

		// Make sure the cache is up-to-date before we get the resource list
		if ok := cache.WaitForCacheSync(stopCh, managedClusterConnection.WlsOperatorInformer.HasSynced); !ok {
			glog.V(4).Info("Failed to wait for caches to sync")
		}

		// Get list of WLS Operators for this cluster and given binding
		existingWlsOperatorCRList, err := managedClusterConnection.WlsOperatorLister.WlsOperators("").List(selector)
		if err != nil {
			return err
		}

		// Delete any WLS Operators not expected on this cluster
		for _, wlsOperator := range existingWlsOperatorCRList {
			if !containsWlsOperator(mc.WlsOperator, wlsOperator.Name, wlsOperator.Namespace) {
				glog.V(4).Infof("Deleting WLSOperator custom resource %s:%s in cluster %s", wlsOperator.Namespace, wlsOperator.Name, clusterName)
				err := managedClusterConnection.WlsOprClientSet.VerrazzanoV1beta1().WlsOperators(wlsOperator.Namespace).Delete(context.TODO(), wlsOperator.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForCRDeletion(managedClusterConnection, "wlsoperator", wlsOperator.Name, wlsOperator.Namespace, clusterName, 1*time.Minute, 5*time.Second)
				if err != nil {
					return err
				}
			}
		}
	}

	// Get the managed clusters that this binding does NOT apply to
	unmatchedClusters := util.GetManagedClustersNotForVerrazzanoBinding(mbPair, availableManagedClusterConnections)

	for clusterName, managedClusterConnection := range unmatchedClusters {
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{constants.VerrazzanoBinding: mbPair.Binding.Name, constants.VerrazzanoCluster: clusterName})

		// Get list of Helidon apps for this cluster and given binding
		appList, err := managedClusterConnection.HelidonLister.HelidonApps("").List(selector)
		if err != nil {
			return err
		}
		// Delete these Helidon apps since none are expected on this cluster
		for _, app := range appList {
			glog.V(4).Infof("Deleting HelidonApp custom resource %s:%s in cluster %s", app.Namespace, app.Name, clusterName)
			err := managedClusterConnection.HelidonClientSet.VerrazzanoV1beta1().HelidonApps(app.Namespace).Delete(context.TODO(), app.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			err = waitForCRDeletion(managedClusterConnection, "helidonapp", app.Name, app.Namespace, clusterName, 1*time.Minute, 5*time.Second)
			if err != nil {
				return err
			}
		}

		// Get list of Coherence clusters for this cluster and given binding
		// Don't use Coherence cluster lister since during a restart of the operator the lister may not be available.
		_, err = managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "coherenceclusters.coherence.oracle.com", metav1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil
		}

		if err == nil {
			clusterList, err := managedClusterConnection.CohClusterClientSet.CoherenceV1().CoherenceClusters("").List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: constants.VerrazzanoBinding + "=" + mbPair.Binding.Name})
			if err != nil {
				return err
			}

			// Delete these Coherence clusters since none are expected on this cluster
			for _, cluster := range clusterList.Items {
				glog.V(4).Infof("Deleting CoherenceCluster custom resource %s:%s in cluster %s", cluster.Namespace, cluster.Name, clusterName)
				err := managedClusterConnection.CohClusterClientSet.CoherenceV1().CoherenceClusters(cluster.Namespace).Delete(context.TODO(), cluster.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = cleanupCoherenceCustomResources(managedClusterConnection, cluster.Name, cluster.Namespace, clusterName)
				if err != nil {
					return err
				}
			}
		}

		// Get list of Coherence operators for this cluster and given binding
		cohList, err := managedClusterConnection.CohOperatorLister.CohClusters("").List(selector)
		if err != nil {
			return err
		}
		// Delete these Coherence operators since none are expected on this cluster
		for _, coh := range cohList {
			glog.V(4).Infof("Deleting CohCluster custom resource %s:%s in cluster %s", coh.Namespace, coh.Name, clusterName)
			err := managedClusterConnection.CohOprClientSet.VerrazzanoV1beta1().CohClusters(coh.Namespace).Delete(context.TODO(), coh.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			err = waitForCRDeletion(managedClusterConnection, "cohcluster", coh.Name, coh.Namespace, clusterName, 1*time.Minute, 5*time.Second)
			if err != nil {
				return err
			}
		}

		// Get list of WLS Domains for this cluster and given binding
		// Don't use domain lister since during a restart of the operator the lister may not be available.
		_, err = managedClusterConnection.KubeExtClientSet.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "domains.weblogic.oracle", metav1.GetOptions{})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil
		}

		if err == nil {
			domainList, err := managedClusterConnection.DomainClientSet.WeblogicV8().Domains("").List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: constants.VerrazzanoBinding + "=" + mbPair.Binding.Name})
			if err != nil {
				return err
			}

			// Delete these WLS Domains since none are expected on this cluster
			for _, domain := range domainList.Items {
				glog.V(4).Infof("Deleting Domain custom resource %s:%s in cluster %s", domain.Namespace, domain.Name, clusterName)
				err := managedClusterConnection.DomainClientSet.WeblogicV8().Domains(domain.Namespace).Delete(context.TODO(), domain.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				err = waitForCRDeletion(managedClusterConnection, "domain", domain.Name, domain.Namespace, clusterName, 2*time.Minute, 2*time.Second)
				if err != nil {
					return err
				}
			}
		}

		// Get list of WLS Operators for this cluster and given binding
		operatorList, err := managedClusterConnection.WlsOperatorLister.WlsOperators("").List(selector)
		if err != nil {
			return err
		}
		// Delete these WLS Operators since none are expected on this cluster
		for _, operator := range operatorList {
			glog.V(4).Infof("Deleting WLSOperator custom resource %s:%s in cluster %s", operator.Namespace, operator.Name, clusterName)
			err := managedClusterConnection.WlsOprClientSet.VerrazzanoV1beta1().WlsOperators(operator.Namespace).Delete(context.TODO(), operator.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			err = waitForCRDeletion(managedClusterConnection, "wlsoperator", operator.Name, operator.Namespace, clusterName, 1*time.Minute, 5*time.Second)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func waitForCRDeletion(mc *util.ManagedClusterConnection, crType string, crName string, namespace string, cluster string, timeoutDuration time.Duration, tickDuration time.Duration) error {
	timeout := time.After(timeoutDuration)
	tick := time.Tick(tickDuration)
	var err error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for %s %s to be removed in namespace %s in managed cluster %s", crType, crName, namespace, cluster)
		case <-tick:
			glog.V(4).Infof("Waiting for %s %s in namespace %s in managed cluster %s to be removed..", crType, crName, namespace, cluster)
			switch crType {
			case "domain":
				_, err = mc.DomainClientSet.WeblogicV8().Domains(namespace).Get(context.TODO(), crName, metav1.GetOptions{})
			case "wlsoperator":
				_, err = mc.WlsOprClientSet.VerrazzanoV1beta1().WlsOperators(namespace).Get(context.TODO(), crName, metav1.GetOptions{})
			case "cohcluster":
				_, err = mc.CohOprClientSet.VerrazzanoV1beta1().CohClusters(namespace).Get(context.TODO(), crName, metav1.GetOptions{})
			case "coherencecluster":
				_, err = mc.CohClusterClientSet.CoherenceV1().CoherenceClusters(namespace).Get(context.TODO(), crName, metav1.GetOptions{})
			case "coherenceinternals":
				_, err = mc.CohClusterClientSet.CoherenceV1().CoherenceInternals(namespace).Get(context.TODO(), crName, metav1.GetOptions{})
			case "helidonapp":
				_, err = mc.HelidonClientSet.VerrazzanoV1beta1().HelidonApps(namespace).Get(context.TODO(), crName, metav1.GetOptions{})
			default:
				err = errors.New("Unknown CR type")
			}
			if err != nil && k8sErrors.IsNotFound(err) {
				glog.V(4).Infof("Removed %s %s in namespace %s in managed cluster %s ..", crType, crName, namespace, cluster)
				return nil
			}

			if err != nil {
				return fmt.Errorf("Error removing %s %s in namespace %s in managed cluster %s, error %s", crType, crName, namespace, cluster, err.Error())
			}
		}
	}
}

// Check if the existing and target containers are same
func isContainersEqual(existing []corev1.Container, target []corev1.Container) bool {
	if len(existing) != len(target) {
		return false
	}

	for i := range existing {
		if !reflect.DeepEqual(existing[i], target[i]) {
			return false
		}
	}
	return true
}

// Check if the existing and target volumes are same
func isVolumesEqual(existing []corev1.Volume, target []corev1.Volume) bool {
	if len(existing) != len(target) {
		return false
	}

	for i := range existing {
		if !reflect.DeepEqual(existing[i], target[i]) {
			return false
		}
	}
	return true
}

// Check if the existing and target replicas are same
func isReplicasEqual(existing *int32, target *int32) bool {
	if existing == nil && target == nil {
		return true
	}

	if existing == nil && target != nil {
		return false
	}

	if existing != nil && target == nil {
		return false
	}

	if *existing == *target {
		return true
	}

	return false
}
