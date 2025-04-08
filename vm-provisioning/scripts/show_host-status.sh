#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Check if config file is provided
CONFIG_FILE="./config"
if [ $# -eq 1 ]; then
  CONFIG_FILE="$1"
fi

# Check if the config file exists
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Error: Config file '$CONFIG_FILE' not found."
  echo "Usage: $0 [config_file]"
  exit 1
fi


# Source the config file
# shellcheck disable=SC1090
source "$CONFIG_FILE"
source "${PWD}/scripts/nio_configs.sh"

# optional, mandatory if ca certificate is copied to certs folder and this script running from another host
orch_fqdn_ip='' #10.139.220.XXX

####   ---------Dont touch below-----------------------------------------------------------------------------------------
keycloak_curl_opts=''

# copy ca
if [ -n "$orch_fqdn_ip" ]; then
  if [ -f "certs/ca.crt" ]; then
    echo "File ca.crt exists in the folder."
    keycloak_curl_opts=(
      --noproxy '*'
      --resolve "keycloak.${CLUSTER}:443:${orch_fqdn_ip}"
      --cacert ./certs/ca.crt
    )
  else
    echo "Add ca.crt to certs folder, and orch_fqdn_ip "
    exit
  fi
fi

fetch_jwt_token() {
    # Make the API request and extract the token
    JWT_TOKEN=$(curl -s "${keycloak_curl_opts[@]}" -L -X POST "https://keycloak.${CLUSTER}/realms/master/protocol/openid-connect/token" \
        --header "Content-Type: application/x-www-form-urlencoded" \
        --data-urlencode "grant_type=password" \
        --data-urlencode "client_id=system-client" \
        --data-urlencode "username=${PROJECT_API_USER}" \
        --data-urlencode "password=${PROJECT_API_PASSWORD}" \
        --data-urlencode "scope=openid profile email groups" | jq -r '.access_token')
    export JWT_TOKEN
    
    # Verify token was retrieved
    if [ -z "$JWT_TOKEN" ] || [ "$JWT_TOKEN" == "null" ]; then
        echo "Error: Failed to retrieve JWT token"
        exit 1
    else
        echo "JWT token retrieved successfully"
    fi
}

function show_host_status() {
    echo "Fetching host status..."
    host_data_all=$(curl -X GET -H 'Accept: application/json' -H "Authorization: Bearer ${JWT_TOKEN}" -H "Content-Type: application/json" "https://api.${CLUSTER}/v1/projects/${PROJECT_NAME}/compute/hosts")
    #echo $host_data_all | jq

    # Check for valid JSON response
    if ! echo "$host_data_all" | jq . >/dev/null 2>&1; then
        echo "Error: Invalid JSON response."
        echo "Response preview: $(echo "$host_data_all" | head -c 100)"
        return 1
    fi

    host_data=$(echo "$host_data_all" | jq '.hosts[] | {
        host: .resourceId,
        instanc: .instance.instanceID,
        osname: .instance.os.name,
        devserial: .serialNumber,
        provstat: .instance.provisioningStatus,
        provstage: .instance.provisioningStatusIndicator
    }')
    
    clear
    echo "Host status for project: $PROJECT_NAME"
    echo "======================================="
    
    tput sc
    
    # Print the header
    bold="\e[1m"
    # Reset text formatting escape code
    reset="\e[0m"

    # Print the table with bold headers
    echo -e "${bold}|------|------------------|-----------|---------------|---------------|----------------------------|------------------------------------------------|${reset} "
    echo -e "${bold}| SNo. | Devserial        | OS        | Host          | Instance      | Prov_Stat                  | ProvStage                                      |${reset} "
    echo -e "${bold}|------|------------------|-----------|---------------|---------------|----------------------------|------------------------------------------------|${reset} "

    # Read the JSON data from the variable and convert it to a table
    serial_no=1
    if [ -z "$host_data" ]; then
        echo "No hosts found or data format is unexpected"
        return 0
    fi
    
    echo "$host_data" | jq -c '.' | while read -r line; do
        # echo $line
        host=$(echo "$line" | jq -r '.host')
        devserial=$(echo "$line" | jq -r '.devserial')
        osname=$(echo "$line" | jq -r '.osname')
        instanc=$(echo "$line" | jq -r '.instanc')
        provstat=$(echo "$line" | jq -r '.provstat')
        #  provstage=$(echo "$line" | jq -r '.provstage')

        # Split provstat into two parts
        a=$(echo "$provstat" | cut -d ':' -f 1)
        b=$(echo "$provstat" | cut -d ':' -f 2-)

        # Trim whitespace from a and b
        a=$(echo "$a" | xargs)
        b=$(echo "$b" | xargs)

        # Determine the color based on the value of a
        if [ "$a" = "Provisioned" ]; then
            color="\033[1;32m" # Green
        elif [ "$a" = "Provisioning Failed" ]; then
            color="\033[5;33m" # Yellow/Orange blinking
        else
            color="\033[5;33m" # Yellow/Orange blinking
        fi

        # Reset color
        reset="\e[0m"

        regexpattern="VH[0-9]{3}N[0-9]{3}.*"
        if [[ "$devserial" =~ $regexpattern ]]; then
            devserial="-- $devserial"
        #  echo "--Pattern found in the string."
        fi
        
        regexpattern="Edge Microvisor.*"
        if [[ "$osname" =~ $regexpattern ]]; then
            osname="Microvisor"
        else
            osname="Ubuntu"
        fi

        # Print the row with colored a
        printf "| %-4s | %-16s | %-9s | %-13s | %-13s | ${color}%-26s${reset} | %-46s |\n" "$serial_no" "$devserial" "$osname" "$host" "$instanc" "$a" "$b"

        # Increment the serial number
        ((serial_no++))
    done

    echo "|------|------------------|-----------|---------------|---------------|----------------------------|------------------------------------------------|"

    tput rc
}

# Main execution
echo "==== Host Status Utility ===="
echo "Configuration loaded from: $CONFIG_FILE"

# Fetch the JWT token first
fetch_jwt_token

# Then show host status
show_host_status

