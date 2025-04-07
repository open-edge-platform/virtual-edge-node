#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#set -x
source ./config
source "${PWD}/scripts/nio_configs.sh"

cluster_fqdn=$CLUSTER
project_name=${PROJECT_NAME:-default_project_name}
api_user=${PROJECT_API_USER:-api_user}
api_password=${PROJECT_API_PASSWORD:- default_api_password}

function get_jwt_token() {
  JWT_TOKEN=$(curl --location --insecure --request POST "https://keycloak.${cluster_fqdn}/realms/master/protocol/openid-connect/token" \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode 'grant_type=password' \
  --data-urlencode 'client_id=system-client' \
  --data-urlencode "username=${api_user}" \
  --data-urlencode "password=${api_password}" \
  --data-urlencode 'scope=openid profile email groups' | jq -r '.access_token')

  if [ -z "$JWT_TOKEN" ] || [ "$JWT_TOKEN" == "null" ]; then
    echo "ERROR: Failed to obtain JWT Token"
    exit 1
  fi

  echo "$JWT_TOKEN"
}

function does_project_exist() {
  JWT_TOKEN=$(get_jwt_token)
  echo "JWT Token: ${JWT_TOKEN}"
  proj_name=$(curl -X GET -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" --header "Content-Type: application/json" "https://api.${cluster_fqdn}/v1/projects/${project_name}" | jq -r .spec.description)

  echo "Project name:$project_name"
  if [ -z "$proj_name" ] || [ "$proj_name" == "null" ]; then
    echo "ERROR: Provided Project name does not exist"
    exit 1
  else
    echo "Project name $proj_name exist"
  fi
}

#does_project_exist
#set +x
