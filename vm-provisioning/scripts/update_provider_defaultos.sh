#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Load configuration variables
os_type="$1"
source "${PWD}/config"
source "${PWD}/scripts/io_configs.sh"
source "${PWD}/scripts/nio_configs.sh"

# Function to obtain JWT token
get_jwt_token() {
    curl -s -k -X POST \
        "https://keycloak.${CLUSTER}/realms/master/protocol/openid-connect/token" \
        --data-urlencode "username=${PROJECT_API_USER}" \
        --data-urlencode "password=${PROJECT_API_PASSWORD}" \
        --data-urlencode "grant_type=password" \
        --data-urlencode "client_id=system-client" \
        --data-urlencode "scope=openid" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        --fail-with-body | jq -r '.access_token'
}

# Retry logic
max_retries=3
retry_count=0
JWT_TOKEN=""

while [ $retry_count -lt $max_retries ]; do
    JWT_TOKEN=$(get_jwt_token)
    if [ -n "$JWT_TOKEN" ]; then
        break
    fi
    echo "Failed to obtain JWT token, retrying... ($((retry_count + 1))/$max_retries)"
    retry_count=$((retry_count + 1))
    sleep 5  # Optional: add a delay between retries
done

if [ -z "$JWT_TOKEN" ]; then
    echo "Failed to obtain JWT token after $max_retries attempts"
    exit 1
fi

echo "Successfully obtained JWT token"

# Function to update the defaultOs provider
function update_defaultOs_provider() {
    # Get OS Profile List
    curl -s -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" \
      "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/os" | jq > os_profile.json
    # Get Provider List
    curl -s -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" \
    "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers" | jq > provider.json
    local osResourceID1
    if [ "$1" = "microvisor" ]; then
        osResourceID1=$(jq -r ".OperatingSystemResources[] | select(.profileName == \"microvisor-nonrt\") | .osResourceID" os_profile.json)
        echo "microvisor-nonrt profile osResourceID=$osResourceID1"
    elif [ "$1" = "ubuntu" ]; then
        osResourceID1=$(jq -r ".OperatingSystemResources[] | select(.profileName == \"ubuntu-22.04-lts-generic-ext\") | .osResourceID" os_profile.json)
        echo "Ubuntu Profile ubuntu-22.04-lts-generic-ext osResourceID=$osResourceID1"
    else
        echo "Wrong argument selected"
        return
    fi

    if [ -z "$osResourceID1" ]; then
      echo "$1 OS Resource ID not detected. Exiting script."
      cat os_profile.json  
      exit 1
    fi

    # Check if a provider with the defaultOs set to osResourceID1 already exists
    existing_provider_id=$(jq -r --arg osResourceID1 "$osResourceID1" '.providers[] | select(.config | fromjson? | .defaultOs == $osResourceID1) | .providerID' provider.json)

    if [ -n "$existing_provider_id" ]; then
        echo "Provider with defaultOs set to $osResourceID1 already exists with Provider ID: $existing_provider_id. Skipping creation."
    else
        # Get Providers
        provider_id=$(jq -r '.providers[] | select(.config | fromjson? | has("defaultOs")) | .providerID' provider.json)
        if [ -n "$provider_id" ]; then
            echo "Deleting old Provider ID: $provider_id"
            echo "The providerID for the provider with defaultOs is \"$provider_id\"."
            curl -s -X DELETE -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" \
            "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers/${provider_id}"
        else
            echo "No provider found with defaultOs."
        fi
        echo "Creating new provider with defaultOs=$osResourceID1"
        # Update Default Provider
        curl -s -X POST "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers" \
        -H "Accept: application/json" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        --data '{
            "providerKind": "PROVIDER_KIND_BAREMETAL",
            "name": "infra_onboarding",
            "apiEndpoint": "xyz123",
            "apiCredentials": ["abc123"],
            "config": "{\"defaultOs\":\"'"${osResourceID1}"'\",\"autoProvision\":true}"
        }'
    fi

    # Refresh OS Profile List
    curl -s -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" \
    "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/os" | jq

    # Refresh Provider List
    curl -s -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" \
    "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers" | jq
}

# Execute functions based on os_type
# print out a guide
usage() {
    echo "Usage: /bin/bash $0 [microvisor|ubuntu]" 1>&2;
    echo "
        ./scripts/update_provider_defaultos.sh microvisor  –> Create Microvisor DefaultOS Provider with SB Disable
        ./scripts/update_provider_defaultos.sh ubuntu  –> Create Ubuntu DefaultOS Provider with SB Disable
    "
    exit 0;
}

case $os_type in
    "" | "-h" | "--help")
      usage
      ;;
    "microvisor")
       update_defaultOs_provider "microvisor"
      ;;
    "ubuntu")
      update_defaultOs_provider "ubuntu"
      ;;
    *)
      usage
      ;;
esac
