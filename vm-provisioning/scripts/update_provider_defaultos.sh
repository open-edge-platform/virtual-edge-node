#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Validate input arguments
if [ $# -ne 1 ]; then
    echo "Error: Exactly one argument required" >&2
    exit 1
fi

# Load configuration variables
os_type="$1"
source "${PWD}/config"
source "${PWD}/scripts/nio_configs.sh"

# Define OS profile mappings
declare -A OS_PROFILES=(
    ["microvisor"]="microvisor-nonrt"
    ["microvisor-standalone"]="microvisor-standalone"
    ["ubuntu"]="ubuntu-22.04-lts-generic-ext"
)

# Validate OS type
if [[ ! "${OS_PROFILES[$os_type]:-}" ]]; then
    echo "Error: Invalid OS type '$os_type'" >&2
    echo "Valid options: ${!OS_PROFILES[*]}" >&2
    exit 1
fi

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

# Function to make API calls with error handling
make_api_call() {
    local method="$1"
    local url="$2"
    local data="${3:-}"
    
    local curl_cmd="curl -s -H 'Accept: application/json' -H 'Authorization: Bearer ${JWT_TOKEN}'"
    
    if [ "$method" = "POST" ]; then
        curl_cmd="$curl_cmd -X POST -H 'Content-Type: application/json' --data '$data'"
    elif [ "$method" = "DELETE" ]; then
        curl_cmd="$curl_cmd -X DELETE"
    fi
    
    eval "$curl_cmd '$url'"
}

# Function to update the defaultOs provider
function update_defaultOs_provider() {
    local profile_name="${OS_PROFILES[$1]}"
    echo "Processing OS profile: $profile_name"
    
    # Get OS Profile and Provider Lists
    make_api_call "GET" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/os" | jq > os_profile.json
    make_api_call "GET" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers" | jq > provider.json
    
    # Get OS Resource ID
    local osResourceID1
    osResourceID1=$(jq -r --arg profile "$profile_name" '.OperatingSystemResources[] | select(.profileName == $profile) | .osResourceID' os_profile.json)
    
    if [ -z "$osResourceID1" ] || [ "$osResourceID1" = "null" ]; then
        echo "Error: $profile_name OS Resource ID not found. Available profiles:" >&2
        jq -r '.OperatingSystemResources[].profileName' os_profile.json >&2
        exit 1
    fi
    
    echo "$profile_name profile osResourceID=$osResourceID1"

    # Check if provider already exists
    local existing_provider_id
    existing_provider_id=$(jq -r --arg osResourceID1 "$osResourceID1" '.providers[] | select(.config | fromjson? | .defaultOs == $osResourceID1) | .providerID' provider.json)

    if [ -n "$existing_provider_id" ] && [ "$existing_provider_id" != "null" ]; then
        echo "Provider with defaultOs set to $osResourceID1 already exists with Provider ID: $existing_provider_id. Skipping creation."
        return 0
    fi

    # Remove old provider if exists
    local old_provider_id
    old_provider_id=$(jq -r '.providers[] | select(.config | fromjson? | has("defaultOs")) | .providerID' provider.json)
    
    if [ -n "$old_provider_id" ] && [ "$old_provider_id" != "null" ]; then
        echo "Deleting old Provider ID: $old_provider_id"
        make_api_call "DELETE" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers/${old_provider_id}"
    fi

    # Create new provider
    echo "Creating new provider with defaultOs=$osResourceID1"
    local provider_data='{
        "providerKind": "PROVIDER_KIND_BAREMETAL",
        "name": "infra_onboarding",
        "apiEndpoint": "xyz123",
        "apiCredentials": ["abc123"],
        "config": "{\"defaultOs\":\"'"${osResourceID1}"'\",\"autoProvision\":true}"
    }'
    
    make_api_call "POST" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers" "$provider_data"
    echo "Provider created successfully"

    # Display updated lists
    echo -e "\n=== Updated OS Profiles ==="
    make_api_call "GET" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/os" | jq
    
    echo -e "\n=== Updated Providers ==="
    make_api_call "GET" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/providers" | jq
}

# Usage function
usage() {
    echo "Usage: $0 [microvisor|microvisor-standalone|ubuntu]" >&2
    echo ""
    echo "Available options:"
    for os_type in "${!OS_PROFILES[@]}"; do
        printf "  %-20s -> Create %s DefaultOS Provider\n" "$os_type" "${OS_PROFILES[$os_type]}"
    done
    echo ""
    exit 0
}

# Main execution
case $os_type in
    "" | "-h" | "--help")
        usage
        ;;
    *)
        if [[ "${OS_PROFILES[$os_type]:-}" ]]; then
            update_defaultOs_provider "$os_type"
        else
            echo "Error: Invalid OS type '$os_type'" >&2
            usage
        fi
        ;;
esac
