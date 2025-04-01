#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2018-present Open Networking Foundation
# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# helmlint.sh
# run `helm lint` on all helm charts that are found

set +e -o pipefail

# verify that we have helm installed
command -v helm >/dev/null 2>&1 || { echo "helm not found, please install it" >&2; exit 1; }

echo "# helmlint.sh, using helm version: $(helm version -c --short) #"

# Collect success/failure, and list/types of failures
fail_lint=0
failed_lint=""
failed_req=""

# when not running under Jenkins, use current dir as workspace
WORKSPACE=${WORKSPACE:-.}

# cleanup repos if `clean` option passed as parameter
if [ "$1" = "clean" ]
then
  echo "Removing any downloaded charts"
  find "${WORKSPACE}" -type d -name 'charts' -exec rm -rf {} \;
fi

# now that $1 is checked, error on undefined vars
set -u

# loop on result of 'find -name Chart.yaml'
while IFS= read -r -d '' chart
do
  chartdir=$(dirname "${chart}")

  echo "Checking chart: $chartdir"

  # update dependencies (if any)
  helm dependency update "${chartdir}"

  # lint the chart (with values.yaml if it exists)
  if [ -f "${chartdir}/values.yaml" ]; then
    helm lint --strict --values "${chartdir}/values.yaml" "${chartdir}"
  else
    helm lint --strict "${chartdir}"
  fi

  rc=$?
  if [[ $rc != 0 ]]; then
    fail_lint=1
    failed_lint+="${chartdir} "
  fi

  # check that requirements are available if they're specified
  if [ -f "${chartdir}/requirements.yaml" ]
  then
    echo "Chart has requirements.yaml, checking availability"
    helm dependency update "${chartdir}"
    rc=$?
    if [[ $rc != 0 ]]; then
      fail_lint=1
      failed_req+="${chartdir} "
    fi

    # remove charts dir after checking for availability, as this chart might be
    # required by other charts in the next loop
    rm -rf "${chartdir}/charts"
  fi

done < <(find "${WORKSPACE}" -name Chart.yaml -print0)

if [[ $fail_lint != 0 ]]; then
  echo "# helmlint.sh Failure! #"
  echo "Charts that failed to lint: $failed_lint"
  echo "Charts with failures in requirements.yaml: $failed_req"
  exit 1
fi

echo "# helmlint.sh Success! - all charts linted and have valid requirements.yaml #"

exit 0

