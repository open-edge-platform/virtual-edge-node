# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

export ORCH_C_URL=cluster-orch-node.${_ORCH_FQDN_}:443
export ORCH_I_URL=infra-node.${_ORCH_FQDN_}:443
export ORCH_RA_URL=infra-node.${_ORCH_FQDN_}:443
export ORCH_N_L_OBS=logs-node.${_ORCH_FQDN_}
export ORCH_N_L_OBS_PORT=443
export ORCH_N_M_OBS=metrics-node.${_ORCH_FQDN_}
export ORCH_N_M_OBS_PORT=443
export ORCH_I_MM_URL=update-node.${_ORCH_FQDN_}:443
export ORCH_I_TM_URL=telemetry-node.${_ORCH_FQDN_}:443
export ORCH_TOKEN_URL=keycloak.${_ORCH_FQDN_}
export RS_TOKEN_URL=release.${_ORCH_FQDN_}
export RS_TYPE=no-auth
export APT_SOURCE_URL=files-rs.edgeorchestration.intel.com
export APT_SOURCE_REPO_ROOT=files-edge-orch
export APT_SOURCE_PROXY_PORT=60444
# Agents version
export NODE_AGENT_VERSION=${_NODE_AGENT_VERSION_}
export REMOTE_ACCESS_AGENT_VERSION=${_REMOTE_ACCESS_AGENT_VERSION_}
export CLUSTER_AGENT_VERSION=${_CLUSTER_AGENT_VERSION_}
export HDA_AGENT_VERSION=${_HDA_AGENT_VERSION_}
export POA_AGENT_VERSION=${_POA_AGENT_VERSION_}
export PLATFORM_UPDATE_AGENT_VERSION=${_PLATFORM_UPDATE_AGENT_VERSION_}
export PLATFORM_TELEMETRY_AGENT_VERSION=${_PLATFORM_TELEMETRY_AGENT_VERSION_}
export CADDY_VERSION=${_CADDY_VERSION_}
