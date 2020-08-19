// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	v8weblogic "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	v1helidonapp "github.com/verrazzano/verrazzano-helidon-app-operator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/cohcluster"
	"github.com/verrazzano/verrazzano-operator/pkg/cohoperator"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
	"github.com/verrazzano/verrazzano-operator/pkg/helidonapp"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/wlsdom"
	"github.com/verrazzano/verrazzano-operator/pkg/wlsopr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateModelBindingPair(model *v1beta1v8o.VerrazzanoModel, binding *v1beta1v8o.VerrazzanoBinding, verrazzanoUri string, sslVerify bool) *types.ModelBindingPair {

	mbPair := &types.ModelBindingPair{
		Model:           model,
		Binding:         binding,
		ManagedClusters: map[string]*types.ManagedCluster{},
		Lock:            sync.RWMutex{},
		VerrazzanoUri:   verrazzanoUri,
		SslVerify:       sslVerify,
	}
	return buildModelBindingPair(mbPair)
}

func acquireLock(mbPair *types.ModelBindingPair) {
	mbPair.Lock.Lock()
}

func releaseLock(mbPair *types.ModelBindingPair) {
	mbPair.Lock.Unlock()
}

// Update a model/binding pair by rebuilding it
func UpdateModelBindingPair(mbPair *types.ModelBindingPair, model *v1beta1v8o.VerrazzanoModel, binding *v1beta1v8o.VerrazzanoBinding, verrazzanoUri string, sslVerify bool) {
	newMBPair := CreateModelBindingPair(model, binding, verrazzanoUri, sslVerify)
	lock := mbPair.Lock
	lock.Lock()
	defer lock.Unlock()
	*mbPair = *newMBPair
}

// Common function for building a model/binding pair
func buildModelBindingPair(mbPair *types.ModelBindingPair) *types.ModelBindingPair {

	// Acquire write lock
	acquireLock(mbPair)
	defer releaseLock(mbPair)

	// Obtain the placement objects
	placements := getPlacements(mbPair.Binding)

	for _, placement := range placements {
		var mc *types.ManagedCluster
		mc, clusterFound := mbPair.ManagedClusters[placement.Name]
		// Create the ManagedCluster object and add it to the map if we don't have one
		// for a given placement (cluster)
		if !clusterFound {
			mc = &types.ManagedCluster{
				Name:        placement.Name,
				Secrets:     map[string][]string{},
				Ingresses:   map[string][]*types.Ingress{},
				RemoteRests: map[string][]*types.RemoteRestConnection{},
			}
			mbPair.ManagedClusters[placement.Name] = mc
		}

		// Add in the Verrazzano system namespace if not already added
		managedNamespace := util.GetManagedClusterNamespaceForSystem()
		appendNamespace(&mc.Namespaces, managedNamespace)

		// Loop over the namespaces within this placement
		for _, namespace := range placement.Namespaces {

			appendNamespace(&mc.Namespaces, namespace.Name)

			// Create a map of the component names
			compNames := make(map[string]v1beta1v8o.BindingComponent, len(namespace.Components))
			for _, s := range namespace.Components {
				compNames[s.Name] = s
			}

			// Does this namespace contain any WebLogic domain components?
			domains := mbPair.Model.Spec.WeblogicDomains
			if domains != nil {
				for _, domain := range domains {
					_, found := compNames[domain.Name]
					if found {

						// The managed cluster needs a WebLogic Operator for the domain
						if mc.WlsOperator == nil {
							oprLabels := util.GetManagedBindingLabels(mbPair.Binding, mc.Name)
							mc.WlsOperator = wlsopr.CreateWlsOperatorCR(mbPair.Binding, mc.Name, namespace.Name, oprLabels)
							appendNamespace(&mc.Namespaces, mc.WlsOperator.Spec.Namespace)
							if len(mc.WlsOperator.Spec.ImagePullSecret) > 0 {
								addSecret(mc, mc.WlsOperator.Spec.ImagePullSecret, namespace.Name)
							}
						}

						// Have the WebLogic Operator watch the namespace of the domain
						appendNamespace(&mc.WlsOperator.Spec.DomainNamespaces, namespace.Name)

						var dbSecrets []string = nil
						var cmData []string = nil

						// For each database binding check to see if there are any corresponding domain connections
						for _, databaseBinding := range mbPair.Binding.Spec.DatabaseBindings {
							datasourceName := getDatasourceName(domain, databaseBinding)
							// If this domain has a database connection that targets this database binding...
							if len(datasourceName) > 0 {
								dbSecrets = append(dbSecrets, databaseBinding.Credentials)

								dataSourceTarget := getDataSourceTarget(&domain.DomainCRValues)

								// Create the datasource model configuration for MySql connections.
								if strings.HasPrefix(strings.TrimSpace(databaseBinding.Url), "jdbc:mysql") {
									modelConfig := createMysqlDatasourceModelConfig(databaseBinding.Credentials, datasourceName, dataSourceTarget)
									if cmData == nil {
										cmData = append(cmData, "resources:\n  JDBCSystemResource:\n")
									}
									cmData = append(cmData, modelConfig)
									continue
								}

								// Create the datasource model configuration for Oracle connections.
								if strings.HasPrefix(strings.TrimSpace(databaseBinding.Url), "jdbc:oracle") {
									modelConfig := createOracleDatasourceModelConfig(databaseBinding.Credentials, datasourceName, dataSourceTarget)
									if cmData == nil {
										cmData = append(cmData, "resources:\n  JDBCSystemResource:\n")
									}
									cmData = append(cmData, modelConfig)
								}
							}
						}

						// Create a config map with all the saved data source model configurations
						var datasourceModelConfigMap = ""
						if cmData != nil {
							var configMap *corev1.ConfigMap
							var domainUID string

							if len(domain.DomainCRValues.DomainUID) > 0 {
								domainUID = domain.DomainCRValues.DomainUID
							} else {
								domainUID = domain.Name
							}

							labels := make(map[string]string)
							labels["weblogic.domainUID"] = domainUID

							data := make(map[string]string)
							data["datasource.yaml"] = strings.Join(cmData, "")
							configMap = &corev1.ConfigMap{
								ObjectMeta: metav1.ObjectMeta{
									Name:      domainUID + "-wdt-config-map",
									Namespace: namespace.Name,
									Labels:    labels,
								},
								Data: data,
							}
							datasourceModelConfigMap = configMap.Name
							mc.ConfigMaps = append(mc.ConfigMaps, configMap)
						}

						// Create the WlsDomain CR and update the secrets list for the namespace
						domLabels := util.GetManagedBindingLabels(mbPair.Binding, mc.Name)
						domainCR := wlsdom.CreateWlsDomainCR(namespace.Name, domain, mbPair, domLabels, datasourceModelConfigMap, dbSecrets)
						if domainCR.Spec.ImagePullSecrets != nil {
							for _, secret := range domainCR.Spec.ImagePullSecrets {
								addSecret(mc, secret.Name, namespace.Name)
							}
						}
						addSecret(mc, domainCR.Spec.WebLogicCredentialsSecret.Name, namespace.Name)
						mc.WlsDomainCRs = append(mc.WlsDomainCRs, domainCR)

						// Create fluentd configmap
						configMap := wlsdom.CreateFluentdConfigMap(namespace.Name, domLabels)
						mc.ConfigMaps = append(mc.ConfigMaps, configMap)

						// Add secret for binding to the namespace, it contains the credentials fluentd needs for ElasticSearch
						addSecret(mc, constants.VmiSecretName, namespace.Name)

						virtualSerivceDestinationPort := int(getDomainDestinationPort(domainCR))
						processIngressConnections(mc, domain.Connections, namespace.Name, domainCR, getDomainDestinationHost(domainCR), virtualSerivceDestinationPort, &mbPair.Binding.Spec.IngressBindings)
					}
				}
			}

			// Does this namespace contain any Coherence cluster components?
			cohClusters := mbPair.Model.Spec.CoherenceClusters
			if cohClusters != nil {
				for _, cluster := range cohClusters {
					_, found := compNames[cluster.Name]
					if found {
						// Get the Coherence binding
						var cohBinding *v1beta1v8o.VerrazzanoCoherenceBinding
						for _, binding := range mbPair.Binding.Spec.CoherenceBindings {
							if binding.Name == cluster.Name {
								cohBinding = &binding
								break
							}
						}

						if cohBinding != nil {
							// The managed cluster namespace needs a Coherence Operator to create coherence clusters
							// There should only be one of these created per namespace
							oprFound := false
							for _, cohOperator := range mc.CohOperatorCRs {
								if cohOperator.Spec.Namespace == namespace.Name {
									oprFound = true
									break
								}
							}

							// Create the CR to create the coherence operator if we don't have an operator
							// for given namespace
							if !oprFound {
								oprLabels := util.GetManagedBindingLabels(mbPair.Binding, mc.Name)
								operatorCR := cohoperator.CreateCR(mc.Name, namespace.Name, &cluster, oprLabels)
								mc.CohOperatorCRs = append(mc.CohOperatorCRs, operatorCR)
								for _, secret := range operatorCR.Spec.ImagePullSecrets {
									addSecret(mc, secret.Name, operatorCR.Spec.Namespace)
								}
							}

							// Create the CR to create the coherence cluster
							clusterLabels := util.GetManagedBindingLabels(mbPair.Binding, mc.Name)
							clusterCR := cohcluster.CreateCR(namespace.Name, &cluster, cohBinding, clusterLabels)
							mc.CohClusterCRs = append(mc.CohClusterCRs, clusterCR)
							for _, secret := range clusterCR.Spec.ImagePullSecrets {
								addSecret(mc, secret.Name, namespace.Name)
							}
						} else {
							glog.Errorf("Coherence binding '%s' not found in binding file", cluster.Name)
						}
					}
				}
			}

			// Does this namespace contain any Helidon application components?
			helidonApps := mbPair.Model.Spec.HelidonApplications
			if helidonApps != nil {
				for _, app := range helidonApps {
					_, found := compNames[app.Name]
					if found {
						// Create the HelidonApp CR and update the secrets list for the namespace
						helidonLabels := util.GetManagedBindingLabels(mbPair.Binding, mc.Name)
						helidonCR := helidonapp.CreateHelidonAppCR(mc.Name, namespace.Name, &app, mbPair, helidonLabels)
						for _, secret := range helidonCR.Spec.ImagePullSecrets {
							addSecret(mc, secret.Name, namespace.Name)
						}
						mc.HelidonApps = append(mc.HelidonApps, helidonCR)

						// Include Fluentd if Fluentd integration is enabled
						if helidonapp.IsFluentdEnabled(&app) {
							configMap := helidonapp.CreateFluentdConfigMap(&app, namespace.Name, helidonLabels)
							mc.ConfigMaps = append(mc.ConfigMaps, configMap)
							// Add secret for binding to the namespace, it contains the credentials fluentd needs for ElasticSearch
							addSecret(mc, constants.VmiSecretName, namespace.Name)
						}

						virtualSerivceDestinationPort := int(helidonCR.Spec.Port) //service.port
						processIngressConnections(mc, app.Connections, namespace.Name, nil,
							getHelidonDestinationHost(helidonCR), virtualSerivceDestinationPort, &mbPair.Binding.Spec.IngressBindings)
					}
				}
			}
		}
	}

	// Now that we have the managed clusters setup take another pass to process rest connections
	// since we need all clusters setup to do this
	for _, placement := range placements {
		mc := mbPair.ManagedClusters[placement.Name]

		// Loop over the namespaces within this placement
		for _, namespace := range placement.Namespaces {
			// Create a map of the component names for a given namespace
			compNames := make(map[string]v1beta1v8o.BindingComponent, len(namespace.Components))
			for _, s := range namespace.Components {
				compNames[s.Name] = s
			}

			domains := mbPair.Model.Spec.WeblogicDomains
			if domains != nil {
				for _, domain := range domains {
					_, found := compNames[domain.Name]
					if found {
						for _, connection := range domain.Connections {
							createRemoteRestConnections(mbPair, mc, connection.Rest, namespace.Name)
							envVars := getRestConnectionEnvVars(mbPair, connection.Rest, domain.Name)
							wlsdom.UpdateEnvVars(mc, domain.Name, envVars)
						}
					}
				}
			}

			cohClusters := mbPair.Model.Spec.CoherenceClusters
			if cohClusters != nil {
				for _, cluster := range cohClusters {
					_, found := compNames[cluster.Name]
					if found {
						for _, connection := range cluster.Connections {
							createRemoteRestConnections(mbPair, mc, connection.Rest, namespace.Name)
							envVars := getRestConnectionEnvVars(mbPair, connection.Rest, cluster.Name)
							cohcluster.UpdateEnvVars(mc, cluster.Name, envVars)
						}
					}
				}
			}

			helidonApps := mbPair.Model.Spec.HelidonApplications
			if helidonApps != nil {
				for _, app := range helidonApps {
					_, found := compNames[app.Name]
					if found {
						for _, connection := range app.Connections {
							createRemoteRestConnections(mbPair, mc, connection.Rest, namespace.Name)
							envVars := getRestConnectionEnvVars(mbPair, connection.Rest, app.Name)
							helidonapp.UpdateEnvVars(mc, app.Name, envVars)
						}
					}
				}
			}
		}
	}

	return mbPair
}

func getPlacements(binding *v1beta1v8o.VerrazzanoBinding) []v1beta1v8o.VerrazzanoPlacement {
	placements := binding.Spec.Placement
	return placements
}

// Add the name of a secret to the array of secret names in a managed cluster.
func addSecret(mc *types.ManagedCluster, secretName string, namespace string) {
	// Check to see if the name of the secret has already been added
	secretsList, ok := mc.Secrets[namespace]
	if ok {
		for _, name := range secretsList {
			if name == secretName {
				return
			}
		}
	}
	mc.Secrets[namespace] = append(mc.Secrets[namespace], secretName)
}

// Add the name of an ingress to the array of ingresses in a managed cluster
func addIngress(mc *types.ManagedCluster, ingressConn v1beta1v8o.VerrazzanoIngressConnection, namespace string, domainCR *v8weblogic.Domain, destinationHost string, virtualSerivceDestinationPort int) {
	var domainName = ""
	if domainCR != nil {
		domainName = domainCR.Spec.DomainUID
	}
	ingress := getOrNewIngress(mc, ingressConn, namespace)
	dest := getOrNewDest(ingress, destinationHost, virtualSerivceDestinationPort)
	dest.DomainName = domainName
	dest.Match = []types.MatchRequest{}
	for _, m := range ingressConn.Match {
		for k, v := range m.Uri {
			dest.Match = addMatch(dest.Match, k, v)
		}
	}
	if len(dest.Match) == 0 {
		//default prefix := "/"
		dest.Match = addMatch(dest.Match, "prefix", "/")
	}
}

func addMatch(matches []types.MatchRequest, key, value string) []types.MatchRequest {
	found := false
	for _, match := range matches {
		if match.Uri[key] == value {
			found = true
		}
	}
	if !found {
		matches = append(matches, types.MatchRequest{
			Uri: map[string]string{key: value},
		})
	}
	return matches
}

func getOrNewIngress(mc *types.ManagedCluster, ingressConn v1beta1v8o.VerrazzanoIngressConnection, namespace string) *types.Ingress {
	ingressList, ok := mc.Ingresses[namespace]
	if ok {
		for _, ingress := range ingressList {
			if ingress.Name == ingressConn.Name {
				return ingress
			}
		}
	}
	ingress := types.Ingress{
		Name:        ingressConn.Name,
		Destination: []*types.IngressDestination{},
	}
	mc.Ingresses[namespace] = append(mc.Ingresses[namespace], &ingress)
	return &ingress
}

func getOrNewDest(ingress *types.Ingress, destinationHost string, virtualSerivceDestinationPort int) *types.IngressDestination {
	for _, destination := range ingress.Destination {
		if destination.Host == destinationHost && destination.Port == virtualSerivceDestinationPort {
			return destination
		}
	}
	dest := types.IngressDestination{
		Host: destinationHost,
		Port: virtualSerivceDestinationPort,
	}
	ingress.Destination = append(ingress.Destination, &dest)
	return &dest
}

// Add a remote rest connection to the array of remote rest connections in a managed cluster
func addRemoteRest(mc *types.ManagedCluster, restName string, localNamespace string, remoteMc *types.ManagedCluster, remoteNamespace string, remotePort uint32, remoteClusterName string, remoteType types.ComponentType) {
	// If already added for a local namespace just return
	for _, restConnections := range mc.RemoteRests {
		for _, rest := range restConnections {
			if rest.Name == restName && rest.LocalNamespace == localNamespace {
				return
			}
		}
	}

	// Add a new remote rest for this namespace
	mc.RemoteRests[localNamespace] = append(mc.RemoteRests[localNamespace], &types.RemoteRestConnection{
		Name: func() string {
			var name string
			if remoteType == types.Wls {
				for _, domain := range remoteMc.WlsDomainCRs {
					if restName == domain.Name {
						name = getDomainHostPrefix(domain)
						break
					}
				}
			} else {
				name = restName
			}
			return name
		}(),
		RemoteNamespace:   remoteNamespace,
		LocalNamespace:    localNamespace,
		Port:              remotePort,
		RemoteClusterName: remoteClusterName,
	})
}

func processIngressConnections(mc *types.ManagedCluster, connections []v1beta1v8o.VerrazzanoConnections, namespace string, domainCR *v8weblogic.Domain, destinationHost string, virtualSerivceDestinationPort int, ingressBindings *[]v1beta1v8o.VerrazzanoIngressBinding) {
	for _, connection := range connections {
		for _, ingress := range connection.Ingress {
			inBindings := []v1beta1v8o.VerrazzanoIngressBinding{}
			for _, binding := range *ingressBindings {
				if binding.Name == ingress.Name {
					inBindings = append(inBindings, binding)
				}
			}
			addIngress(mc, ingress, namespace, domainCR, destinationHost, virtualSerivceDestinationPort)
		}
	}
}

// Create a RemoteRestConnection type for each remote rest connection
func createRemoteRestConnections(mbPair *types.ModelBindingPair, mc *types.ManagedCluster, connections []v1beta1v8o.VerrazzanoRestConnection, localNamespace string) {
	for _, rest := range connections {
		// Get the placement objects
		placements := getPlacements(mbPair.Binding)
		for _, placement := range placements {
			isRemote, remoteNamespace, remoteClusterName := isRestTargetRemote(mc, rest.Target, placement)
			if isRemote {
				remoteType, remotePort, err := getTargetTypePort(mbPair.ManagedClusters[placement.Name], rest.Target)
				if err == nil {
					addRemoteRest(mc, rest.Target, localNamespace, mbPair.ManagedClusters[placement.Name], remoteNamespace, remotePort, remoteClusterName, remoteType)
				} else {
					glog.Errorf("error getting remote port, %v", err)
				}
				break
			}
		}
	}
}

func isRestTargetRemote(mc *types.ManagedCluster, restTarget string, placement v1beta1v8o.VerrazzanoPlacement) (bool, string, string) {
	// Rest connection is within the same cluster so this is not remote connection
	if placement.Name == mc.Name {
		return false, "", ""
	}

	// Rest connection must be to another cluster to be a remote connection
	for _, namespace := range placement.Namespaces {
		for _, component := range namespace.Components {
			// We found the remote target.  Return the target name and remote cluster name
			if restTarget == component.Name {
				return true, namespace.Name, placement.Name
			}
		}
	}

	return false, "", ""
}

// Get the environment variables that need to be set based on the rest connection
func getRestConnectionEnvVars(mbPair *types.ModelBindingPair, connections []v1beta1v8o.VerrazzanoRestConnection, crName string) *[]corev1.EnvVar {
	var env corev1.EnvVar
	var envs []corev1.EnvVar
	for _, restConnection := range connections {
		// Get the placement for the component we need to create env variables for
		sourcePlacement, err := getSourcePlacement(crName, mbPair.Binding)
		if err != nil {
			glog.Errorf("unable to create rest connection env variables: %v", err)
			continue
		}

		// Get the namespace and placement for the rest target
		targetNamespace, targetPlacement, err := getTargetNamespacePlacement(restConnection.Target, mbPair.Binding)
		if err != nil {
			glog.Errorf("unable to create rest connection env variables: %v", err)
			continue
		}

		// Check if this rest connection is remote or not
		isRemote := false
		if sourcePlacement != targetPlacement {
			isRemote = true
		}

		remoteMc := mbPair.ManagedClusters[targetPlacement]

		// Get the target type and target port for the rest connection
		targetType, targetPort, err := getTargetTypePort(remoteMc, restConnection.Target)
		if err != nil {
			glog.Errorf("error getting port, %v", err)
			continue
		}

		// Add the environment variables for the rest connection
		env.Name = restConnection.EnvironmentVariableForPort
		env.Value = fmt.Sprint(targetPort)
		envs = append(envs, env)

		env.Name = restConnection.EnvironmentVariableForHost
		if isRemote {
			if targetType == types.Wls {
				for _, domain := range remoteMc.WlsDomainCRs {
					if restConnection.Target == domain.Name {
						env.Value = fmt.Sprintf("%s.%s.global", getDomainHostPrefix(domain), targetNamespace)
						break
					}
				}
			} else {
				env.Value = fmt.Sprintf("%s.%s.global", restConnection.Target, targetNamespace)
			}
		} else {
			if targetType == types.Wls {
				for _, domain := range remoteMc.WlsDomainCRs {
					if restConnection.Target == domain.Name {
						env.Value = fmt.Sprintf("%s.%s.svc.cluster.local", getDomainHostPrefix(domain), targetNamespace)
						break
					}
				}
			} else {
				env.Value = fmt.Sprintf("%s.%s.svc.cluster.local", restConnection.Target, targetNamespace)
			}
		}
		envs = append(envs, env)
	}

	return &envs
}

// Get the namespace and placement for a rest target
func getTargetNamespacePlacement(restTarget string, binding *v1beta1v8o.VerrazzanoBinding) (string, string, error) {
	for _, placement := range binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			for _, component := range namespace.Components {
				if component.Name == restTarget {
					return namespace.Name, placement.Name, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("rest target %s not found in binding", restTarget)
}

// Get the type and port for a rest target
func getTargetTypePort(managedCluster *types.ManagedCluster, target string) (types.ComponentType, uint32, error) {
	for _, wls := range managedCluster.WlsDomainCRs {
		if wls.Name == target {
			return types.Wls, getDomainDestinationPort(wls), nil
		}
	}

	for _, helidon := range managedCluster.HelidonApps {
		if helidon.Name == target {
			return types.Helidon, 8080, nil
		}
	}

	for _, coh := range managedCluster.CohClusterCRs {
		if coh.Name == target {
			return types.Coherence, 0, fmt.Errorf("component %s of type Coherence cannot be target of rest connection", target)
		}
	}

	return types.Unknown, 0, fmt.Errorf("component %s not found on cluster %s", target, managedCluster.Name)
}

// Get the placement for a component
func getSourcePlacement(compName string, binding *v1beta1v8o.VerrazzanoBinding) (string, error) {
	for _, placement := range binding.Spec.Placement {
		for _, namespace := range placement.Namespaces {
			for _, component := range namespace.Components {
				if component.Name == compName {
					return placement.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("component name %s not found in binding", compName)
}

// Utility function to generate the destination host name for a domain
func getDomainDestinationHost(domain *v8weblogic.Domain) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", getDomainHostPrefix(domain), domain.Namespace)
}

func getDomainHostPrefix(domain *v8weblogic.Domain) string {
	if len(domain.Spec.Clusters) == 0 {
		return fmt.Sprintf("%s-AdminServer", domain.Spec.DomainUID)
	} else {
		return fmt.Sprintf("%s-cluster-%s", domain.Spec.DomainUID, domain.Spec.Clusters[0].ClusterName)
	}
}

// Utility function to generate the destination port number for a domain
func getDomainDestinationPort(domain *v8weblogic.Domain) uint32 {
	if len(domain.Spec.Clusters) == 0 {
		return 7001
	} else {
		return 8001
	}
}

func getDataSourceTarget(domainSpec *v8weblogic.DomainSpec) string {
	if len(domainSpec.Clusters) == 0 {
		return "AdminServer"
	} else {
		return domainSpec.Clusters[0].ClusterName
	}
}

// Utility function to generate the destination host name for a helidon app
func getHelidonDestinationHost(app *v1helidonapp.HelidonApp) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace)
}

func appendNamespace(Namespaces *[]string, namespace string) {
	nsFound := false
	for _, ns := range *Namespaces {
		if ns == namespace {
			nsFound = true
			break
		}
	}
	if !nsFound {
		*Namespaces = append(*Namespaces, namespace)
	}

}

// Get the datasource name from the database connection in the given domain that targets the given database binding
func getDatasourceName(domain v1beta1v8o.VerrazzanoWebLogicDomain, databaseBinding v1beta1v8o.VerrazzanoDatabaseBinding) string {
	for _, connection := range domain.Connections {
		for _, databaseConnection := range connection.Database {
			if databaseConnection.Target == databaseBinding.Name {
				return databaseConnection.DatasourceName
			}
		}
	}
	return ""
}

// Create a MySql datasource model configuration for the given db secret and datasource
func createMysqlDatasourceModelConfig(dbSecret string, datasourceName string, dataSourceTarget string) string {

	format := `    %s:
      Target: '%s'
      JdbcResource:
        JDBCDataSourceParams:
          JNDIName: [
            jdbc/%s
          ]
        JDBCDriverParams:
          DriverName: com.mysql.cj.jdbc.Driver
          URL: '@@SECRET:%s:url@@'
          PasswordEncrypted: '@@SECRET:%s:password@@'
          Properties:
            user:
              Value: '@@SECRET:%s:username@@'
        JDBCConnectionPoolParams:
          ConnectionReserveTimeoutSeconds: 10
          InitialCapacity: 0
          MaxCapacity: 5
          MinCapacity: 0
          TestConnectionsOnReserve: true
          TestTableName: SQL SELECT 1
`
	return fmt.Sprintf(format, datasourceName, dataSourceTarget, datasourceName, dbSecret, dbSecret, dbSecret)
}

// Create a Oracle datasource model configuration for the given db secret and datasource
func createOracleDatasourceModelConfig(dbSecret string, datasourceName string, dataSourceTarget string) string {

	format := `    %s:
      Target: '%s'
      JdbcResource:
        JDBCDataSourceParams:
          JNDIName: [
            jdbc/%s
          ]
          GlobalTransactionsProtocol: TwoPhaseCommit
        JDBCDriverParams:
          DriverName: oracle.jdbc.xa.client.OracleXADataSource
          URL: '@@SECRET:%s:url@@'
          PasswordEncrypted: '@@SECRET:%s:password@@'
          Properties:
            user:
              Value: '@@SECRET:%s:username@@'
            oracle.net.CONNECT_TIMEOUT:
              Value: 5000
            oracle.jdbc.ReadTimeout:
              Value: 30000
        JDBCConnectionPoolParams:
            InitialCapacity: 0
            MaxCapacity: 1
            TestTableName: SQL ISVALID
            TestConnectionsOnReserve: true
`

	return fmt.Sprintf(format, datasourceName, dataSourceTarget, datasourceName, dbSecret, dbSecret, dbSecret)
}
