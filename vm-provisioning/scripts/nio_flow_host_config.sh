#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

source "${PWD}/scripts/nio_flow_validation.sh"
source "${PWD}/scripts/common_vars.sh"

cluster_fqdn=$CLUSTER
project_name=${PROJECT_NAME:-default_project_name}
num_vms=$1
# Initialize the counter
count=0

serial_number=$2

echo "Checking for serial number in $log_file..."

# Get jwt token
JWT_TOKEN=$(get_jwt_token)
if [ -z "$JWT_TOKEN" ]; then
    echo 'FAIL="ERROR: JWT Token is required!"' >> "${log_file}"
fi

function process_serial_number()
{
  serial_number=$1
  host_data=$(curl -X POST -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" --data "{\"name\":\"${serial_number}\",\"serialNumber\":\"${serial_number}\",\"autoOnboard\": true}" --header "Content-Type: application/json" "https://api.${cluster_fqdn}/v1/projects/${project_name}/compute/hosts/register" --insecure)

  # echo "host_data: $host_data"
  host_status=$(echo "$host_data" | jq -r '.hostStatus')
  resource_id=$(echo "$host_data" | jq -r '.resourceId')

  if [ -z "$host_status" ]; then
    echo "INFO: Host is created with Resource ID: $resource_id" >> "${log_file}"
  else
    echo "FAIL=\"ERROR: $host_data\"" >> "${log_file}"
  fi
}

function validate_serial() {
    local serial="$1"
    if [[ ! "$serial" =~ ^[A-Za-z0-9]{5,20}$ ]]; then
        echo "Error: Invalid serial '$serial'. Must be 5-20 alphanumeric chars."
        return 1
    fi
    return 0
}

if [[ $serial_number ]]; then 
   # Split into an array
   IFS=',' read -ra serial_array <<< "$serial_number"

   for serial in "${serial_array[@]}"; do
      validate_serial "$serial"
   done

   # Loop through each serial number
   for serial in "${serial_array[@]}"; do
      process_serial_number "$serial"
   done
else
   # Check if the log file exists
   tail -f "$log_file" | while read -r line; do
      # Extract serial numbers from the line
      if [[ $line =~ serial=([^,]+), ]]; then
	serial_number="${BASH_REMATCH[1]}"
	count=$((count + 1))
	echo "seial number found #${count}: ${serial_number}"
	process_serial_number "$serial_number"
        if [ "$count" -eq "$num_vms" ]; then
	   echo "serial number generated for all VMs"
	   exit 0
	fi
      fi
   done
fi
