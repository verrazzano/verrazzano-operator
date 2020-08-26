
# Verrazzano Operator

The Verrazzano Operator is the Kubernetes operator that runs in the Verrazzano Management Cluster,
watches local CRDs for models/bindings, and launches micro operators into Verrazzano Managed Clusters.

## Release

A github release is created on every successful build on the master branch. The release version is obtained by incrementing the minor version of last release by 1. On a successful release, this repo
- publises a Docker image: `container-registry.oracle.com/verrazzano/verrazzano-operator:<release-version>`

## Building

Go build:
```
make go-install
```

Docker build:
```
export ACCESS_USERNAME=<username with read access to github verrazzano project>
export ACCESS_PASSWORD=<password for account with read access to github verrazzano project>
make build
```

Docker push:
```
make push
```

## Running

First, as a one-time operation, create relevant CRDs in the cluster where you'll be running the Verrazzano Operator:

```
kubectl apply -f vendor/github.com/verrazzano/verrazzano-crd-generator/deploy/crds/verrazzano_v1beta1_verrazzanomanagedcluster_crd.yaml
kubectl apply -f vendor/github.com/verrazzano/verrazzano-crd-generator/deploy/crds/verrazzano_v1beta1_verrazzanobinding_crd.yaml
kubectl apply -f vendor/github.com/verrazzano/verrazzano-crd-generator/deploy/crds/verrazzano_v1beta1_verrazzanomodel_crd.yaml
```

### Running locally

While developing, it's usually most efficient to run the Verrazzano Operator as an out-of-cluster process,
pointing it to your Kubernetes cluster:

```
export KUBECONFIG=<your_kubeconfig>
make go-run
```

### Running in a Kubernetes Cluster

```
kubectl apply -f ./k8s/manifests/verrazzano-operator-serviceaccount.yaml
kubectl apply -f ./k8s/manifests/verrazzano-operator-deployment.yaml
```

**Note:** - if you don't intend to use the latest official Docker image, fill in your own Docker image in
`verrazzano-operator-deployment.yaml` above.

## Demo

First, install the Verrazzano Operator using one of the approaches above.  Now, make it aware of 2 Managed Clusters (you
can set these to be same as the cluster running the Super Domain Operator if you are short on clusters:

```
# For now, make sure that your kubeconfig files are literally called "kubeconfig"
export MANAGED1_KUBECONFIG=<my_kubeconfig1>
export MANAGED2_KUBECONFIG=<my_kubeconfig2>

kubectl delete secret managed-cluster1
kubectl create secret generic managed-cluster1 --from-file=$MANAGED1_KUBECONFIG
kubectl apply -f k8s/examples/managed-cluster1.yaml

kubectl delete secret managed-cluster2
kubectl create secret generic managed-cluster2 --from-file=$MANAGED2_KUBECONFIG
kubectl apply -f k8s/examples/managed-cluster2.yaml
```

Then create a VerrazzanoBinding:
```
kubectl apply -f k8s/examples/app-binding.yaml
```

And view the artifacts that have been created in Managed Cluster1:

```
kubectl get pod,serviceaccounts --all-namespaces -l 'k8s-app=verrazzano.io'
NAMESPACE                       NAME                                                   READY   STATUS    RESTARTS   AGE
monitoring                      pod/verrazzano-prom-pusher-69b57c4986-snvgb            1/1     Running   0          18m
verrazzano-bobs-books-binding   pod/verrazzano-coh-cluster-operator-7dcc8c48d6-4qmwg   1/1     Running   0          18m
verrazzano-bobs-books-binding   pod/verrazzano-helidon-app-operator-6b84c5dbc9-2shgq   1/1     Running   0          18m
verrazzano-bobs-books-binding   pod/verrazzano-wko-operator-846db5747-szzgz            1/1     Running   0          18m

NAMESPACE                       NAME                                           SECRETS   AGE
bob                             serviceaccount/verrazzano-bobs-books-binding   1         18m
bobby                           serviceaccount/verrazzano-bobs-books-binding   1         18m
monitoring                      serviceaccount/verrazzano-bobs-books-binding   1         18m
verrazzano-bobs-books-binding   serviceaccount/verrazzano-bobs-books-binding   1         18m
wls-operator                    serviceaccount/verrazzano-bobs-books-binding   1         18m
```

And Managed Cluster2:

```
kubectl get pod,serviceaccounts --all-namespaces -l 'k8s-app=verrazzano.io'
NAMESPACE                       NAME                                                   READY   STATUS    RESTARTS   AGE
monitoring                      pod/verrazzano-prom-pusher-969746f55-swvrz             1/1     Running   0          19m
verrazzano-bobs-books-binding   pod/verrazzano-coh-cluster-operator-59684b8bc-j7467    1/1     Running   0          19m
verrazzano-bobs-books-binding   pod/verrazzano-helidon-app-operator-5955d6b874-246gh   1/1     Running   0          19m

NAMESPACE                       NAME                                           SECRETS   AGE
monitoring                      serviceaccount/verrazzano-bobs-books-binding   1         19m
robert                          serviceaccount/verrazzano-bobs-books-binding   1         19m
verrazzano-bobs-books-binding   serviceaccount/verrazzano-bobs-books-binding   1         19m
```

## Development

### Running Tests

To run unit tests:

```
make unit-test
```

To run integration tests:

```
make integ-test
```
