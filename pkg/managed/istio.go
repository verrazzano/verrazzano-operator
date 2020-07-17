// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package managed

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/rs/zerolog"
	istiocrd "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/networking.istio.io/v1alpha3"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/diff"
	istio "istio.io/api/networking/v1alpha3"
	istiosecv1beta1 "istio.io/api/security/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const PrometheusPodPrefix  = "prometheus-"
const IstioSystemNamespace = "istio-system"

func CreateIngresses(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating ingresses
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "Ingresses").Str("name", "ClusterConnection").Logger()

	logger.Debug().Msgf("Creating/updating Ingresses for VerrazzanoBinding %s", mbPair.Binding.Name)

	// In case of System binding, skip creating ingresses
	if mbPair.Binding.Name == constants.VmiSystemBindingName {
		logger.Debug().Msgf("Skip creating Ingresses for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)
		return nil
	}

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct ingresses for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct the set of expected ingresses (gateway and virtual service objects returned)
		newGateways, newVirtualServices := newIngresses(mbPair.Binding, managedClusterObj)

		// Create or update ingresses
		for _, newGateway := range newGateways {
			existingGateway, err := managedClusterConnection.IstioGatewayLister.Gateways(newGateway.Namespace).Get(newGateway.Name)
			if existingGateway != nil {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingGateway, newGateway)
				if specDiffs != "" {
					logger.Debug().Msgf("Istio Gateway %s : Spec differences %s", newGateway.Name, specDiffs)
					logger.Info().Msgf("Updating Istio Gateway %s:%s in cluster %s", newGateway.Namespace, newGateway.Name, clusterName)
					if len(newGateway.ResourceVersion) == 0 {
						newGateway.ResourceVersion = existingGateway.ResourceVersion
					}
					_, err = managedClusterConnection.IstioClientSet.NetworkingV1alpha3().Gateways(newGateway.Namespace).Update(context.TODO(), newGateway, metav1.UpdateOptions{})
				}
			} else {
				logger.Info().Msgf("Creating Istio Gateway %s:%s in cluster %s", newGateway.Namespace, newGateway.Name, clusterName)
				_, err = managedClusterConnection.IstioClientSet.NetworkingV1alpha3().Gateways(newGateway.Namespace).Create(context.TODO(), newGateway, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}

		// Create or update ingresses
		for _, newVirtualService := range newVirtualServices {
			existingVirtualService, err := managedClusterConnection.IstioVirtualServiceLister.VirtualServices(newVirtualService.Namespace).Get(newVirtualService.Name)
			if existingVirtualService != nil {
				specDiffs := diff.CompareIgnoreTargetEmpties(existingVirtualService, newVirtualService)
				if specDiffs != "" {
					logger.Debug().Msgf("Istio VirtualService %s : Spec differences %s", newVirtualService.Name, specDiffs)
					logger.Info().Msgf("Updating Istio VirtualService %s:%s in cluster %s", newVirtualService.Namespace, newVirtualService.Name, clusterName)
					if len(newVirtualService.ResourceVersion) == 0 {
						newVirtualService.ResourceVersion = existingVirtualService.ResourceVersion
					}
					_, err = managedClusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices(newVirtualService.Namespace).Update(context.TODO(), newVirtualService, metav1.UpdateOptions{})
				}
			} else {
				logger.Info().Msgf("Creating Istio VirtualService %s:%s in cluster %s", newVirtualService.Namespace, newVirtualService.Name, clusterName)
				_, err = managedClusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices(newVirtualService.Namespace).Create(context.TODO(), newVirtualService, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateServiceEntries(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for returning creating service entries
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ServiceEntries").Str("name", "ClusterConnection").Logger()

	logger.Debug().Msgf("Creating/updating istio serviceentries for VerrazzanoBinding %s", mbPair.Binding.Name)

	// In case of System binding, skip creating Service Entries
	if mbPair.Binding.Name == constants.VmiSystemBindingName {
		logger.Debug().Msgf("Skip creating Service Entries for VerrazzanoApplicationBinding %s", mbPair.Binding.Name)
		return nil
	}

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct istio serviceentries for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		// Construct the set of expected isto serviceentries
		newServiceEntries := newServiceEntries(mbPair, managedClusterObj, availableManagedClusterConnections)

		// Create or update service entries
		var startIPIndex = 1
		for _, newServiceEntry := range newServiceEntries {
			existingServiceEntry, err := managedClusterConnection.IstioServiceEntryLister.ServiceEntries(newServiceEntry.Namespace).Get(newServiceEntry.Name)
			if existingServiceEntry != nil {
				if existingServiceEntry.Spec.Addresses == nil {
					newServiceEntry.Spec.Addresses = make([]string, 1)
					newServiceEntry.Spec.Addresses[0], err = getUniqueServiceEntryAddress(managedClusterConnection, &startIPIndex)
					if err != nil {
						return err
					}
				} else {
					// Make sure the address is set to prevent a diff
					newServiceEntry.Spec.Addresses = existingServiceEntry.Spec.Addresses
				}
				specDiffs := diff.CompareIgnoreTargetEmpties(existingServiceEntry, newServiceEntry)
				if specDiffs != "" {
					logger.Debug().Msgf("Istio ServiceEntry %s : Spec differences %s", newServiceEntry.Name, specDiffs)
					logger.Info().Msgf("Updating Istio ServiceEntry %s:%s in cluster %s", newServiceEntry.Namespace, newServiceEntry.Name, clusterName)
					// resourceVersion field cannot be empty on an update
					if len(newServiceEntry.ResourceVersion) == 0 {
						newServiceEntry.ResourceVersion = existingServiceEntry.ResourceVersion
					}
					_, err = managedClusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries(newServiceEntry.Namespace).Update(context.TODO(), newServiceEntry, metav1.UpdateOptions{})
				}
			} else {
				logger.Info().Msgf("Creating Istio ServiceEntry %s:%s in cluster %s", newServiceEntry.Namespace, newServiceEntry.Name, clusterName)
				newServiceEntry.Spec.Addresses = make([]string, 1)
				newServiceEntry.Spec.Addresses[0], err = getUniqueServiceEntryAddress(managedClusterConnection, &startIPIndex)
				if err != nil {
					return err
				}

				logger.Info().Msg(fmt.Sprintf("%+v", newServiceEntry))
				_, err = managedClusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries(newServiceEntry.Namespace).Create(context.TODO(), newServiceEntry, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateDestinationRules(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating destination rules
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "DestinationRules").Str("name", "ClusterConnection").Logger()

	logger.Debug().Msgf("Creating/updating DestinationRules for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct istio destination rules for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{})
		pods, err := managedClusterConnection.PodLister.List(selector)
		if err != nil {
			return err
		}

		// Construct the set of expected isto destination rules
		newRules, err := newDestinationRules(mbPair, managedClusterObj, pods)
		if err != nil {
			return err
		}

		for _, newRule := range newRules {
			existingRule, err := managedClusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules(newRule.Namespace).Get(context.TODO(), newRule.Name, metav1.GetOptions{})
			if err == nil {
				if existingRule != nil {
					specDiffs := diff.CompareIgnoreTargetEmpties(existingRule, newRule)
					if specDiffs != "" {
						logger.Debug().Msgf("Istio DestinationRule %s : Spec differences %s", newRule.Name, specDiffs)
						logger.Info().Msgf("Updating Istio DestinationRule %s:%s in cluster %s", newRule.Namespace, newRule.Name, clusterName)

						if len(newRule.ResourceVersion) == 0 {
							newRule.ResourceVersion = existingRule.ResourceVersion
						}
						_, err = managedClusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules(newRule.Namespace).Update(context.TODO(), newRule, metav1.UpdateOptions{})
					}
				}
			} else if k8sErrors.IsNotFound(err) {
				logger.Info().Msgf("Creating Istio DestinationRule %s:%s in cluster %s", newRule.Namespace, newRule.Name, clusterName)
				_, err = managedClusterConnection.IstioAuthClientSet.NetworkingV1alpha3().DestinationRules(newRule.Namespace).Create(context.TODO(), newRule, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CreateAuthorizationPolicies(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating authorization policies
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "AuthorizationPolicies").Str("name", "ClusterConnection").Logger()

	logger.Debug().Msgf("Creating/updating AuthorizationPolicies for VerrazzanoBinding %s", mbPair.Binding.Name)

	// Parse out the managed clusters that this binding applies to
	filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
	if err != nil {
		return err
	}

	// Construct istio authorization policies for each ManagedCluster
	for clusterName, managedClusterObj := range mbPair.ManagedClusters {
		managedClusterConnection := filteredConnections[clusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		selector := labels.SelectorFromSet(map[string]string{})
		pods, err := managedClusterConnection.PodLister.List(selector)
		if err != nil {
			return err
		}

		// Construct the set of expected isto authorization policies
		newAuthorizationPolicies := newAuthorizationPolicies(mbPair, managedClusterObj, pods)

		for _, newPolicy := range newAuthorizationPolicies {
			existingPolicy, err := managedClusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies(newPolicy.Namespace).Get(context.TODO(), newPolicy.Name, metav1.GetOptions{})

			if err == nil {
				if existingPolicy != nil {
					specDiffs := diff.CompareIgnoreTargetEmpties(existingPolicy, newPolicy)
					if specDiffs != "" {
						logger.Debug().Msgf("Istio AuthorizationPolicy %s : Spec differences %s", newPolicy.Name, specDiffs)
						logger.Info().Msgf("Updating Istio AuthorizationPolicy %s:%s in cluster %s", newPolicy.Namespace, newPolicy.Name, clusterName)

						if len(newPolicy.ResourceVersion) == 0 {
							newPolicy.ResourceVersion = existingPolicy.ResourceVersion
						}
						_, err = managedClusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies(newPolicy.Namespace).Update(context.TODO(), newPolicy, metav1.UpdateOptions{})
					}
				}
			} else if k8sErrors.IsNotFound(err) {
				logger.Info().Msgf("Creating Istio AuthorizationPolicy %s:%s in cluster %s", newPolicy.Namespace, newPolicy.Name, clusterName)
				_, err = managedClusterConnection.IstioAuthClientSet.SecurityV1beta1().AuthorizationPolicies(newPolicy.Namespace).Create(context.TODO(), newPolicy, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//Get a unique IP address to be used for the ServiceEntry address
func getUniqueServiceEntryAddress(managedClusterConnection *util.ManagedClusterConnection, startIPIndex *int) (string, error) {
	// Create log instance for getting service entry address
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "ServiceEntryAddress").Str("name", "ClusterConnection").Logger()

	var baseIP = "240.0.0"
	uniqueIP := fmt.Sprintf("%s.%s", baseIP, strconv.Itoa(*startIPIndex))

	// Get list of serviceentries for this cluster connection across all namespaces
	seList, err := managedClusterConnection.IstioServiceEntryLister.ServiceEntries("").List(labels.Nothing())
	if err != nil {
		return "", err
	}

	// Get an IP address starting at 240.0.0.1 that is not being used by any other serviceentries
	for {
		match := false
		for _, se := range seList {
			if se.Spec.Addresses != nil {
				for _, address := range se.Spec.Addresses {
					if address == uniqueIP {
						logger.Debug().Msgf("found ServiceEntry with address %s: trying next address", uniqueIP)
						*startIPIndex += 1
						uniqueIP = fmt.Sprintf("%s.%s", baseIP, strconv.Itoa(*startIPIndex))
						match = true
						break
					}
				}
			}
			if match {
				break
			}
		}
		if !match {
			*startIPIndex += 1
			break
		}
	}

	return uniqueIP, nil
}

func CleanupOrphanedIngresses(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for cleaning ingresses
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "CleanIngress").Str("name", "ClusterConnection").Logger()

	logger.Debug().Msgf("Cleaning up orphaned Ingresses for VerrazzanoBinding %s", mbPair.Binding.Name)

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

		// Get the set of expected ingresses (gateway and virtual service objects returned)
		ingGateways, ingVirtualServices := newIngresses(mbPair.Binding, mc)

		// Get list of Gateways for this cluster and given binding
		existingGatewayList, err := managedClusterConnection.IstioGatewayLister.Gateways("").List(selector)
		if err != nil {
			return err
		}
		// Create list of Gateway names expected on this cluster
		var gatewayNames []string
		for _, expGateway := range ingGateways {
			gatewayNames = append(gatewayNames, expGateway.Name)
		}
		// Delete any Gateways not expected on this cluster
		for _, gateway := range existingGatewayList {
			if !util.Contains(gatewayNames, gateway.Name) {
				logger.Info().Msgf("Deleting Istio Gateway %s:%s in cluster %s", gateway.Namespace, gateway.Name, clusterName)
				err := managedClusterConnection.IstioClientSet.NetworkingV1alpha3().Gateways(gateway.Namespace).Delete(context.TODO(), gateway.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}

		// Get list of Virtual Services for this cluster and given binding
		existingVirtualServiceList, err := managedClusterConnection.IstioVirtualServiceLister.VirtualServices("").List(selector)
		if err != nil {
			return err
		}
		// Create list of Virtual Service names expected on this cluster
		var virtualServices []string
		for _, expVirtualService := range ingVirtualServices {
			virtualServices = append(virtualServices, expVirtualService.Name)
		}
		// Delete any Virtual Services not expected on this cluster
		for _, virtualService := range existingVirtualServiceList {
			if !util.Contains(virtualServices, virtualService.Name) {
				logger.Info().Msgf("Deleting Istio VirtualService %s:%s in cluster %s", virtualService.Namespace, virtualService.Name, clusterName)
				err := managedClusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices(virtualService.Namespace).Delete(context.TODO(), virtualService.Name, metav1.DeleteOptions{})
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

		// Get list of Gateways for this cluster and given binding
		existingGatewayList, err := managedClusterConnection.IstioGatewayLister.Gateways("").List(selector)
		if err != nil {
			return err
		}
		// Delete these Gateways since none are expected on this cluster
		for _, gw := range existingGatewayList {
			logger.Info().Msgf("Deleting Istio Gateway %s:%s in cluster %s", gw.Namespace, gw.Name, clusterName)
			err := managedClusterConnection.IstioClientSet.NetworkingV1alpha3().Gateways(gw.Namespace).Delete(context.TODO(), gw.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}

		// Get list of Virtual Services for this cluster and given binding
		existingVirtualServiceList, err := managedClusterConnection.IstioVirtualServiceLister.VirtualServices("").List(selector)
		if err != nil {
			return err
		}
		// Delete these Virtual Services since none are expected on this cluster
		for _, vs := range existingVirtualServiceList {
			logger.Info().Msgf("Deleting Istio VirtualService %s:%s in cluster %s", vs.Namespace, vs.Name, clusterName)
			err := managedClusterConnection.IstioClientSet.NetworkingV1alpha3().VirtualServices(vs.Namespace).Delete(context.TODO(), vs.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CleanupOrphanedServiceEntries(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) error {
	// Create log instance for creating destination rules
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "CleanServiceEntries").Str("name", "ClusterConnection").Logger()

	logger.Debug().Msgf("Cleaning up orphaned ServiceEntries for VerrazzanoBinding %s", mbPair.Binding.Name)

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

		// Get the set of expected serviceentries
		serviceEntries := newServiceEntries(mbPair, mc, availableManagedClusterConnections)

		// Get list of ServiceEntries for this cluster and given binding
		existingServiceEntryList, err := managedClusterConnection.IstioServiceEntryLister.ServiceEntries("").List(selector)
		if err != nil {
			return err
		}
		// Create list of ServiceEntry names expected on this cluster
		var serviceEntryNames []string
		for _, se := range serviceEntries {
			serviceEntryNames = append(serviceEntryNames, se.Name)
		}
		// Delete any ServiceEntries not expected on this cluster
		for _, serviceEntry := range existingServiceEntryList {
			if !util.Contains(serviceEntryNames, serviceEntry.Name) {
				logger.Info().Msgf("Deleting Istio ServiceEntry %s:%s in cluster %s", serviceEntry.Namespace, serviceEntry.Name, clusterName)
				err := managedClusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries(serviceEntry.Namespace).Delete(context.TODO(), serviceEntry.Name, metav1.DeleteOptions{})
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

		// Get list of ServiceEntries for this cluster and given binding
		existingServiceEntryList, err := managedClusterConnection.IstioServiceEntryLister.ServiceEntries("").List(selector)
		if err != nil {
			return err
		}
		// Delete these ServiceEntries since none are expected on this cluster
		for _, se := range existingServiceEntryList {
			logger.Info().Msgf("Deleting Istio ServiceEntry %s:%s in cluster %s", se.Namespace, se.Name, clusterName)
			err := managedClusterConnection.IstioClientSet.NetworkingV1alpha3().ServiceEntries(se.Namespace).Delete(context.TODO(), se.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Construct the necessary Gateway and VirtualService objects
func newIngresses(binding *v1beta1v8o.VerrazzanoBinding, mc *types.ManagedCluster) ([]*istiocrd.Gateway, []*istiocrd.VirtualService) {
	ingressLabels := util.GetManagedBindingLabels(binding, mc.Name)
	var gateways []*istiocrd.Gateway
	var virtualServices []*istiocrd.VirtualService

	for namespace, ingresses := range mc.Ingresses {
		for _, ingress := range ingresses {
			ingressBinding := getIngressBinding(binding, ingress.Name)
			if ingressBinding != nil {
				// Create the Istio Gateway
				gateways = append(gateways, &istiocrd.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ingress.Name + "-gateway",
						Namespace: namespace,
						Labels:    ingressLabels,
					},
					Spec: istiocrd.GatewaySpec{
						Servers: []istiocrd.Server{
							{
								Port: istiocrd.Port{
									Name:     "http",
									Number:   80,
									Protocol: "HTTP",
								},
								Hosts: []string{
									ingressBinding.DnsName,
								},
							},
						},
						Selector: map[string]string{"istio": "ingressgateway"},
					},
				})

				// Create the Istio VirtualService
				virtualServices = append(virtualServices,
					newVirtualService(namespace, ingressLabels, ingress, ingressBinding))
			}
		}
	}
	return gateways, virtualServices
}

// Create the Istio VirtualService
func newVirtualService(namespace string, ingressLabels map[string]string, ingress *types.Ingress, ingressBinding *v1beta1v8o.VerrazzanoIngressBinding) *istiocrd.VirtualService {
	return &istiocrd.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingress.Name + "-virtualservice",
			Namespace: namespace,
			Labels:    ingressLabels,
		},
		Spec: istiocrd.VirtualServiceSpec{
			Gateways: []string{
				ingress.Name + "-gateway",
			},
			Hosts: []string{
				ingressBinding.DnsName,
			},
			Http: httpRoutes(namespace, ingress),
		},
	}
}

func httpRoutes(namespace string, ingress *types.Ingress) []istiocrd.HttpRoute {
	var routes []istiocrd.HttpRoute
	for _, destination := range ingress.Destination {
		route := istiocrd.HttpRoute{
			Match: []istiocrd.MatchRequest{},
			Route: []istiocrd.HTTPRouteDestination{
				{
					Destination: istiocrd.Destination{
						Host: destination.Host,
						Port: istiocrd.PortSelector{
							Number: destination.Port,
						},
					},
				},
			},
		}
		for _, m := range destination.Match {
			for k, v := range m.Uri {
				route.Match = addMatch(route.Match, k, v)
			}
		}
		routes = append(routes, route)

		// If the destination is a WebLogic domain then add a match for the
		// WLS console
		if len(destination.DomainName) != 0 {
			routes = append(routes, istiocrd.HttpRoute{
				Match: []istiocrd.MatchRequest{
					{
						Uri: map[string]string{"prefix": "/console"},
					},
				},
				Route: []istiocrd.HTTPRouteDestination{
					{
						Destination: istiocrd.Destination{
							Host: fmt.Sprintf("%s-admin-server.%s.svc.cluster.local", destination.DomainName, namespace),
							Port: istiocrd.PortSelector{
								Number: 7001,
							},
						},
					},
				},
			})
		}
	}
	return routes
}

func addMatch(matches []istiocrd.MatchRequest, key, value string) []istiocrd.MatchRequest {
	found := false
	for _, match := range matches {
		if match.Uri[key] == value {
			found = true
		}
	}
	if !found {
		matches = append(matches, istiocrd.MatchRequest{
			Uri: map[string]string{key: value},
		})
	}
	return matches
}

// Construct the necessary ServiceEntry objects
func newServiceEntries(mbPair *types.ModelBindingPair, mc *types.ManagedCluster, availableManagedClusterConnections map[string]*util.ManagedClusterConnection) []*istiocrd.ServiceEntry {

	bindingLabels := util.GetManagedBindingLabels(mbPair.Binding, mc.Name)
	var serviceEntries []*istiocrd.ServiceEntry

	for namespace, rests := range mc.RemoteRests {
		for _, rest := range rests {
			// Get the istio gateway address
			gatewayAddress := getIstioGateways(mbPair, availableManagedClusterConnections, rest.RemoteClusterName)
			// Create the istio ServiceEntry
			serviceEntries = append(serviceEntries, &istiocrd.ServiceEntry{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rest.Name,
					Namespace: namespace,
					Labels:    bindingLabels,
				},
				Spec: istio.ServiceEntry{
					Hosts: []string{
						fmt.Sprintf("%s.%s.global", rest.Name, rest.RemoteNamespace),
					},
					Location: istio.ServiceEntry_MESH_INTERNAL,
					Ports: []*istio.Port{
						{
							Name:     "http1",
							Number:   rest.Port,
							Protocol: "http",
						},
					},
					Resolution: istio.ServiceEntry_DNS,
					Endpoints: []*istio.WorkloadEntry{
						{
							Address: gatewayAddress,
							Ports: map[string]uint32{
								"http1": 15443,
							},
						},
					},
					ExportTo: []string{
						".",
					},
				},
			})
		}
	}

	return serviceEntries
}

// Construct the necessary Istio DestinationRule objects
// Destination rules are created for each namespace named in the binding placement.  Each destination rule enables
// MTLS for the entire namespace with one exception... MTLS is disabled for traffic on the WebLogic admin port to avoid
// issues with WebLogic managed server communication with the admin server.
func newDestinationRules(mbPair *types.ModelBindingPair, mc *types.ManagedCluster, pods []*v1.Pod) ([]*v1alpha3.DestinationRule, error) {
	var rules []*v1alpha3.DestinationRule
	namespaceMap := make(map[string]struct{})

	// get all the namespaces from the binding
	for _, placement := range mbPair.Binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			namespaceMap[namespace.Name] = struct{}{}
		}
	}
	// create an destination rule for each binding namespace in the given cluster
	for _, namespace := range mc.Namespaces {
		if _, ok := namespaceMap[namespace]; ok {
			// create a destination rule spec that enables MTLS for the namespace
			rules = append(rules, &v1alpha3.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespace + "-destination-rule",
					Namespace: namespace,
				},
				Spec: istio.DestinationRule{
					Host: "*." + namespace + ".svc.cluster.local",
					TrafficPolicy: &istio.TrafficPolicy{
						Tls: &istio.ClientTLSSettings{
							Mode: istio.ClientTLSSettings_ISTIO_MUTUAL,
						},
					},
				},
			})
		}
	}
	return rules, nil
}

// Construct the necessary Istio AuthorizationPolicy objects
// Authorization policies are created for each namespace specified in the binding.  The policy for each namespace
// only allows traffic from other component namespaces for which their is a connection specified in the model.
// Since Coherence needs to be excluded from the service mesh, traffic is also allowed from all pod IPs in the
// same namespace to allow for traffic between the COH storage nodes.
func newAuthorizationPolicies(mbPair *types.ModelBindingPair, mc *types.ManagedCluster, pods []*v1.Pod) []*v1beta1.AuthorizationPolicy {
	var authorizationPolicies []*v1beta1.AuthorizationPolicy
	// map of v8o components to namespace
	componentNameSpaces := make(map[string]string)
	// map of namespaces to set of connected component namespaces
	namespaceSources := make(map[string]map[string]struct{})

	// map all of the model components to their namespaces
	mapComponentNamespaces(mbPair, componentNameSpaces, namespaceSources)

	// map the connected namespace sources for each WebLogic component
	for _, weblogic := range mbPair.Model.Spec.WeblogicDomains {
		mapNamespaceSources(weblogic.Name, weblogic.Connections, componentNameSpaces, namespaceSources)
	}

	// map the connected namespace sources for each Helidon component
	for _, helidon := range mbPair.Model.Spec.HelidonApplications {
		mapNamespaceSources(helidon.Name, helidon.Connections, componentNameSpaces, namespaceSources)
	}

	// create an authorization policy for each binding namespace in the given cluster
	for _, ns := range mc.Namespaces {
		if namespaceSources[ns] != nil {
			// get the ips for all the pods connected to this namespace
			podIPs := getIpsOfNamespacePods(ns, pods)
			systemPrometheusPodIp := getIpOfSystemPrometheusPod(pods)
			if systemPrometheusPodIp != "" {
				podIPs = append(podIPs, systemPrometheusPodIp)
			}
			// allow traffic from any of the binding namespaces that are connected to components in this namespace
			fromRules := []*istiosecv1beta1.Rule_From{
				{
					Source: &istiosecv1beta1.Source{
						Namespaces: append(GetNamespaceValues(namespaceSources, ns), IstioSystemNamespace),
					},
				},
			}
			// allow traffic from the Coherence pod ips
			if len(podIPs) > 0 {
				fromRules = append(fromRules, &istiosecv1beta1.Rule_From{
					Source: &istiosecv1beta1.Source{
						IpBlocks: podIPs,
					},
				})
			}
			// create a authorization policy in this namespace with the 'from' rules
			authorizationPolicies = append(authorizationPolicies, &v1beta1.AuthorizationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ns + "-authorization-policy",
					Namespace: ns,
				},
				Spec: istiosecv1beta1.AuthorizationPolicy{
					Rules: []*istiosecv1beta1.Rule{
						{
							From: fromRules,
						},
					},
				},
			})
		}
	}
	return authorizationPolicies
}

// map the model components to their namespaces
func mapComponentNamespaces(mbPair *types.ModelBindingPair, componentNameSpaces map[string]string, namespaces map[string]map[string]struct{}) {
	for _, placement := range mbPair.Binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			for _, component := range namespace.Components {
				componentNameSpaces[component.Name] = namespace.Name
				AddToNamespace(namespaces, namespace.Name, namespace.Name)
			}
		}
	}
}

// map the connected namespace sources for the given component connections
func mapNamespaceSources(componentName string, connections []v1beta1v8o.VerrazzanoConnections, componentNameSpaces map[string]string, namespaceSources map[string]map[string]struct{}) {

	ns := componentNameSpaces[componentName]
	for _, connection := range connections {
		for _, rest := range connection.Rest {
			// add the component namespace to the set of namespaces allowed to access components in the connection target namespace
			AddToNamespace(namespaceSources, componentNameSpaces[rest.Target], ns)
		}
	}
}

// Get the IP addresses of all pods in the provided namespace
func getIpsOfNamespacePods(ns string, pods []*v1.Pod) []string {
	var podIPs []string
	for _, pod := range pods {
		if pod.Status.PodIP != "" && ns == pod.Namespace {
			podIPs = append(podIPs, pod.Status.PodIP)
		}
	}
	return podIPs
}

func getIpOfSystemPrometheusPod(pods []*v1.Pod) string {
	// Create log instance for cleaning ingresses
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "PrometheusPod").Str("name", "IPOutOfSystem").Logger()

	for _, pod := range pods {
		if pod.Namespace == IstioSystemNamespace && strings.HasPrefix(pod.Name, PrometheusPodPrefix) {
			return pod.Status.PodIP
		}
	}
	logger.Error().Msg("Unable to obtain IP address of System Prometheus Pod for authorization policy")
	return ""
}

// Add a value to a set for a given namespace map
func AddToNamespace(nsMap map[string]map[string]struct{}, ns string, value string) {
	if nsMap[ns] == nil {
		nsMap[ns] = map[string]struct{}{}
	}
	nsMap[ns][value] = struct{}{}
}

// Get the set of values for a given namespace and map
func GetNamespaceValues(nsMap map[string]map[string]struct{}, ns string) []string {
	var values []string
	if nsMap[ns] != nil {
		for value := range nsMap[ns] {
			values = append(values, value)
		}
	}
	return values
}

// Determine whether a value is set for a given namespace
func NamespaceContainsValue(nsMap map[string]map[string]struct{}, ns string, value string) bool {
	if nsMap[ns] != nil {
		if _, ok := nsMap[ns][value]; ok {
			return true
		}
	}
	return false
}

func getIstioGateways(mbPair *types.ModelBindingPair, availableManagedClusterConnections map[string]*util.ManagedClusterConnection, remoteClusterName string) string {
	// Create log instance for getting istio gateways
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "RemoteCluster").Str("name", remoteClusterName).Logger()

	logger.Debug().Msgf("Getting istio-ingressgateway addresses for VerrazzanoBinding %s", mbPair.Binding.Name)

	if _, ok := mbPair.ManagedClusters[remoteClusterName]; ok {
		// Parse out the managed clusters that this binding applies to
		filteredConnections, err := util.GetManagedClustersForVerrazzanoBinding(mbPair, availableManagedClusterConnections)
		if err != nil {
			return ""
		}

		managedClusterConnection := filteredConnections[remoteClusterName]
		managedClusterConnection.Lock.RLock()
		defer managedClusterConnection.Lock.RUnlock()

		logger.Debug().Msgf("Getting istio-ingressgateway address in cluster %s", remoteClusterName)
		service, err := managedClusterConnection.KubeClient.CoreV1().Services(IstioSystemNamespace).Get(context.TODO(), "istio-ingressgateway", metav1.GetOptions{})

		if err != nil || service == nil {
			logger.Error().Msgf("failed to get istio-ingressgateway service for cluster %s, %v", remoteClusterName, err)
			return ""
		}

		if service.Status.LoadBalancer.Ingress == nil || len(service.Status.LoadBalancer.Ingress) == 0 || service.Status.LoadBalancer.Ingress[0].IP == "" {
			logger.Error().Msgf("Invalid external load balancer configuration for istio-ingressgateway service for cluster %s", remoteClusterName)
			return ""
		}

		return service.Status.LoadBalancer.Ingress[0].IP
	}

	return ""
}

func getIngressBinding(binding *v1beta1v8o.VerrazzanoBinding, name string) *v1beta1v8o.VerrazzanoIngressBinding {
	for _, ingressBinding := range binding.Spec.IngressBindings {
		if ingressBinding.Name == name {
			return &ingressBinding
		}
	}
	return nil
}
