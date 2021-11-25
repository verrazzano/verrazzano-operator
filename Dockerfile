# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y oracle-golang-release-el7 \
    && yum-config-manager --add-repo http://yum.oracle.com/repo/OracleLinux/OL7/developer/golang113/x86_64 \
    && yum install -y golang-1.13.3-1.el7 \
    && yum clean all \
    && go version

# Compile to /usr/bin
ENV GOBIN=/usr/bin

# Set go path
ENV GOPATH=/go

ARG BUILDVERSION
ARG BUILDDATE

# Need to use specific WORKDIR to match verrazzano-operator's source packages
WORKDIR /root/go/src/github.com/verrazzano/verrazzano-operator
COPY . .

ENV CGO_ENABLED 0
RUN go version
RUN go env

RUN GO111MODULE=on go build \
    -mod= \
    -ldflags '-extldflags "-static"' \
    -ldflags "-X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}" \
    -o /usr/bin/verrazzano-operator ./cmd/...

FROM ghcr.io/oracle/oraclelinux:7-slim

RUN yum update -y \
    && yum clean all \
    && rm -rf /var/cache/yum

COPY --from=build_base /usr/bin/verrazzano-operator /usr/local/bin/verrazzano-operator

# Copy source tree to image
RUN mkdir -p go/src/github.com/verrazzano/verrazzano-operator
COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano-operator go/src/github.com/verrazzano/verrazzano-operator

COPY --from=build_base /root/go/src/github.com/verrazzano/verrazzano-operator/THIRD_PARTY_LICENSES.txt /licenses/

RUN groupadd -r verrazzano-operator && useradd --no-log-init -r -g verrazzano-operator -u 1000 verrazzano-operator
RUN chown 1000:verrazzano-operator /usr/local/bin/verrazzano-operator && chmod 500 /usr/local/bin/verrazzano-operator
USER 1000

ENTRYPOINT ["/usr/local/bin/verrazzano-operator"]
