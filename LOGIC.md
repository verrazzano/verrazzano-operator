# Logic for Super Domain Operator

This page describes the desired logic for the SDO.  

** Important ** The details on this page are still subject to change.

## Triggers/Inputs

As stated in the [README](README.md), SDO watches `VerrazzanoApplicationBinding` CR's in
the management cluster.  This CRD has the following structure:

### VerrazzanoApplicationBinding

This CRD is used to describe a "binding".  A binding provides environment/instance-specific
information about an application, i.e. information that would be different in each 
deployment/instance.  A good example would be credentials and URLs used to connect to a 
database.  Bindings refer to (and extend) "models" (see below).

```
VerrazzanoApplicationBinding
    name                       the name of the binding
    version                    the version of the binding
    descripiton                a description of the binding
    model                      a reference to a TibubronApplicationModel
        name                   the name of the model file this binding refers to
        version                the version of the model file this binding refers to
    []-coherenceBindings
        name                   the name of the binding
        -clusterSize
        -serviceAccountName    may go away
        -store
            -logging
                -level
                -configFile
                -configMapName
        -maxHeap
        -jvmArgs
        -javaOpts
        -wkaRelease
        -wka
        -ports
        -env
        -annotations
        -labels
        -persistence
            -size
            -volume
        -snapshot
            -size
            -volume
        -jmx
            -enabled
            -replicas
            -maxHeap                            model or binding???
        -readinessProbe
            -initialDelaySeconds
            -periodSeconds
            -timeoutSeconds
            -successThreshold
            -failureThreshold
    []-databaseBindings
        name                   the name of the binding
        credentials            (type=database only) the name of the secret containing the database credentials
        url                    (type=database only) the database connect string (jdbc url) 
    []-atpBindings
        name                    the name of the binding, and also db name and display name for the ATP instance
        compartmentId           the OCID of the compartment in which the ATP DB exists or is to be provisioned
        -cpuCount               the number of ATP CPUs. default = 1
        -storageSizeTBs         the ATP storage size. default = 1
        -licenseType            the ATP license type (NEW or BYOL). default = BYOL
        -walletSecret           the name of the secret that contains/will contain the ATP wallet, default = name with "-wallet" appended
        -walletPassphraseSecret the name of the secret that contains/will contain the passphrase for the ATP wallet. default = name with "-passphrase" appended    
    []-ingressBindings
        name                   the name of the binding
        port                   the TCP/IP port number
        dnsName                the DNS name for the ingress
        -prefix                the prefix for the ingress. default = "/"
    []-helidonBindings
        name                   the name of the binding
        -replicas              the number of replicas for a Helidon application
    []-placement               a list of "placements
        name                   the name of the k8s cluster 
        []namespaces           a list of k8s namespaces in that cluster
            name               the name of the namespace
            []components       a list of components to place in that namespace on that cluster
                name           the name of the component
```

Note: A hyphen prefix denotes an optional element.

Here is an example of a `VerrazzanoApplicationBinding`:

```yaml
name: "Bob's Books Test Environment"
description: "The test environment for Bob's Books"
version: "1.0"
model:
  name: "Bob's Books"
  version: "1.0"
coherenceBindings:
  - name: "bobbys-coherence"
    cacheConfig: "bobbys-cache-config.xml"
    pofConfig: "bobbys-pof-config.xml"
databaseBindings:
  - name: "books"
    credentials: "booksdb-secret"
    url: "jdbc:mysql://mysql:3306/books"
atpBindings:
  - name: "books-atp"
    compartmentId: "COMPARTMENT_OCID"
    cpuCount: 1
    storageSizeTBs: 1
    licenseType: BYOL
    walletSecret: books-atp-wallet
    walletPassphraseSecret: books-atp-passphrase      
ingressBindings:
  - name: "roberts-ingress"
    port: "80"
    dnsName: "roberts-books.weblogick8s.org"
  - name: "bobbys-ingress"
    port: "31380"
    dnsName: "bobbys-books.weblogick8s.org"
helidonBindings:
  - name: bobbys-helidon-stock-application
    replicas: 1
placement: 
  - name: "cloud"
    namespaces:
      - name: "robert"
        components:
          - name: "roberts-helidon-stock-application"
          - name: "roberts-coherence"
  - name: "onprem"
    namespaces:
      - name: "bobby"
        components:
          - name: "bobbys-front-end"
          - name: "bobbys-helidon-stock-application"
          - name: "bobbys-coherence"
      - name: "bob"
        components:
          - name: "bobs-bookstore"
```


### VerrazzanoApplicationModel

This CRD describes an "application" which is made up of several components.  Components
may be WebLogic domains, Coherence clusters, Helidon microservices, or other generic 
k8s components/deployments.  The model captures information about the application which
does not vary between instances of the application.  For example, the WebLogic domain X
always talks to database Y, not matter how many times this application is deployed. 

In a particular instance/deployment of the application, e.g. the "test" instance, there 
may be different credentials to access database Y, but X always talks to Y.

Here is the structure of the `VerrazzanoApplicationModel`:

```
VerrazzanoApplicationModel
    name                                        the name of the model
    version                                     the version of the model
    description                                 a description of the model
    []-coherenceClusters
        name                                    the name of the component
        cohCRValues
            -imagePullSecrets                       the name of the image pull secret
            -role
            -cluster
            -store
                -cacheConfig
                -pof
                -storageEnabled
                -overrideConfig
                -logging
                    -level
                    -configFile
                    -configMapName
            -maxHeap
            -jvmArgs
            -javaOpts
            -annotations
            -labels
            -podManagementPolicy                    Value generated by SDO
            -revisionHistoryLimit                   Value generated by SDO
            -persistence
                -enabled
                -size
                -storageClass                       Value generated by SDO
                -dataSource                         Value generated by SDO
                -volumeMode                         Value generated by SDO
                -volumeName                         Value generated by SDO
                -selector                           Value generated by SDO
            -snapshot
                -enabled
                -size
                -storageClass                       Value generated by SDO
                -dataSource                         Value generated by SDO
                -volumeMode                         Value generated by SDO
                -volumeName                         Value generated by SDO
                -selector                           Value generated by SDO
            -management                             Value generated by SDO (including all sub-values)
                -port
                -ssl
                -enabled
                -secrets
                -keyStore
                -keyStorePasswordFile
                -keyPasswordFile
                -keyStoreAlgorithm
                -keyStoreProvider
                -keyStoreType
                -trustStore
                -trustStorePasswordFile
                -trustStoreAlgorithm
                -trustStoreProvider
                -trustStoreType
                -requireClientCert
            -metrics                             Value generated by SDO (including all sub-values)
                -port
                -ssl
                -enabled
                -secrets
                -keyStore
                -keyStorePasswordFile
                -keyPasswordFile
                -keyStoreAlgorithm
                -keyStoreProvider
                -keyStoreType
                -trustStore
                -trustStorePasswordFile
                -trustStoreAlgorithm
                -trustStoreProvider
                -trustStoreType
                -requireClientCert
            -jmx
                -enabled
                -replicas
                -maxHeap                            model or binding???
                -service                            Value generated by SDO (including all sub-values)
                    -enabled
                    -type
                    -domain
                    -loadBalancerIP
                    -annotations
                    -externalPort
            -service                               Value generated by SDO (including all sub-values)
                -enabled
                -type
                -domain
                -loadBalancerIP
                -annotations
                -externalPort
                -managementHttpPort
                -metricsHttpPort
            coherence
                image                               the image:tag for the coherence user artifacts
                -imagePullPolicy
            coherenceUtils
                image
                -imagePullPolicy
            -logCaptureEnabled                     Value generated by SDO
            -fluentd                               Value generated by SDO (including all sub-values)
                -image
                -imagePullPolicy
            -userArtifacts
                -image
                -imagePullPolicy
                -libDir
                -configDir 
        -metrics                                See definition in weblogicDomains
        -logging                                See definition in weblogicDomains
    []-weblogicDomains
        name                                    the name of the component
        -adminPort                              external port number for admin console
        -t3Port                                 external port number for T3
        domainCRValues                          domain CR values, can provide any valid Domain CR value
            domainHome                          Path to WebLogic domain home for the image
            image                               The docker image to use
            logHome                             Path to log home for WebLogic domain
            webLogicCredentialsSecret
                name                            the secret containing credentials for admin console
            imagePullSecrets
                name                            the name of secret for pulling docker images
            clusters
                clusterName                     the name of the cluster
        -connections                            a list of connections
           []-rest                              connections of type REST
              target                            the name of the target component
              -environmentVariableForHost       the dns name of the target component (its k8s service)
              -environmentVariableForPort       the port for the target component
            []-ingress
              -name                             the name of the ingress
            []-database
              target                            the name of the target component
              datasourceName                    the JDBC data source name for the database
            []-coherence
              target                            the name of the target component
              address                           the coherence cluster services address
        -metrics                                metrics configuration information
            endpoint                            the metrics endpoint (to be scraped)
            authSecret                          the name of the secret containing the credentials to access the metrics endpoint
            interval                            the interval to scrape metrics
        -logging                                logging configuration information
            type                                the type of logging - one of: exporter, filebeat, whatever (TBD)
            indexPattern                        the index-pattern to use (in elasticsearch)
    []-helidonApplications
        name                                    the name of the component
        image                                   the docker image:tag that runs the application
        -imagePullSecret                        name of secret containing credentials for pulling image
        []-connections                          See definition weblogicDomains
        -metrics                                See definition weblogicDomains
        -logging                                See definition weblogicDomains
```

Here is an example `VerrazzanoApplicationModel`:

```yaml
name: "Bob's Books"
description: "The model for Bob's Books"
version: 1.0
components:
  weblogicDomains:
    - name: "bobbys-front-end"
      domainCRValues:
        image: "container-registry.oracle.com/weblogick8s/bobbys-front-end"
        domainHome: "/u01/oracle/user_projects/domains/bobbys-front-end"
        logHome: "/u01/oracle/user_projects/domains/bobbys-front-end/logs"
      connections:
        rest:
          - target: "bobbys-helidon-stock-application"
            environmentVariableForHost: "HELIDON_HOSTNAME"
            environmentVariableForPort: "HELIDON_PORT"
        ingress:
          - name: "bobbys-ingress"
    - name: "bobs-bookstore-order-manager"
      domainCRValues:
        image: "container-registry.oracle.com/weblogick8s/bobs-bookstore-order-manager"
        domainHome: "/u01/oracle/user_projects/domains/bobbys-order-manager"
        logHome: "/u01/oracle/user_projects/domains/bobbys-order-manager/logs"
      connections:
      - type: "database"
        target: "books"
        datasourceName: "books"
      metrics:
      - endpoint: "/wls/metrics"
        authSecret: "bobs-bookstore-weblogic-credentials"
        interval: "42s"
      logging:
        type: exporter
        indexPattern: "bobs-books"
  helidonApplications:
    - name: "bobbys-helidon-stock-application"
      image: "container-registry.oracle.com/weblogick8s/bobbys-helidon-stock-application"
      connections:
        coherence:
          - target: "bobbys-coherence"
            address: "bobbys-coherence-headless"
        rest:
          - target: "bobs-book-store-order-manager"  
            environmentVariableForHost: "BACKEND_HOSTNAME"
            environmentVariableForPort: "BACKEND_PORT"
    - name: "roberts-helidon-stock-app"
      image: "container-registry.oracle.com/weblogick8s/roberts-helidon-stock-application"
      connections:
        ingress:
          - name: "ingress2"
        rest:
          - target: "bobs-book-store-order-manager"
            environmentVariableForHost: "BACKEND_HOSTNAME"
            environmentVariableForPort: "BACKEND_PORT"
        coherence:
          - target: "roberts-coherence"
  coherenceClusters:
    - name: "roberts-coherence"
      image: "container-registry.oracle.com/weblogick8s/roberts-coherence:768174"
    - name: "bobbys-coherence"
      iamge: "container-registry.oracle.com/weblogick8s/bobbys-coherence:768174"
```

## Logic 

The combination of a model and and binding produces an instance of an application. 
Both the model and binding are meant to be sparse - they contain only the information 
that is needed from the user.  Anything that SDO can infer or default is omited from 
these files. 

When a binding CR is created, SDO should proceed as follows:

```
// pseudocode

// phase 1 - build the "to be" state
build a list "components" from the model, include all details under each component

iterate over the bindings file
for each binding:
    if you can find the matching connection in the "to be" state
        update the connection with the information in the binding
    else
        abort
    
for each placement:
    if you can find the matching component in the "to be" state
        update the component with its placement information
    else
        abort
        
// phase 2 - build the "current" state

iterate over the "to be" state
for each component:
    if that component already exists:
        if the config matches the "to be" state
            nothing to do, continue
    else
        mark the component to be created or updated (which is pretty much the same thing)

// phase 3 - initiate updates

iterate over the "to be" state
for each component:
    if the component is marked for create/update
        create the CR that describes this component (* see below *)

```

The pseudocode above provides the high level logic for SDO.  Most of the detail is in the
logic for each type of object, which is listed below. 


### WebLogic domain

For components of type "WebLogic domain" - the model/binding is converted into the CR as 
follows: 

```
// pseudocode

if the k8s cluster does not have the micro-operator wls-operator deployed:
    mark this k8s cluster as requiring deployment of wls-operator
    
if the k8s cluster does not have the micro-operator wls-domain deployed:
    mark this k8s cluster a requiring deployment of wls-domain

if there are no other domains in the same k8s cluster (see placement):
    mark this domain as requiring WebLogic operator installation

if there are no other domains in the same namespace (see placement):
    mark this domain as requiring the WebLogic operator targetNamespace list to be updated

if the imagePullSecret was specified:
    mark this domain as requiring a secret to be created

if the target namespace does not exist:
    mark this domain as requiring a namespace to be created

create a domain CR as follows - {{ ... }} indicates where values are substituted in

---
apiVersion: "weblogic.oracle/v4"
kind: Domain
metadata:
  name: {{ component.name  }}
  namespace: {{ component.namespace }}
  labels:
    weblogic.resourceVersion: domain-v2
    weblogic.domainUID: {{ component.name }}
spec:
  domainHome: {{ **NEED** }}
  domainHomeInImage: false
  image: {{ component.image }}
  imagePullPolicy: "IfNotPresent"
  {{ if component.imagePullSecret is specified }}
  imagePullSecrets:
  - name: {{ component.imagePullSecret }}
  {{ end if }}
  webLogicCredentialsSecret: 
    name: {{ component.name }}-domain-credentials
  includeServerOutInPodLog: true
  logHomeEnabled: true
  logHome: {{ **NEED** }}
  serverStartPolicy: "IF_NEEDED"
  serverPod:
    env:
    - name: JAVA_OPTIONS
      value: "-Dweblogic.StdoutDebugEnabled=false"
    - name: USER_MEM_ARGS
      value: "-XX:+UseContainerSupport -Djava.security.egd=file:/dev/./urandom "
    {{ if any env vars are specified - e.g. in a connection of type rest, add them here, one example: }}
    - name: {{ environment_variable_for_host }}
      value: {{ whatever k8s service name "we" (SDO) allocated that component }}
    {{ end if }}
  adminServer:
    serverStartState: "RUNNING"
    {{ if component.adminPort or component.t3port are specified }}
    adminService:
      channels:
       - channelName: default
         nodePort: {{ component.adminPort }}
       - channelName: T3Channel
         nodePort: {{ component.t3port }}
    {{ end if }}
---

NOTE: NodePorts must be unique across the entire k8s cluster

NOTE: Based on conversation with Ryan 6/25, looks like we need to explicitly list all the clusters and provide
`serverStartState` for each one -- so this will mean we need to get that list from the model.... will update
as new information becomes available. 

// defaulted/infered values

the domain will be configured to send logs to this application's VMI-provided elasticsearch
    if component.logging.type is specified:
        if it is exporter:
            udpate the WebLogicLoggingExporter.yaml in `<domain_home>/config` with the right host/port
            it looks like this: 

            publishHost:  {{ the VMI elasticsearch host }}
            publishPort:  {{ the VMI elasticsearch port }}
            domainUID:  {{ component.name }}
            weblogicLoggingExporterEnabled: true
            weblogicLoggingIndexName:  {{  component.logging.index-pattern if specified, else "wls-"component.name }}
            weblogicLoggingExporterSeverity:  Notice
            weblogicLoggingExporterBulkSize: 1
            weblogicLoggingExporterFilters:
            - filterExpression:  'severity > Warning'

        if it is anything else:
            ignore for now

this application's VMI-provided prometheus will be configured to scrape metrics from each pod in this domain
    if component.metrics is specified:
        create a ServiceMonitor object (??? check with Sandeep ???) 
        it looks like this: 

        ** NEED SAMPLE **

iterate over connections:
    if connection is of type ingress:
        create an istio virtual service for this domain (see below)
        create an istio gateway for this service (see below)

    if connection is of type rest:
        if target is in a different k8s cluster (see placement):
            create istio service entry for the remote service
            if istio service gateway for the target k8s cluster does not exist:
                create istio service gateway for the target k8s cluster

    if connection is of type database:
        ignore it for now - we are faking it
        // in real life, we would:
        // generate a sit-cfg file to update the named datasource with the 
        // details from the matching database connection 

    if connection is of type atp:
        if the atp instance exists:
            do nothing
        else:
            provision the atp database (see below)
            copy the atp wallet secret to the target namespace/cluster                
        

// check pre-reqs (from above)

if marked for namespace creation:
    create CR to tell k8s-micro-operator to create the namespace and label it for envoy injection

if marked for secret creation: 
    create CR to tell k8s-micro-operator to create the secret

if marked for WebLogic operator installation:
    create CR to tell WKO-micro-operator to install WKO

if marked for WebLogic operator target namespace update:
    create CR to tell WKO-micro-operator to update targetNamespace list

// now create the domain

create the CR to tell domain-micro-operator to create the actual domain CR in the target cluster/namespace


```

####  Virtual Service

The Istio Virtual Service for a domain looks like this: 

```
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: {{ component.name }}-virtualservice
  namespace: {{ component.namespace }}
spec:
  gateways:
  - {{ component.name }}-gateway
  hosts:
  - '{{ component.connection[ingress].dns_name }}'
  http:
  - match:
    - uri:
        prefix: /console
    - port: 7001
    route:
    - destination:
        host: {{ the service name for the admin server }}.{{ component.namespace }}.svc.cluster.local
        port:
          number: 7001
  tcp:
  - match:
    - port: {{ component.connection[ingress].port }}
    route:
    - destination:
        host: {{ the service name for the cluster service }}.{{ component.namespace }}.svc.cluster.local
        port:
          number: {{ component.connection[ingress].port }}
```


#### Gateway

The Istio Gateway looks like this: 

```
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: {{ component.name }}-gateway
  namespace: {{ component.namespace }}
spec:
  selector:
    istio: ingressgateway
  servers:
  - hosts:
    - '*'
    port:
      name: http
      number: 80
      protocol: HTTP
  - hosts:
    - '*'
    port:
      name: tcp
      number: {{ component.connection[ingress].port }}
      protocol: TCP
```

NOTE: this might not be 100% correct ^^^

### Coherence

```
// pseudocode

if the k8s cluster does not have the micro-operator coh-cluster deployed:
    mark this k8s cluster as requiring deployment of coh-cluster
    
```

### Helidon-MP

```
// 
pseudocode

if the k8s cluster does not have the micro-operator helidon-app deployed:
    mark this k8s cluster as requiring deployment of helidon-app

```


### Generic



### ATP Database

To provision an ATP database, we need to create two objects: 

- ServiceInstance
- ServiceBinding

For the ServiceInstance, we take the example below, and merge in the values from the model/binding:

```yaml
apiVersion: servicecatalog.k8s.io/v1beta1
kind: ServiceInstance
metadata:
  name: osb-atp-demo-1
spec:
  clusterServiceClassExternalName: atp-service
  clusterServicePlanExternalName: standard
  parameters:
    name: osbdemo
    compartmentId: "CHANGE_COMPARTMENT_OCID_HERE"
    dbName: osbdemo
    cpuCount: 1
    storageSizeTBs: 1
    licenseType: BYOL
  parametersFrom:
    - secretKeyRef:
        name: atp-secret
        key: password
```

* metadata.name, spec.parameters.name and spec.parameters.dbName all come from the name in the binding. 
* fields that are not mentioned in the binding are hardcoded/defaulted to the values shown in the example above. 
* the secretKeyRef.name comes from the atp-wallet-secret and the secretKeyRef.key comes from the value in 
  the secret pointed to by atp-passphrase-secret.
  

For the ServiceBinding, we take the example below and merge in the values from the model/binding: 

```yaml
apiVersion: servicecatalog.k8s.io/v1beta1
kind: ServiceBinding
metadata:
  name: atp-demo-binding
spec:
  instanceRef:
    name: books-atp
  parametersFrom:
    - secretKeyRef:
        name: books-atp-secret
        key: walletPassword
```

