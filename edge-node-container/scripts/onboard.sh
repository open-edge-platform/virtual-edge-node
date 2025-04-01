#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Script Name: onboard.sh
# Description: This script is meant to run as systemd service
# and is used to onboard/provision enic.

set -xeo pipefail

function onboard-provision() {
  BINARY_PATH="/opt/enic/bin/enic"
  echo "EdgeNode setup using golang scripts/binary ${BINARY_PATH}"
  $BINARY_PATH -globalLogLevel="debug" -orchFQDN="${_ORCH_FQDN_}" -orchCAPath="/usr/local/share/ca-certificates/ca.crt" -baseFolder="/etc/intel_edge_node" -onbUser="${_ORCH_USER_}" -onbPass="${_ORCH_PASS_}" -projectID="${_ORCH_PROJECT_}" -oamServerAddress="${_OAM_SERVER_ADDRESS_}" -enableNIO="${_ENABLE_NIO_}"
}

echo "Onboard/Provision"
onboard-provision

touch /var/edge_node/edge_node_onboarded
