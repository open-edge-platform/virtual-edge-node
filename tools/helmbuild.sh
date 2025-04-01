#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# helmbuild.sh
# build helm charts based on change folders

set -eu -o pipefail

echo "# helmbuild.sh, using git: $(git --version) #"

# when not running under Jenkins, use current dir as workspace
WORKSPACE=${WORKSPACE:-.}

# Label to add Helm CI meta-data
LABEL_REVISION=$(git rev-parse HEAD)
LABEL_CREATED=$(date -u "+%Y-%m-%dT%H:%M:%SZ")

# Get the changed file name from the latest commit and then get the root folder name.
# shellcheck disable=SC1001
changed_dirs=$(git show --pretty="" --name-only | xargs dirname \$\1 | cut -d "/" -f1 | sort | uniq)

# Print lists of files that are changed/untracked
if [ -z "$changed_dirs" ]
then
  echo "# chart_version_check.sh - No changes, Success! #"
  exit 0
fi

for dir in ${changed_dirs};
do
  if [ ! -f "$dir/Chart.yaml" ]; then
    continue
  fi
  echo "---------$dir-------------"
  echo "--download helm dependency"
  helm dep build "$dir"
  echo "--add annotations"
  yq eval -i ".annotations.revision = \"${LABEL_REVISION}\"" "$dir"/Chart.yaml
  yq eval -i ".annotations.created = \"${LABEL_CREATED}\"" "$dir"/Chart.yaml
  echo "--package helm"
  helm package "$dir"  
done


echo "# helmbuild.sh Success! - all charts have updated packaged#"
exit 0

