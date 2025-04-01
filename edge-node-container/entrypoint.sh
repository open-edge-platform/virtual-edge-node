#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Script Name: entrypoint.sh
# Description: This script is the entrypoint of ENiC

set -xeo pipefail

# Setup the default environment variables for systemd
mkdir -p /etc/systemd/system.conf.d/
tee /etc/systemd/system.conf.d/myenvironment.conf << END
[Manager]
DefaultEnvironment=$(while read -r Line; do echo -n "$Line " ; done < <(env))
END

# Start systemd
if [ "$DEPLOY_TYPE" = "ENIVM" ]; then
  uid=$(id -u)
  export XDG_RUNTIME_DIR=/run/user/$uid
  exec /lib/systemd/systemd --user >/dev/null &
else
  exec /lib/systemd/systemd
fi
