#!/usr/bin/env bash
# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

CERTS_DIR=$(cd $(dirname "$0"); pwd -P)
CERTS_OUT=$(cd $(dirname "$0"); cd ../..; pwd -P)/build/verrazzano-operator-cert

VERRAZZANO_NS=verrazzano-system

set -eu

function create_admission_controller_cert()
{
  rm -rf $CERTS_OUT
  mkdir -p $CERTS_OUT

  # Prepare ca_config.txt and verrazzano_config.txt
  sed "s/VERRAZZANO_NS/${VERRAZZANO_NS}/g" $CERTS_DIR/ca_config.txt > $CERTS_OUT/ca_config.txt
  sed "s/VERRAZZANO_NS/${VERRAZZANO_NS}/g" $CERTS_DIR/verrazzano_config.txt > $CERTS_OUT/verrazzano_config.txt

  # Create the private key for our custom CA
  openssl genrsa -out $CERTS_OUT/ca.key 2048

  # Generate a CA cert with the private key
  openssl req -new -x509 -key $CERTS_OUT/ca.key -out $CERTS_OUT/ca.crt -config $CERTS_OUT/ca_config.txt

  # Create the private key for our server
  openssl genrsa -out $CERTS_OUT/verrazzano-key.pem 2048

  # Create a CSR from the configuration file and our private key
  openssl req -new -key $CERTS_OUT/verrazzano-key.pem -subj "/CN=verrazzano-validation.${VERRAZZANO_NS}.svc" -out $CERTS_OUT/verrazzano.csr -config $CERTS_OUT/verrazzano_config.txt

  # Create the cert signing the CSR with the CA created before
  openssl x509 -req -in $CERTS_OUT/verrazzano.csr -CA $CERTS_OUT/ca.crt -CAkey $CERTS_OUT/ca.key -CAcreateserial -out $CERTS_OUT/verrazzano-crt.pem
}

create_admission_controller_cert
