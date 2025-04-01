#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -eu
# Load configuration
source "${PWD}/config"
# Function to display usage information
usage() {
    echo "Usage: $0 <serial_number>"
    echo "Check the status of hosts with the specified serial number."
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

# Function to check host status
function host_status() {
    while true; do
            curl --noproxy "*" --location \
                "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/hosts" \
                -H 'Accept: application/json' -H "Authorization: Bearer $JWT_TOKEN" | jq '.' > host.json || true
        index_len=$(jq '.hosts[].uuid' host.json | wc -l)
        index_len=$((index_len - 1))
        rm -rf host-list;touch host-list
        for i in $(seq 0 $index_len); do
            if jq -r "[.hosts[]][${i}].serialNumber" host.json | grep -q "$EN_SERIAL_NO"; then
                host_id=$(jq -r "[.hosts[]][${i}].resourceId" host.json)
                instance_id=$(jq -r "[.hosts[]][${i}].instance.instanceID" host.json)
                get_guid=$(jq -r "[.hosts[]][${i}].uuid" host.json)
                sn_no=$(jq -r "[.hosts[]][${i}].serialNumber" host.json)
                host_status=$(jq -r "[.hosts[]][${i}].hostStatus" host.json)
                os_name=$(jq -r "[.hosts[]][${i}].instance.desiredOs.name" host.json)
                image_url=$(jq -r "[.hosts[]][${i}].instance.desiredOs.imageUrl" host.json)
                echo "$host_id,$os_name,$image_url,$instance_id,$sn_no,$get_guid,$host_status" >> host-list
            fi
        done

        host_running=$(grep -c "Running" host-list || true)
        total_host=$(wc -l < host-list)

        if [ "$host_running" -eq 0 ]; then
            echo "ERROR: No hosts with 'Running' status found for serial no $EN_SERIAL_NO."
            cat host-list
        else
            echo "Total No of onboarded hosts starting with serial no $EN_SERIAL_NO = $total_host"
            grep "Running" host-list || true
            grep -v "Running" host-list || true
             {
                 echo "VEN_OS_NAME=$os_name"
                 echo "VEN_IMAGE_URL=$image_url"
                 echo "VEN_EN_SERIAL_NO=$sn_no"
                 echo "VEN_EN_UUID=$get_guid"
                 echo "VEN_EN_STATUS=$host_status"
             } > VEN_EN_INFO
             cat VEN_EN_INFO
            exit 0
        fi
       sleep 10
    done
}

# Execute the host status function
host_status
