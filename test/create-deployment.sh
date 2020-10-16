#!/usr/bin/env bash

# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

BASE_DIR=$(cd $(dirname "$0"); cd ..; pwd -P)
DOCKER_IMAGE_NAME=$1
DOCKER_IMAGE_TAG=$2
CERTS=${BASE_DIR}/build/verrazzano-operator-cert
DEPLOY=${BASE_DIR}/build/deploy

mkdir -p "${DEPLOY}"

cat "${BASE_DIR}"/test/k8resource/deployment.yaml | sed -e "s|IMAGE_NAME:IMAGE_TAG|${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}|g" > "${DEPLOY}"/deployment.yaml

export CA_BUNDLE="`base64 -i "${CERTS}"/verrazzano-crt.pem | tr -d '\n'`"

sed -i -e "s|CA_BUNDLE|${CA_BUNDLE}|g" "${DEPLOY}"/deployment.yaml

