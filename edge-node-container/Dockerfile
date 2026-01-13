# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.5-bookworm AS build

ENV GO111MODULE=on
ARG MAKE_TARGET=go-build
# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

COPY Makefile go.mod go.sum /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-container/
COPY version.mk common.mk /go/src/github.com/open-edge-platform/virtual-edge-node/
COPY cmd/ /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-container/cmd/
COPY vendor/ /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-container/vendor/

WORKDIR /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-container
RUN CGO_ENABLED=0 DOCKER_LABEL_REPO_URL=${REPO_URL} DOCKER_LABEL_VERSION=${VERSION} DOCKER_LABEL_REVISION=${REVISION} DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} make ${MAKE_TARGET}

# POA requires yq to be installed in the machine
FROM golang:1.25.5-bookworm AS build-yq
RUN go install github.com/mikefarah/yq/v4@v4.45.1


FROM debian:bookworm-slim@sha256:cea2634840f5a87503d8210e4df97b9f23a2acd67ff860a76c133d963032f866

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

# Fix GPG keys and install base dependencies
RUN rm -rf /var/lib/apt/lists/* && \
    apt-get update -o Acquire::Check-Valid-Until=false -o Acquire::AllowInsecureRepositories=true || true && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y -o APT::Get::AllowUnauthenticated=true \
    apparmor-utils apt-transport-https apt-utils curl debian-keyring \
    dialog dmidecode gpg incron intel-gpu-tools\
    ipmitool iptables jq libcap2-bin libpq5 libyaml-0-2 lsb-release lshw \
    lxc needrestart net-tools nginx pciutils unzip rasdaemon smartmontools \
    sudo systemd udev ufw usbutils uuid-runtime wget rsyslog clevis-initramfs \
    console-setup console-setup-linux cpio cryptsetup cryptsetup-bin \
    cryptsetup-initramfs initramfs-tools initramfs-tools-core \
    kbd keyboard-configuration klibc-utils kmod libcjson1 libev4 libklibc \
    liblocale-gettext-perl libplymouth5 libtss2-mu0 libtss2-rc0 \
    libtss2-sys1 libtss2-tcti-cmd0 libtss2-tcti-device0 libtss2-tcti-mssim0 \
    libtss2-tcti-swtpm0 libtss2-tctildr0 libuv1 linux-base \
    plymouth plymouth-theme-breeze tpm-udev tpm2-abrmd xkb-data zstd libcurl3-gnutls \
    libtss2-esys-3.0.2-0 tpm2-tools cron procps sed gawk systemd-resolved && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y && \
    # Explicitly upgrade vulnerable packages to apply security patches \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --only-upgrade libc6 perl-base && \
    # Remove packages that pull in vulnerable dependencies not needed for core functionality \
    apt-get remove -y qemu-block-extra qemu-utils && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*
    
COPY --from=build-yq /go/bin/yq /usr/bin

# Needed to track the installation progress
RUN mkdir -p /var/edge_node/ && \
  # Create other dirs needed during the deployment
  mkdir -p /etc/default/grub.d && \
  mkdir -p /etc/intel_edge_node/client-credentials && \
  mkdir -p /etc/systemd/system/containerd.service.d/ && \
  mkdir -p /etc/systemd/system/docker.service.d/ && \
  mkdir -p /etc/intel_edge_node/customer_id && \
  touch /etc/lsb-release

WORKDIR /opt

# Install the enic-agents systemd service
RUN mkdir -p enic/bin
COPY ./scripts/agents.sh ./enic/bin
COPY ./scripts/agents.service /etc/systemd/system

# Install the enic-onboard systemd service
COPY ./scripts/onboard.sh ./enic/bin
COPY ./scripts/onboard.service /etc/systemd/system

# Copy enic main binary
COPY --from=build /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-container/out/enic /opt/enic/bin/enic

# Install the cluster-dns service
RUN mkdir /etc/cluster-dns
COPY ./scripts/cluster-dns/cluster-dns.sh   /usr/bin/cluster-dns
COPY ./scripts/cluster-dns/cluster-dns.service  /etc/systemd/system
COPY ./scripts/cluster-dns/env.tpl    /etc/cluster-dns

# Copy utils.sh
COPY ./scripts/_utils.sh ./enic/bin

# Copy bma packages
RUN mkdir bma_packages
COPY ./bma_packages ./bma_packages
RUN ls ./bma_packages

# Copy tpl files
COPY ./scripts/proxy.tpl  /etc/
COPY ./scripts/agents_env.tpl  /etc/

# Copy txt files
COPY ./scripts/sudoers.txt /etc/

# Setup systemd services modifications
RUN systemctl disable ufw && \
  systemctl disable nginx && \
  systemctl enable agents && \
  systemctl enable onboard && \
  systemctl enable cluster-dns

ENV PATH="${PATH}:/opt/enic/bin"

COPY ./entrypoint.sh ./entrypoint.sh
ENTRYPOINT ["./entrypoint.sh"]
