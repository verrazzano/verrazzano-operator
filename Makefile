# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

NAME:=verrazzano-operator

DOCKER_IMAGE_NAME ?= ${NAME}-dev
TAG=$(shell git rev-parse HEAD)
DOCKER_IMAGE_TAG = ${TAG}

CREATE_LATEST_TAG=0

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),push release))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
	endif
	DOCKER_IMAGE_FULLNAME = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}
endif

HELM_CHART_VERSION = v0.0.0-${TAG}
OPERATOR_VERSION = ${TAG}
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
CRD_PATH = deploy/crds
DIST_OBJECT_STORE_NAMESPACE:=stevengreenberginc
DIST_OBJECT_STORE_BUCKET:=verrazzano-helm-chart
HELM_CHART_NAME:=verrazzano
HELM_CHART_ARCHIVE_NAME = ${HELM_CHART_NAME}-${HELM_CHART_VERSION}.tgz

.PHONY: all
all: build

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
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/)

.PHONY: go-vet
go-vet:
	echo $(GO) vet $(shell go list ./... | grep -v /vendor/)

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
# Tests-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v ./pkg/... ./cmd/...

.PHONY: coverage
coverage: unit-test
	./build/scripts/coverage.sh html

.PHONY: integ-test
integ-test: go-install
	$(GO) test -v ./test/integ/ -timeout 30m --kubeconfig=${KUBECONFIG} --namespace=${K8S_NAMESPACE} --runid=${INTEG_RUN_ID}

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

.PHONY chart-build:
chart-build: go-mod
	echo "Creating helm chart archive."
	rm -rf ${DIST_DIR}
	mkdir ${DIST_DIR}
	mkdir ${DIST_DIR}/charts
	mkdir ${DIST_DIR}/crds
	cp -r chart/Chart.yaml $(DIST_DIR)/
	cp -r chart/templates $(DIST_DIR)/
	cp -r vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanobindings_crd.yaml ${DIST_DIR}/crds/verrazzano.io_verrazzanobindings_crd.yaml
	cp -r vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanomodels_crd.yaml ${DIST_DIR}/crds/verrazzano.io_verrazzanomodels_crd.yaml
	cp -r vendor/${CRDGEN_PATH}/${CRD_PATH}/verrazzano.io_verrazzanomanagedclusters_crd.yaml ${DIST_DIR}/crds/verrazzano.io_verrazzanomanagedclusters_crd.yaml
	cp -r chart/NOTES.txt  $(DIST_DIR)/
	cp -r chart/values.yaml  $(DIST_DIR)/
	# Fill in tag version that's being built
	sed -i.bak -e "s/latest/${HELM_CHART_VERSION}/g" $(DIST_DIR)/Chart.yaml
	sed -i.bak -e "s/OPERATOR_VERSION/${OPERATOR_VERSION}/g" -e "s/OPERATOR_IMAGE_NAME/${DOCKER_IMAGE_NAME}/g" $(DIST_DIR)/values.yaml

	rm -rf archive
	mkdir archive
	tar cvzf archive/${HELM_CHART_ARCHIVE_NAME} -C ${DIST_DIR}/ .
	mv archive/${HELM_CHART_ARCHIVE_NAME} ${DIST_DIR}/
	rm -rf archive
	echo "Helm chart archive created - ${HELM_CHART_ARCHIVE_NAME}."

.PHONY chart-publish:
chart-publish:
	echo "Publishing Helm chart to OCI object storage"
	export OCI_CLI_SUPPRESS_FILE_PERMISSIONS_WARNING=True
	echo ${HELM_CHART_VERSION} > latest
	helm repo index --url https://objectstorage.us-phoenix-1.oraclecloud.com/n/${DIST_OBJECT_STORE_NAMESPACE}/b/${DIST_OBJECT_STORE_BUCKET}/o/${HELM_CHART_VERSION}/ ${DIST_DIR}/
	oci os object put --force --namespace ${DIST_OBJECT_STORE_NAMESPACE} -bn ${DIST_OBJECT_STORE_BUCKET} --name ${HELM_CHART_VERSION}/index.yaml --file ${DIST_DIR}/index.yaml
	oci os object put --force --namespace ${DIST_OBJECT_STORE_NAMESPACE} -bn ${DIST_OBJECT_STORE_BUCKET} --name ${HELM_CHART_VERSION}/${HELM_CHART_ARCHIVE_NAME} --file ${DIST_DIR}/${HELM_CHART_ARCHIVE_NAME}
	oci os object put --force --namespace ${DIST_OBJECT_STORE_NAMESPACE} -bn ${DIST_OBJECT_STORE_BUCKET} --name latest --file latest
	echo "Published Helm chart to https://objectstorage.us-phoenix-1.oraclecloud.com/n/${DIST_OBJECT_STORE_NAMESPACE}/b/${DIST_OBJECT_STORE_BUCKET}/o/${HELM_CHART_VERSION}/${HELM_CHART_ARCHIVE_NAME}"
	
	echo "Check and upload release assets to github."
	@rm -rf response.txt
	@curl -ksH "Authorization: token ${GITHUB_API_TOKEN}" "https://api.github.com/repos/verrazzano/verrazzano-operator/releases/tags/${HELM_CHART_VERSION}" -o response.txt
	@while [ ! -f response.txt ]; do sleep 1; done;
	@cat response.txt
	@set -e; \
	msg=$$(jq -r .message response.txt); \
	if [ "$$msg" == "Not Found" ]; then \
		echo "No release found associated with version ${HELM_CHART_VERSION}, skipping uploading release assets."; \
	else \
		id=$$(jq -r .id response.txt); \
		if [ -z "$$id" ]; then \
			echo "Error: Failed to get release id for tag: ${HELM_CHART_VERSION}."; \
			exit 1; \
		else \
			existingAssetId=$$(jq -r '.assets[] | select(.name == ("${HELM_CHART_ARCHIVE_NAME}")) | .id' response.txt); \
			if [ ! -z "$$existingAssetId" ]; then \
				echo "Release asset with name ${HELM_CHART_ARCHIVE_NAME} already exists with ID $$existingAssetId for release ${HELM_CHART_VERSION}. Deleting..."; \
				status=$$(curl -w '%{http_code}' -s -k -X DELETE -H "Authorization: token ${GITHUB_API_TOKEN}" "https://api.github.com/repos/verrazzano/verrazzano-operator/releases/assets/$$existingAssetId"); \
				if [ "$$status" != "204" ]; then \
					echo "Unable to delete existing asset with name ${HELM_CHART_ARCHIVE_NAME} for release ${HELM_CHART_VERSION}, invalid status ${status}, aborting.."; \
					echo "$$status"; \
					exit 1; \
				fi; \
				echo "Deleted asset with name ${HELM_CHART_ARCHIVE_NAME} for release ${HELM_CHART_VERSION}."; \
			fi; \
			echo "Uploading ${HELM_CHART_ARCHIVE_NAME} to release ${HELM_CHART_VERSION}."; \
			status=$$(curl -s -o /dev/null -w '%{http_code}' --data-binary @"${DIST_DIR}/${HELM_CHART_ARCHIVE_NAME}" -H "Authorization: token ${GITHUB_API_TOKEN}" -H "Content-Type: application/octet-stream" "https://uploads.github.com/repos/verrazzano/verrazzano-operator/releases/$$id/assets?name=${HELM_CHART_ARCHIVE_NAME}"); \
			if [ "$$status" != "201" ]; then \
				echo "Unable to upload asset with name ${HELM_CHART_ARCHIVE_NAME} for release ${HELM_CHART_VERSION}, invalid status ${status}, aborting.."; \
				echo "$$status"; \
				exit 1; \
			fi; \
			echo "Uploaded ${HELM_CHART_ARCHIVE_NAME} to release ${HELM_CHART_VERSION}."; \
		fi; \
	fi; \
	set +e

	rm -rf response.txt
	rm -rf ${DIST_DIR}

.PHONY release-version:
release-version:
	@set -e; \
	if [ ! -z "${RELEASE_VERSION}" ]; then \
		echo "Using input release version ${RELEASE_VERSION}."; \
		version=$$(echo "${RELEASE_VERSION}"); \
	else \
		echo "Getting latest tag."; \
		rm -rf response.txt; \
		curl -ksH "Authorization: token ${GITHUB_API_TOKEN}" "https://api.github.com/repos/verrazzano/verrazzano-operator/tags" -o response.txt; \
		while [ ! -f response.txt ]; do sleep 1; done; \
		msg=$$(jq -r 'if type=="array" then "" else .message end' response.txt); \
		if [ "$$msg" == "Not Found" ]; then \
			echo "No existing tag found. Creating release with version v0.1.0."; \
			version="v0.1.0"; \
		else \
			latest=$$(jq -r '[.[]|.name][0]' response.txt); \
			if [ -z "$$latest" ]; then \
				echo "Error: Failed to get latest tag. Aborting.."; \
				exit 1; \
			else \
				echo "Version for latest tag is $$latest."; \
				major=$$(echo $$latest | cut -d. -f1); \
				minor=$$(echo $$latest | cut -d. -f2); \
				revision=$$(echo $$latest | cut -d. -f3); \
				revision=$$(expr $$revision + 1); \
				version=$$(echo "$$major.$$minor.$$revision"); \
				echo "New release version $$version."; \
			fi; \
		fi; \
	fi; \
	set +e; \
	echo "$$version" > chart/latest

.PHONY release-image:
release-image: release-version
	RELEASE_VERSION=$$(cat chart/latest); \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$RELEASE_VERSION"; \
	docker pull ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}; \
	docker tag ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:$$RELEASE_VERSION; \
	docker push ${DOCKER_IMAGE_FULLNAME}:$$RELEASE_VERSION;

.PHONY github-release:
github-release: release-image
	make chart-build RELEASE_VERSION=`cat chart/latest`
	touch chart/index.yaml
	cp chart/index.yaml ${DIST_DIR}/
	helm repo index --merge ${DIST_DIR}/index.yaml --url https://github.com/verrazzano/verrazzano-operator/releases/download ${DIST_DIR}/
	cp ${DIST_DIR}/index.yaml chart/.

	@set -e; \
	echo "Updating index.yaml in github repo."; \
	git clone -b ${RELEASE_BRANCH} https://github.com/verrazzano/verrazzano-operator; \
	cp chart/index.yaml verrazzano-operator/chart/index.yaml; \
	cp chart/latest verrazzano-operator/chart/latest; \
	cd verrazzano-operator; \
	git config user.email "${GITHUB_RELEASE_EMAIL}"; \
	git config user.name "${GITHUB_RELEASE_USERID}"; \
	git add -f chart/index.yaml; \
	git add -f chart/latest; \
	git commit -m "[automatic helm release] Release $$(cat chart/latest)"; \
	git push; \
	set +e; \
	echo "Updated index.yaml in github repo.";
	rm -rf verrazzano-operator

	@set -e; \
	RELEASE_VERSION=$$(cat chart/latest); \
	echo "Creating release $$RELEASE_VERSION in github."; \
	request="{\"tag_name\": \"$$RELEASE_VERSION\",\"target_commitish\": \"${RELEASE_BRANCH}\",\"name\": \"$$RELEASE_VERSION\",\"body\": \"${RELEASE_DESCRIPTION}\"}"; \
	echo $$request; \
	status=$$(curl -s -o /dev/null -w '%{http_code}' --data "$$request" -H "Authorization: token ${GITHUB_API_TOKEN}" -H "Content-Type: application/json" "https://api.github.com/repos/verrazzano/verrazzano-operator/releases"); \
	if [ "$$status" != "201" ]; then \
		echo "Unable to create release with name $$RELEASE_VERSION, invalid status ${status}, aborting.."; \
		echo "$$status"; \
		exit 1; \
	fi; \
	set +e; \
	echo "Release $$RELEASE_VERSION created in github.";

.PHONY release:
release: github-release
	make chart-publish RELEASE_VERSION=`cat chart/latest`