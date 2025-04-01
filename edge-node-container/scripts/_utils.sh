#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# contains functions shared across files

configureEnvironment() {
  echo "Configure Environment"
  set +e
  # NOTE that the service will restart indefinetely until it finishes the config
  if [ ! -f /opt/enic/bin/agents_env.sh ]; then
    set -e
    echo "Generate configuration files"
    agents_env=$(envsubst < /etc/agents_env.tpl)
    echo "${agents_env}" > /opt/enic/bin/agents_env.sh
  fi
  set -e
  # shellcheck disable=SC1091
  source /opt/enic/bin/agents_env.sh
}
