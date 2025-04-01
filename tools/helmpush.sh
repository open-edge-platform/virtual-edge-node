#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# helmpush.sh
# search all packages with *.tgz name and then push to remote Helm server

set -u -o pipefail

echo "# helmpush.sh, using git: $(git --version) #"

# when not running under Jenkins, use current dir as workspace
WORKSPACE=${WORKSPACE:-.}
HELM_CM_NAME=${HELM_CM_NAME:-oie}

# Filter pakage with $name-$version.tgz, and version should be $major.$minor.$patch format
pkg_list=$(find "${WORKSPACE}" -maxdepth 1 -type f -regex ".*tgz"  | grep -E ".*[0-9]+\.[0-9]+\.[0-9]+\.tgz")
if [ -z "$pkg_list" ];
then
  echo "No Packages found, exit"
  exit 0
fi

for pkg in $pkg_list
do
  echo "------$pkg------"
  helm cm-push "$pkg" "$HELM_CM_NAME"
done

echo "# helmpush.sh Success! - all charts have been pushed"
exit 0

