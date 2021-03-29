#!/bin/bash
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# This script will run the verrazzano-operator outside of the cluster, for local debugging/testing
# purposes.
#
# Pre-requisites:
# - golang is installed as described in the README
# - the verrazzano installer is cloned from github in a parallel directory named to this repo (https://github.com/verrazzano/verrazzano)
# - kubectl is installed
# - KUBECONFIG is pointing to a valid Kubernetes cluster
# - If on corporate network set proxy environment variables
#
# Simply run
#
#    sh run-vo.sh [profile-name]
#
# where
#
#    profile-name = [dev|prod]
#
# By default, profile-name="dev"
#
# When executed, the script will
#
# - build the operator
# - set up the required environment variables for the operator
# - scale down the in-cluster monitoring operator to 0
# - Execute the local operator
# - Once the local operator is terminated (hit Ctrl-C 2x), scale up the in-cluster operator to 1 replica again

profile=${1-:"dev"}

# Customize these to provide the location of your verrazzano and verrazzano repos
#export WORK=${HOME}/Projects
#export VERRAZZANO_REPO=${WORK}/vz/src/github.com/verrazzano/verrazzano
export VERRAZZANO_OPERATOR_REPO=$(pwd)
export VERRAZZANO_REPO=${VERRAZZANO_OPERATOR_REPO}/../verrazzano
export VERRAZZANO_VALUES=platform-operator/helm_config/charts/verrazzano/values.yaml

echo "Building and installing the verrazzano-operator."
cd ${VERRAZZANO_OPERATOR_REPO}
make go-install
echo ""

echo "Stopping the in-cluster verrazzano-operator."
set -x
kubectl scale deployment verrazzano-operator --replicas=0 -n verrazzano-system
set +x
echo ""

# Extract the images required by verrazzano-operator from values.yaml into environment variables.
export COH_MICRO_IMAGE=$(grep cohMicroImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export HELIDON_MICRO_IMAGE=$(grep helidonMicroImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export WLS_MICRO_IMAGE=$(grep wlsMicroImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export PROMETHEUS_PUSHER_IMAGE=$(grep prometheusPusherImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export NODE_EXPORTER_IMAGE=$(grep nodeExporterImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | head -1 | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//' | sort -u)
export FILEBEAT_IMAGE=$(grep filebeatImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export JOURNALBEAT_IMAGE=$(grep journalbeatImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export WEBLOGIC_OPERATOR_IMAGE=$(grep weblogicOperatorImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export FLUENTD_IMAGE=$(grep fluentdImage ${VERRAZZANO_REPO}/${VERRAZZANO_VALUES} | cut -d':' -f2,3 | sed -e 's/^[[:space:]]*//')
export COH_MICRO_REQUEST_MEMORY="28Mi"
export HELIDON_MICRO_REQUEST_MEMORY="24Mi"
export WLS_MICRO_REQUEST_MEMORY="32Mi"
export GRAFANA_REQUEST_MEMORY="48Mi"
export PROMETHEUS_REQUEST_MEMORY="128Mi"
export KIBANA_REQUEST_MEMORY="192Mi"

export ES_INGEST_NODE_REQUEST_MEMORY="2.5Gi"
export ES_DATA_NODE_REQUEST_MEMORY="4.8Gi"

echo "Running ${profile} profile"
if [ "${profile}" == "dev" ]; then
  # Dev profile
  export ES_MASTER_NODE_REQUEST_MEMORY="1Gi"
  export ES_MASTER_NODE_REPLICAS="1"
  export ES_DATA_NODE_REPLICAS="0"
  export ES_INGEST_NODE_REPLICAS="0"
  export USE_SYSTEM_VMI=true
  export ES_DATA_STORAGE=""
  export PROMETHEUS_DATA_STORAGE="50Gi"  # Keep for now?
  export GRAFANA_DATA_STORAGE="50Gi"     # Keep for now?
else
  # Prod profile
  export ES_MASTER_NODE_REQUEST_MEMORY="1.4Gi"
  export ES_MASTER_NODE_REPLICAS="3"
  export ES_DATA_NODE_REPLICAS="2"
  export ES_INGEST_NODE_REPLICAS="1"
  export USE_SYSTEM_VMI=false
  export ES_DATA_STORAGE="50Gi"
  export PROMETHEUS_DATA_STORAGE="50Gi"
  export GRAFANA_DATA_STORAGE="50Gi"
fi

# Extract the API server realm from values.yaml.
export API_SERVER_REALM=$(grep apiServerRealm ${VERRAZZANO_REPO}/platform-operator/helm_config/charts/verrazzano/values.yaml | cut -d':' -f2 | sed -e 's/^[[:space:]]*//')

# Extract the Verrazzano system ingress IP from the NGINX ingress controller status.
export VERRAZZANO_SYSTEM_INGRESS_IP=$(kubectl get svc -n ingress-nginx ingress-controller-nginx-ingress-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

cat <<EOF
Variables:
USE_SYSTEM_VMI=${USE_SYSTEM_VMI}
COH_MICRO_IMAGE=$COH_MICRO_IMAGE
HELIDON_MICRO_IMAGE=$HELIDON_MICRO_IMAGE
WLS_MICRO_IMAGE=$WLS_MICRO_IMAGE
PROMETHEUS_PUSHER_IMAGE=$PROMETHEUS_PUSHER_IMAGE
NODE_EXPORTER_IMAGE=$NODE_EXPORTER_IMAGE
FILEBEAT_IMAGE=$FILEBEAT_IMAGE
JOURNALBEAT_IMAGE=$JOURNALBEAT_IMAGE
WEBLOGIC_OPERATOR_IMAGE=$WEBLOGIC_OPERATOR_IMAGE
FLUENTD_IMAGE=$FLUENTD_IMAGE
WLS_MICRO_REQUEST_MEMORY=$HELIDON_MICRO_REQUEST_MEMORY
ES_MASTER_NODE_REQUEST_MEMORY=$ES_MASTER_NODE_REQUEST_MEMORY
ES_INGEST_NODE_REQUEST_MEMORY=$ES_INGEST_NODE_REQUEST_MEMORY
ES_DATA_NODE_REQUEST_MEMORY=$ES_DATA_NODE_REQUEST_MEMORY
GRAFANA_REQUEST_MEMORY=$GRAFANA_REQUEST_MEMORY
PROMETHEUS_REQUEST_MEMORY=$PROMETHEUS_REQUEST_MEMORY
KIBANA_REQUEST_MEMORY=$KIBANA_REQUEST_MEMORY
ES_MASTER_NODE_REPLICAS=$ES_MASTER_NODE_REPLICAS
ES_DATA_NODE_REPLICAS=$ES_DATA_NODE_REPLICAS
ES_INGEST_NODE_REPLICAS=$ES_INGEST_NODE_REPLICAS
API_SERVER_REALM=$API_SERVER_REALM
VERRAZZANO_SYSTEM_INGRESS_IP=$VERRAZZANO_SYSTEM_INGRESS_IP
ES_DATA_STORAGE=${ES_DATA_STORAGE}
PROMETHEUS_DATA_STORAGE=${PROMETHEUS_DATA_STORAGE}
GRAFANA_DATA_STORAGE=${GRAFANA_DATA_STORAGE}
EOF

# Run the out-of-cluster Verrazzano operator.
cmd="verrazzano-operator\
 --zap-log-level=debug \
 --kubeconfig=${KUBECONFIG:-${HOME}/.kube/config}\
 --verrazzanoUri=default.${VERRAZZANO_SYSTEM_INGRESS_IP}.xip.io\
 --apiServerRealm=${API_SERVER_REALM}

# --enableMonitoringStorage=${ENABLE_MONITORING_STORAGE}"
echo "Command"
echo "${cmd}"
eval ${cmd}

echo "Starting the in-cluster verrazzano-operator."
set -x
kubectl scale deployment verrazzano-operator --replicas=1 -n verrazzano-system
set +x
echo ""
