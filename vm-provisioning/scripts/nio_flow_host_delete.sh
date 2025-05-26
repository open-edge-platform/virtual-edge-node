#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -eu
# Load configuration
source "${PWD}/config"
source "${PWD}/scripts/nio_configs.sh"

# Function to display usage information
usage() {
    echo "Usage: $0 <serial_number>"
    echo "Check the delete host with the specified serial number."
    echo
    echo "Arguments:"
    echo "  <serial_number>  The serial number of the host to check."
    exit 1
}

# Check if serial number is provided
EN_SERIAL_NO=$1
if [ -z "$EN_SERIAL_NO" ]; then
    echo "ERROR: Serial number argument is required."
    usage
fi

# Obtain JWT token
JWT_TOKEN=$(curl -s -k -X POST \
    "https://keycloak.${CLUSTER}/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "username=${PROJECT_API_USER}" \
    --data-urlencode "password=${PROJECT_API_PASSWORD}" \
    --data-urlencode "grant_type=password" \
    --data-urlencode "client_id=system-client" \
    --data-urlencode "scope=openid" \
    --fail-with-body | jq -r '.access_token')

if [ -z "$JWT_TOKEN" ] || [ "$JWT_TOKEN" == "null" ]; then
        echo "Error: Failed to retrieve JWT token"
        exit 1
else
    echo "JWT token retrieved successfully"
fi

# Function to delete host
function delete_host() {

    rm -f host.json

    curl --noproxy "*" --location \
        "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/hosts" \
        -H 'Accept: application/json' -H "Authorization: Bearer $JWT_TOKEN" | jq '.' > host.json || true

    index_len=$(jq '.hosts[].uuid' host.json | wc -l)
    index_len=$((index_len - 1))

    host_id=""
    status=""
    instance_id=""

    for i in $(seq 0 $index_len); do
        if jq -r "[.hosts[]][${i}].serialNumber" host.json | grep -q "$EN_SERIAL_NO" ; then

            if jq -r "[.hosts[]][${i}].instance" host.json | grep -q "instanceID"; then
                instance_id=$(jq -r "[.hosts[]][${i}].instance.instanceID" host.json)
            fi

            host_id=$(jq -r "[.hosts[]][${i}].resourceId" host.json)
            break
        fi
    done

    if [ -n "$instance_id" ]; then
        echo "Instance ID found: $instance_id. Deleting instance first."
        response=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H 'Accept: application/json'\
                -H "Authorization: Bearer ${JWT_TOKEN}" \
                "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/instances/${instance_id}")

        if [ "$response" -eq 200 ] || [ "$response" -eq 204 ]; then
            echo "Instance with ID ${instance_id} deleted successfully."
        else
            echo "Failed to delete instance with ID ${instance_id}. HTTP status: $response"
            exit 1
        fi
    fi

    if [ -n "$host_id" ]; then
        response=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H 'Accept: application/json'\
            -H "Authorization: Bearer ${JWT_TOKEN}" \
            "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/hosts/${host_id}")

        if [ "$response" -eq 200 ] || [ "$response" -eq 204 ]; then
            echo "Host with ID ${host_id} deleted successfully."
        else
            echo "Failed to delete host with ID ${host_id}. HTTP status: $response"
            exit 1
        fi
    else
        echo "ERROR: No host found with serial number $EN_SERIAL_NO."
        exit 1
    fi

    rm -f host.json
    echo "Host with serial number $EN_SERIAL_NO has been successfully deleted."
    exit 0
}

# Execute the delete host function
delete_host
