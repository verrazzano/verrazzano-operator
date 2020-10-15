# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

NAME:=verrazzano-operator

DOCKER_IMAGE_NAME ?= ${NAME}-dev
TAG=$(shell git rev-parse HEAD)
DOCKER_IMAGE_TAG ?= local-$(shell git rev-parse --short HEAD)

CREATE_LATEST_TAG=0

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),push push-tag))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
	endif
	DOCKER_IMAGE_FULLNAME = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}
endif

HELM_CHART_VERSION = v0.0.0-${DOCKER_IMAGE_TAG}
OPERATOR_VERSION = ${DOCKER_IMAGE_TAG}
ifdef RELEASE_VERSION
	HELM_CHART_VERSION = ${RELEASE_VERSION}
	OPERATOR_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
K8S_NAMESPACE:=default
WATCH_NAMESPACE:=
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-operator
GO ?= GO111MODULE=on GOPRIVATE=github.com/verrazzano go
WKO_PATH = github.com/verrazzano/verrazzano-wko-operator
HELIDON_PATH = github.com/verrazzano/verrazzano-helidon-app-operator
COH_PATH = github.com/verrazzano/verrazzano-coh-cluster-operator
CRDGEN_PATH = github.com/verrazzano/verrazzano-crd-generator
VMO_PATH = github.com/verrazzano/verrazzano-monitoring-operator
VMO_CRD_PATH = k8s/crds
CRD_PATH = deploy/crds
HELM_CHART_NAME:=verrazzano
HELM_CHART_ARCHIVE_NAME = ${HELM_CHART_NAME}-${HELM_CHART_VERSION}.tgz

.PHONY: all
all: build

.PHONY: check
check: go-fmt go-vet go-ineffassign go-lint

#
# Go build related tasks
#
.PHONY: go-install
go-install: go-mod
	$(GO) install ./cmd/...

.PHONY: go-run
go-run: go-install
	$(GO) run cmd/verrazzano-operator/main.go --kubeconfig=${KUBECONFIG} --v=4 --watchNamespace=${WATCH_NAMESPACE} ${EXTRA_PARAMS}

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/ | grep -v /pkg/assets/) > error.txt
	if [ -s error.txt ]; then\
		cat error.txt;\
		rm error.txt;\
		exit 1;\
	fi
	rm error.txt

.PHONY: go-vet
go-vet:
	$(GO) vet $(shell go list ./... | grep -v github.com/verrazzano/verrazzano-operator/pkg/assets)

.PHONY: go-lint
go-lint:
	$(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status $(shell go list ./... | grep -v github.com/verrazzano/verrazzano-operator/pkg/assets)

.PHONY: go-ineffassign
go-ineffassign:
	$(GO) get -u github.com/gordonklaus/ineffassign
	ineffassign $(shell find . -name "*.go" | grep -v /vendor/ | grep -v /pkg/assets/)

.PHONY: go-mod
go-mod:
	# Generate Manifest file assets, needs to be done before go mod vendor
	$(GO) get -u github.com/jteeuwen/go-bindata/...
	go-bindata -pkg assets -o assets.go ./manifest.json ./dashboards/...
	mkdir -p pkg/assets
	mv assets.go pkg/assets/

	$(GO) mod vendor

	# go mod vendor only copies the .go files.  Also need
	# to populate the vendor folder with the .yaml files
	# that are required to define custom resources.

	# Obtain verrazzano-wko-operator version
	mkdir -p vendor/${WKO_PATH}/${CRD_PATH}
	cp `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-wko-operator`/${CRD_PATH}/*.yaml vendor/${WKO_PATH}/${CRD_PATH}

	# Obtain verrazzano-helidon-app-operator version
	mkdir -p vendor/${HELIDON_PATH}/${CRD_PATH}
	cp `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-helidon-app-operator`/${CRD_PATH}/*.yaml vendor/${HELIDON_PATH}/${CRD_PATH}

	# Obtain verrazzano-coh-cluster-operator version
	mkdir -p vendor/${COH_PATH}/${CRD_PATH}
	cp `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-coh-cluster-operator`/${CRD_PATH}/*.yaml vendor/${COH_PATH}/${CRD_PATH}

	# Obtain verrazzano-crd-generator version
	mkdir -p vendor/${CRDGEN_PATH}/${CRD_PATH}
	cp `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-crd-generator`/${CRD_PATH}/*.yaml vendor/${CRDGEN_PATH}/${CRD_PATH}

	# Obtain verrazzano-monitoring-operator version
	mkdir -p vendor/${VMO_PATH}/${VMO_CRD_PATH}
	cp `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-monitoring-operator`/${VMO_CRD_PATH}/*.yaml vendor/${VMO_PATH}/${VMO_CRD_PATH}

	# List copied CRD YAMLs
	ls vendor/${CRDGEN_PATH}/${CRD_PATH}

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: build
build: go-mod
	docker build --pull \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	docker push ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

	if [ "${CREATE_LATEST_TAG}" == "1" ]; then \
		docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest; \
		docker push ${DOCKER_IMAGE_FULLNAME}:latest; \
	fi

#
# Test-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v ./pkg/... ./cmd/...

.PHONY: coverage
coverage: unit-test
	./build/scripts/coverage.sh html

#
# Test-related tasks
#
CLUSTER_NAME = verrazzano-operator
CERTS = build/verrazzano-operator-cert
VERRAZZANO_NS = verrazzano-system
DEPLOY = build/deploy
K8RESOURCE = test/k8resource

.PHONY: integ-test
integ-test: create-cluster

	echo 'Create CRDs needed by the verrazzano-operator...'
	kubectl create -f vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanomanagedclusters_crd.yaml
	kubectl create -f vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanomodels_crd.yaml
	kubectl create -f vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanobindings_crd.yaml
	kubectl create -f vendor/${COH_PATH}/${CRD_PATH}/verrazzano.io_cohclusters_crd.yaml
	kubectl create -f vendor/${HELIDON_PATH}/${CRD_PATH}/verrazzano.io_helidonapps_crd.yaml
	kubectl create -f vendor/${WKO_PATH}/${CRD_PATH}/verrazzano.io_wlsoperators_crd.yaml
	kubectl create -f vendor/${VMO_PATH}/${VMO_CRD_PATH}/verrazzano-monitoring-operator-crds.yaml

	echo 'Deploy local cluster and required secret ...'
	# Create the local cluster secret with empty kubeconfig data.  This will force the verrazzano-operator
	# to use the in-cluster kubeconfig to access the managed cluster.
	kubectl create secret generic verrazzano-managed-cluster-local --from-literal=kubeconfig=""
	kubectl label secret verrazzano-managed-cluster-local k8s-app=verrazzano.oracle.com verrazzano.cluster=local
	kubectl apply -f ${K8RESOURCE}/local-vmc.yaml

	echo 'Load docker image for the verrazzano-operator...'
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

	echo 'Load docker image needed to install istio CRDs ...'
	docker pull container-registry.oracle.com/olcne/istio_kubectl:1.4.6
	docker image tag container-registry.oracle.com/olcne/istio_kubectl:1.4.6 container-registry.oracle.com/olcne/kubectl:1.4.6
	kind load docker-image --name ${CLUSTER_NAME} container-registry.oracle.com/olcne/kubectl:1.4.6

	echo 'Load docker image for the linux slim used for fake micro operators ...'
	docker pull container-registry.oracle.com/os/oraclelinux:7-slim
	kind load docker-image --name ${CLUSTER_NAME} container-registry.oracle.com/os/oraclelinux:7-slim

	echo 'Install the istio CRDs ...'
	kubectl create ns istio-system
	kubectl apply -f ./test/k8resource/istio-crds.yaml
	kubectl -n istio-system wait --for=condition=complete job --all --timeout=300s

	echo 'Create verrazzano operator required secrets ...'
	kubectl create namespace ${VERRAZZANO_NS}
	kubectl create secret generic verrazzano -n ${VERRAZZANO_NS} \
		--from-literal=password=admin --from-literal=username=admin

	echo 'Deploy verrazzano operator  ...'
	./test/create-deployment.sh ${DOCKER_IMAGE_NAME} ${DOCKER_IMAGE_TAG}
	kubectl apply -f ${DEPLOY}/deployment.yaml

	echo 'Run tests...'
	# ginkgo -v --keepGoing -cover test/integ/... || IGNORE=FAILURE

.PHONY: create-cluster
create-cluster:
ifdef JENKINS_URL
	./build/scripts/cleanup.sh ${CLUSTER_NAME}
endif
	echo 'Create cluster...'
	HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster \
		--name ${CLUSTER_NAME} \
		--wait 5m \
		--config=test/kind-config.yaml
	kubectl config set-context kind-${CLUSTER_NAME}
ifdef JENKINS_URL
	# disabled this - not needed since we are not running from inside docker any more
	# cat ${HOME}/.kube/config | grep server
	# this ugly looking line of code will get the ip address of the container running the kube apiserver
	# and update the kubeconfig file to point to that address, instead of localhost
	sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.IPAddress' | sed 's/"//g'`:6443|g" ${HOME}/.kube/config
	cat ${HOME}/.kube/config | grep server
endif

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name ${CLUSTER_NAME}

#
# Kubernetes-related tasks
#
.PHONY: k8s-deploy
k8s-deploy:
	mkdir -p ${DIST_DIR}/manifests
	cp -r k8s/manifests/* $(DIST_DIR)/manifests

	# Fill in Docker image and tag that's being tested
	sed -i.bak "s|<DOCKER-REPO-TAG>/<DOCKER-NAMESPACE-TAG>/verrazzano/verrazzano-operator:<IMAGE-TAG>|${DOCKER_IMAGE_FULLNAME}:$(DOCKER_IMAGE_TAG)|g" $(DIST_DIR)/manifests/verrazzano-operator-deployment.yaml
	sed -i.bak "s|namespace: default|namespace: ${K8S_NAMESPACE}|g" $(DIST_DIR)/manifests/verrazzano-operator-deployment.yaml
	sed -i.bak "s|--watchNamespace=default|--watchNamespace=${K8S_NAMESPACE}|g" $(DIST_DIR)/manifests/verrazzano-operator-deployment.yaml
	sed -i.bak "s|namespace: default|namespace: ${K8S_NAMESPACE}|g" $(DIST_DIR)/manifests/verrazzano-operator-serviceaccount.yaml
	kubectl delete -f ${DIST_DIR}/manifests
	kubectl apply -f ${DIST_DIR}/manifests

.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker push "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"
