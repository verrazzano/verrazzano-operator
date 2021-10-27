# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
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

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
DIST_DIR:=$(ROOT_DIR)/dist
K8S_NAMESPACE:=default
WATCH_NAMESPACE:=
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-operator
GO ?= GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GOPATH ?= $(shell go env GOPATH)
GOBIN:=$(GOPATH)/bin
VMO_PATH = github.com/verrazzano/verrazzano-monitoring-operator
VMO_CRD_PATH = k8s/crds
CRD_PATH = deploy/crds
HELM_CHART_NAME:=verrazzano
HELM_CHART_ARCHIVE_NAME = ${HELM_CHART_NAME}-${HELM_CHART_VERSION}.tgz
GO_BINDATA_VERSION ?= $(shell go list -m -f '{{.Version}}' 'github.com/go-bindata/go-bindata/v3' )


.PHONY: all
all: build

#
# Run all checks, convenient as a sanity-check before committing/pushing
#
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
	ineffassign $(shell go list ./...)

# find or download go-bindata
# download go-bindata if necessary
.PHONY: go-bindata
go-bindata:
ifeq (, $(shell command -v go-bindata))
	$(GO) get github.com/go-bindata/go-bindata/v3/...@${GO_BINDATA_VERSION}
	echo $(GOBIN)
	$(eval GO_BINDATA=$(GOBIN)/go-bindata)
else
	$(eval GO_BINDATA=$(shell command -v go-bindata))
endif
	@{ \
	set -eu; \
	ACTUAL_GO_BINDATA_VERSION=$$(${GO_BINDATA} --version | head -1 | awk '{print $$2}') ; \
	if [ "v$${ACTUAL_GO_BINDATA_VERSION}" != "${GO_BINDATA_VERSION}" ] ; then \
		echo  "Bad go-bindata version $${ACTUAL_GO_BINDATA_VERSION}, please install ${GO_BINDATA_VERSION}" ; \
	fi ; \
	}

.PHONY: assets
assets: go-bindata
	$(GO_BINDATA) -pkg assets -o assets.go ./manifest.json ./dashboards/...
	mkdir -p $(ROOT_DIR)/pkg/assets
	mv $(ROOT_DIR)/assets.go $(ROOT_DIR)/pkg/assets/

.PHONY: go-mod
go-mod: assets
	$(GO) mod download

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
OPERATOR_SETUP = test/operatorsetup

.PHONY: integ-test
integ-test: build create-cluster

	echo 'Create CRDs needed by the verrazzano-operator...'
	kubectl create -f `go list -f '{{.Dir}}' -m github.com/verrazzano/verrazzano-monitoring-operator`/${VMO_CRD_PATH}/verrazzano.io_verrazzanomonitoringinstances_crd.yaml

	echo 'Load docker image for the verrazzano-operator...'
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

	echo 'Load docker image for the linux slim used for fake micro operators ...'
	docker pull container-registry.oracle.com/os/oraclelinux:7-slim
	kind load docker-image --name ${CLUSTER_NAME} container-registry.oracle.com/os/oraclelinux:7-slim

	echo 'Create verrazzano operator required secrets ...'
	kubectl create namespace ${VERRAZZANO_NS}
	kubectl create secret generic verrazzano -n ${VERRAZZANO_NS} \
		--from-literal=password=admin --from-literal=username=admin

	echo 'Deploy verrazzano operator  ...'
	./test/create-deployment.sh ${DOCKER_IMAGE_NAME} ${DOCKER_IMAGE_TAG}
	kubectl apply -f ${DEPLOY}/deployment.yaml

	echo 'Run tests...'
	ginkgo -v --keepGoing -cover test/integ/... || IGNORE=FAILURE

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
	# Get the ip address of the container running the kube apiserver
	# and update the kubeconfig file to point to that address, instead of localhost
	sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.Networks[].IPAddress' | sed 's/"//g'`:6443|g" ${HOME}/.kube/config
	cat ${HOME}/.kube/config | grep server

	$$(X=$$(docker inspect $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }') | jq '.[].NetworkSettings.Networks' | grep -q kind ; echo $$?); if [[ ! $$X -eq "0" ]]; then docker network connect kind $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }'); fi)
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

