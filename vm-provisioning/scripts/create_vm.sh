#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# set -x

# source variables from common variable file
source "${PWD}/config"
source "${PWD}/scripts/common_vars.sh"

# Assign arguments to variables
export SUPPORT_GUI_XTERM="${SUPPORT_GUI_XTERM:-""}"
export USERNAME_LINUX="${USERNAME_LINUX:-user}"
export PASSWORD_LINUX="${PASSWORD_LINUX:-user}"
export SUPPORT_GUI_XTERM="${SUPPORT_GUI_XTERM:-""}"
export CI_CONFIG="${CI_CONFIG:-false}"

#########################################################################################
if command -v systemd-detect-virt &>/dev/null; then
  env_type=$(systemd-detect-virt)
  if [ "$env_type" == "none" ]; then
    echo "Bare Metal continuing install"
  else
    echo "Running in a VM: $env_type"
  fi
else
  echo "systemd-detect-virt not found. Install or try another method."
fi

BASE_DIR="${PWD}/scripts"
# Number of VMs to create
NUM_VMS=$1
FLOW=$2

if [ "$FLOW" == "-io" ] || [ -z "$FLOW" ]; then
    # Source IO Flow Configurations
    source "${PWD}/scripts/io_configs.sh"
elif [ "$FLOW" == "-nio" ]; then
    # Source NIO Flow Configurations
    source "${PWD}/scripts/nio_configs.sh"
else
    echo "Invalid flow specified. Please use '-io' or '-nio'."
    exit 1
fi

# Starting SSH port (will increment for each VM)
SSH_PORT=6000
STORAGE_FAIL_VMS=0
WORKFLOW_FAIL_VMS=0
IPXE_FAIL_VMS=0
OTH_FAIL_VMS=0

BOOT_EFI_URI="https://${CLUSTER}/tink-stack/signed_ipxe.efi"

# Extract NAME and VERSION_ID from /etc/os-release
NAME=$(grep '^NAME=' /etc/os-release | cut -d'=' -f2 | tr -d '"')
VERSION_ID=$(grep '^VERSION_ID=' /etc/os-release | cut -d'=' -f2 | tr -d '"')
  
export HTTPS_BOOT_FILE="$BOOT_EFI_URI"
#--------------------------------------

# Parse arguments
if [ "$FLOW" = "-nio" ]; then
 while [[ $# -eq 3 ]]; do
    case $3 in
        -serials=*)
            IFS=',' read -r -a serials <<< "${3#*=}" # Split the value into an array
            shift # Move to the next argument
	    if [ "${#serials[@]}" -ne "$NUM_VMS" ]; then
		echo "Error: Number of serials (${#serials[@]}) does not match the expected number of VMs ($NUM_VMS)."
		exit 1
	    fi
	    for serial in "${serials[@]}"; do
    		# Append each serial to the result string with a comma
    		if [ -z "$result_string" ]; then
                  result_string="$serial"
                else
                  result_string="$result_string,$serial"
                fi
            done
	    export STATIC_CONFIG_SERIALS="$result_string"
            ;;
        *)
            echo "Unknown option: $3"
            echo "Usage: $0 <number_of_vms> [-nio] [-serials=<serials>]"
	    exit 1
            ;;
    esac
 done
fi

# Output the parsed values
echo "Serials: $STATIC_CONFIG_SERIALS"

mgmt_intf_name=""
if [ -n "$BRIDGE_NAME" ]; then
   source "${PWD}/scripts/network_file_backup_restore.sh"
   backup_network_file 
   #network_xml_file="${PWD}/${BRIDGE_NAME}.xml"
fi

ip_to_connect=$(ip route get 1 | head -n 1 | grep -o 'src\s[.0-9a-z]\+' | awk '{print $2}')

# cleanup function here
# shellcheck disable=SC2317  # Don't warn about unreachable commands in this function
cleanup_trap() {
  echo "Cleaning up child processes..."

  pkill -P $$

  # Kill all child processes of this script
  vagrant_in_docker destroy -f

  # destroy_network if not done
  network_name=$(grep "<name>" "$network_xml_file" | sed -n 's/.*<name>\(.*\)<\/name>.*/\1/p')
  network_to_remove=$(virsh net-list | grep "${network_name}")

  if [ -n "$BRIDGE_NAME" ]; then
    restore_network_file
    
    #cleanup certificate
    sudo rm -rf "${BOOT_PATH}/${mgmt_intf_name}"_ca.der
    sudo rm -rf "${OVMF_PATH}/OVMF_*_${mgmt_intf_name}"-vm*.fd
    ####  remove_vm_hdd
    if [ -z "$VM_NAME" ]; then
	 sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${mgmt_intf_name}-vm*.qcow2"
         sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${mgmt_intf_name}-vm*.raw"
         sudo bash -c "mv /var/log/libvirt/qemu/$(basename "$PWD")_${mgmt_intf_name}-vm*.log ./out/logs/"
         sudo chmod 644 "./out/logs/$(basename "$PWD")_${mgmt_intf_name}-vm"*.log
    else
         sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${VM_NAME}*.qcow2"
         sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${VM_NAME}*.raw"

         sudo bash -c "mv /var/log/libvirt/qemu/$(basename "$PWD")_${VM_NAME}*.log ./out/logs/"
	 sudo chmod 644 "./out/logs/$(basename "$PWD")_${VM_NAME}"*.log
    fi	

  elif [ -n "$network_to_remove" ]; then
    sudo virsh net-destroy "$network_to_remove"
    sudo virsh net-undefine "$network_to_remove"

    sudo rm -rf /var/lib/libvirt/boot/"${network_name}"_ca.der
    sudo rm -rf /usr/share/OVMF/OVMF_*_"${network_name}"-vm*.fd

    if [ -z "$VM_NAME" ]; then 
    	sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${network_name}-vm*.qcow2"
    	sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${network_name}-vm*.raw"

    	sudo bash -c "mv /var/log/libvirt/qemu/$(basename "$PWD")_$network_name-vm*.log ./out/logs/"
    	sudo chmod 644 "./out/logs/$(basename "$PWD")_$network_name-vm"*.log
    else
        sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${VM_NAME}*.qcow2"
        sudo bash -c "rm -rf ${BOOT_IMAGE}/$(basename "$PWD")_${VM_NAME}*.raw"

    	sudo bash -c "mv /var/log/libvirt/qemu/$(basename "$PWD")_${VM_NAME}*.log ./out/logs/"
    	sudo chmod 644 "./out/logs/$(basename "$PWD")_${VM_NAME}"*.log
    fi
  fi
  exit 0
}

if [ "$CI_CONFIG" = "false" ]; then
  trap cleanup_trap SIGINT EXIT
fi

LOGFILE="out/logs/master_log_${ip_to_connect##*.}.log"

# Function to log messages with timestamp
log_with_timestamp() {
  echo -e "[$(date +"%Y-%m-%d %H:%M:%S")] [$ip_to_connect] $1" | tee -a "$LOGFILE"
}

function wait_for_file() {
  local file_path=$1
  local wait_time=${2:-20} # Default wait time 20 seconds
  local elapsed_time=0

  while ! stat "$file_path" >/dev/null 2>&1; do
    echo "Waiting for file '$file_path' to be present (timeout: $wait_time seconds)..."
    sleep 5
    elapsed_time=$((elapsed_time + 1))
    if [ "$elapsed_time" -ge "$wait_time" ]; then
      echo "Timeout reached: File '$file_path' not found after $wait_time seconds."
      return 1
    fi
  done
  echo "File '$file_path' is now present."
  return 0
}

function vagrant_in_docker() {
  # set -x
  local run_option=""
  if [ -n "$RUN_IN_BACKGROUND" ]; then
    run_option="-d"
  fi
  TTY_OPTIONS=""
  # Declare proxy variables
  http_proxy="${http_proxy:-}"
  https_proxy="${https_proxy:-}"
  no_proxy="${no_proxy:-}"

  # shellcheck disable=SC2086
  docker run $TTY_OPTIONS --rm $run_option \
    -e LIBVIRT_DEFAULT_URI \
    -e HTTP_PROXY="$http_proxy" -e HTTPS_PROXY="$https_proxy" -e NO_PROXY="$no_proxy" -e http_proxy="$http_proxy" -e https_proxy="$https_proxy" \
    -v /var/run/libvirt/:/var/run/libvirt/ \
    -v "/home/${USER}/.vagrant.d_edge-slim:/.vagrant.d" \
    -v "$(realpath "${PWD}"):${PWD}" \
    -v /tmp:/tmp \
    -w "${PWD}" \
    vagrantlibvirt/vagrant-libvirt:edge-slim \
    vagrant "$@"
  # set +x
}

function create_default_storage_pool() {
  # Name of the storage pool to check or create
  POOL_NAME="default"

  # Directory to use for the storage pool
  POOL_DIR="${BOOT_IMAGE}"

  # Check if the 'default' storage pool exists
  if ! virsh pool-list --all | grep -q " $POOL_NAME "; then
    echo "Storage pool '$POOL_NAME' not found. Creating it..."

    # Create the directory for the storage pool if it doesn't exist
    if [ ! -d "$POOL_DIR" ]; then
      echo "Creating directory $POOL_DIR for the storage pool..."
      sudo mkdir -p "$POOL_DIR"
      sudo chown root:libvirt "$POOL_DIR"
      sudo chmod 0750 "$POOL_DIR"
    fi
    # Define and start the storage pool
    sudo virsh pool-define-as --name "$POOL_NAME" --type dir --target "$POOL_DIR"
    sudo virsh pool-autostart "$POOL_NAME"
    sudo virsh pool-start "$POOL_NAME"

    echo "Storage pool '$POOL_NAME' created and started."
  else
    echo "Storage pool '$POOL_NAME' already exists."
  fi
}

#### create random vm-networkname , brige and other configs
function create_random_virtbr_net_name() {
  boot_efi_uri=$1
  timeout_duration=15

  # Reset the SECONDS variable to start counting from 0
  SECONDS=0

    while true; do
      # Generate a random 3-digit number between 2 and 255
      random_number=$(shuf -i 100-220 -n 1)
      virbr_interface="virbr-$random_number"
      
      # Check if the virbr interface exists
      if ! ip addr show "$virbr_interface" &>/dev/null; then

      if [ -n "$BRIDGE_NAME" ]; then
        mgmt_intf_name="$BRIDGE_NAME-$random_number"

	if [ "$STANDALONE" -eq 1 ]; then

          sed -i '/<\/ip>/a\
           <dnsmasq:options>\
           <dnsmasq:option value="dhcp-vendorclass=set:efi-http,HTTPClient:Arch:00016"/>\
           <dnsmasq:option value="dhcp-option-force=tag:efi-http,60,HTTPClient"/>\
           <dnsmasq:option value="dhcp-match=set:ipxe,175"/>\
           <dnsmasq:option value="dhcp-boot=tag:efi-http,&quot;'"${boot_efi_uri}"'&quot;"/>\
           <dnsmasq:option value="log-queries"/>\
           <dnsmasq:option value="log-dhcp"/>\
           <dnsmasq:option value="log-debug"/>\
           </dnsmasq:options>' "$network_xml_file"
      
          sed -i "s|<network.*>|<network xmlns:dnsmasq='http://libvirt.org/schemas/network/dnsmasq/1.0'>|" "$network_xml_file"	
	fi
        
	 echo "$mgmt_intf_name"
	 sed -i "s/libvirt.management_network_name = \"orchvm-net-[0-9]\{1,3\}\"/libvirt.management_network_name = \"$BRIDGE_NAME\"/" "${PWD}/Vagrantfile"
	 sed -i "s|orchvm-net-[0-9]\{1,3\}|$mgmt_intf_name|g" "${PWD}/Vagrantfile"

      else
        # Use sed to replace the network-name  pattern in the file orchvm-net$random_number
        sed -i "s|orchvm-net-[0-9]\{1,3\}|orchvm-net-$random_number|g" "${network_xml_file}"
        # Use sed to replace the bridge-name  pattern in the file virbr-$random_number
        sed -i "s|virbr-[0-9]\{1,3\}|virbr-$random_number|g" "${network_xml_file}"
        # Commented: Use sed to replace the IP range in the file  192.168.$random_number.1
        sed -i "s|192\.168\.[0-9]\{1,3\}\.|192.168.$random_number\.|g" "${network_xml_file}"
  
        # replace the default interface
        default_interface=$(ip route | grep '^default' | awk '{print $5}' | head -n 1)
        sed -Ei "s|<interface dev='[0-9a-zA-Z]+'/>|<interface dev='$default_interface'/>|g" "${network_xml_file}"
  
        # replace the efi uri
        search_pattern="https://[a-zA-Z0-9.-]+/tink-stack/signed_ipxe\.efi"
        replace_pattern="$boot_efi_uri"
        sed -Ei "s|$search_pattern|$replace_pattern|g" "${network_xml_file}"
        
	echo "orchvm-net-$random_number"
        sed -i "s|orchvm-net-[0-9]\{1,3\}|orchvm-net-$random_number|g" "${PWD}/Vagrantfile"
      fi
  
	sed -i "s|orchvm-num-vms|$NUM_VMS|g" "${PWD}/Vagrantfile"
        octet_ip=${ip_to_connect##*.}
        formatted_octet_ip=$(printf "%03d" "$octet_ip")
        sed -i "s|VH[0-9]\{3\}N[0-9]\{3\}|VH${formatted_octet_ip}N${random_number}|g" "${PWD}/Vagrantfile"
	if [ "$STATIC_CONFIG_SERIALS" != "" ]; then
	   sed -Ei "s|static_config_serials = \"\"|static_config_serials = \"$STATIC_CONFIG_SERIALS\"|g" "$PWD/Vagrantfile"
	fi
        break

      # If the interface exists, the loop will continue and generate a new number
      if ((SECONDS >= timeout_duration)); then
        echo "NONE"
        break
      else
        sleep 1
      fi
    fi
    done
}

function create_attach_network() {

  network_xml_path=$1
  network_name=$(grep "<name>" "$network_xml_path" | sed -n 's/.*<name>\(.*\)<\/name>.*/\1/p')
 
  if virsh net-list --all | grep -wq "$network_name"; then
      echo "Network '$network_name' exists."
      if [ "$STANDALONE" -eq 1 ]; then 
        sudo virsh net-destroy "$network_name"
	sudo virsh net-define "$network_xml_path"
	sudo virsh net-start "$network_name"
	sudo systemctl restart libvirtd
	sudo systemctl daemon-reload

	sudo virsh net-autostart "$network_name"
      fi
  else
      echo "Network '$network_name' does not exist. Creating the network: '$network_name'"
      sudo virsh net-define "$network_xml_path"
      sudo virsh net-start "$network_name"
  fi

  sleep 2

}

function prepare_certificate_for_network() {
  if [ -z "$BRIDGE_NAME" ]; then 
  	network_name=$(grep "<name>" "$network_xml_path" | sed -n 's/.*<name>\(.*\)<\/name>.*/\1/p')
  else
	network_name=$mgmt_intf_name
  fi

  if [ "$VERSION_ID" == "22.04" ]; then 
    efisiglist -a -c certs/Full_server.crt -o "${network_name}_ca.der"
  else 
    cert-to-efi-sig-list -g "$(uuidgen)" certs/Full_server.crt "${network_name}_ca.der"
  fi

  sudo bash -c "mv ${network_name}_ca.der ${BOOT_PATH}/"

}

function check_io_or_nio() {
  set -e
  if [ "$FLOW" == "-nio" ] && [ -z "$STATIC_CONFIG_SERIALS" ]; then
    echo "NIO FLOW" 
    log_with_timestamp "Validating JWT token and project name using nio_flow_validation.sh..."
    if [ -f "${BASE_DIR}/nio_flow_validation.sh" ]; then
        ##Todo:remove shell check disable if needed 
        # shellcheck disable=SC1091  # Don't warn about unreachable commands in this function
        . "${BASE_DIR}/nio_flow_validation.sh"
    else
        log_with_timestamp "nio_flow_validation.sh not found. Exiting..."
        exit 1
    fi
    #set -x
    does_project_exist
    #set +x
    # Run nio_flow_validation.sh for validation
    if ! does_project_exist; then
        log_with_timestamp "Validation failed. Exiting..."
        exit 1
    fi
    log_with_timestamp "NIO flow config validation successful."
  elif [ "$FLOW" == "-nio" ]; then
    echo "NIO FLOW with static serial numbers: $STATIC_CONFIG_SERIALS"
  else
    echo "IO FLOW"
  fi

}

function config_serial_number() {
    # Execute nio_flow.exp to wait for the serial number
    log_with_timestamp "Executing nio_flow.exp to fetch the serial number..."
    if [ -f "${BASE_DIR}/nio_flow_host_config.sh" ]; then
        ##Todo:remove shell check disable if needed 
        # shellcheck disable=SC1091  # Don't warn about unreachable commands in this function
        . "${BASE_DIR}/nio_flow_host_config.sh" "$1"
    else
        log_with_timestamp "nio_flow_host_config.sh not found. Exiting..."
        exit 1
    fi
    last_line=$(tail -n 1 "${log_file}")
    echo "Last line: $last_line"
    if [[ "$last_line" == *"FAIL"* ]]; then
        log_with_timestamp "Failed to register the serial number. It may already exist."
        # exit 1
    fi

    # bash ./nio_flow_host_config.sh
    log_with_timestamp "Host configuration completed successfully."
}

function get_print_vnc_id() {
  i=$1
  network_xml_path=$2
  if [ -z "$BRIDGE_NAME" ]; then 
    network_name=$(grep "<name>" "$network_xml_path" | sed -n 's/.*<name>\(.*\)<\/name>.*/\1/p')
  else
    network_name=$mgmt_intf_name
  fi

  cur_folder=$(basename "$PWD")
  if [ -n "$VM_NAME" ]; then
	vm_name="${cur_folder}_${VM_NAME}${i}"
  else
  	vm_name="${cur_folder}_${network_name}-vm${i}"
  fi

  # Extract the graphics port number
  PORT=$(virsh dumpxml "$vm_name" | grep '<graphics' | grep -oP 'port='\''\K[0-9]+') || true

  ip_to_connect=$(ip route get 1 | head -n 1 | grep -o 'src\s[.0-9a-z]\+' | awk '{print $2}')
  octet_ip=${ip_to_connect##*.}
  formatted_octet_ip=$(printf "%03d" "$octet_ip")
  padded_vm=$(printf "%02d" "$i")
  if [[ $network_name =~ orchvm-net-([0-9]+)-vm ]]; then
    net_number=${BASH_REMATCH[1]}
  fi

  # Print the ip:port number
  octet_ip=${ip_to_connect##*.}
  formatted_octet_ip=$(printf "%03d" "$octet_ip")
  padded_vm=$(printf "%02d" "$i")
  if [[ $network_name =~ orchvm-net-([0-9]+)-vm ]]; then
    net_number=${BASH_REMATCH[1]}
  fi

  # Print the ip:port number
  echo -e "\e[30;103m VH${formatted_octet_ip}N${net_number}M${padded_vm} | $vm_name | VNC $ip_to_connect:$PORT \e[0m "
}
function serial_and_switch() {
  i=$1
  local net_name=$2
  echo "os version: $VERSION_ID"
  if [ "$VERSION_ID" == "22.04" ]; then
     SER0_PORT_SOCK="/tmp/console0_${net_name}-vm$i.sock"
  else
     SER0_PORT_SOCK="/tmp/console1_${net_name}-vm$i.sock"
  fi

  #for windows window mobxterm xterm is used
  if [ "$SUPPORT_GUI_XTERM" = "" ]; then
    wait_for_file "${SER0_PORT_SOCK}" &
    wait
  fi
  get_print_vnc_id "$i" "$network_xml_path"

  vm_name="$(basename "$PWD")_${net_name}-vm${i}"
  "$BASE_DIR/socket_login.exp" "$SER0_PORT_SOCK" "${net_name}-vm${i}" "${net_name}-vm${i}"
  ret=$?
  echo "[$ret] Done provisioning"

  vm_name="$(basename "$PWD")_${net_name}-vm-${i}"
  end_time_vmN=$(date +%s)
  time_taken_to_provision=$((end_time_vmN - start_time_vms))
  sleep 2


  if [ "$ret" = 120 ]; then
    log_with_timestamp "Finished provisioning $vm_name $time_taken_to_provision Sec"
  elif [ "$ret" -eq 102 ]; then
    log_with_timestamp "Error [storage] ,Failed provisioning $vm_name $time_taken_to_provision Sec"
  elif [ "$ret" -eq 13 ]; then
    log_with_timestamp "Error [wf] ,Failed provisioning $vm_name $time_taken_to_provision Sec"
  elif [ "$ret" -ge 1 ] && [ "$ret" -le 12 ]; then
    log_with_timestamp "Error [ipxe/boot|$ret] ,Failed provisioning $vm_name $time_taken_to_provision Sec"
  else
    log_with_timestamp "Error [$ret] ,Failed provisioning $vm_name $time_taken_to_provision Sec"
  fi
  #  set +x
  return "$ret"
}

function spawn_vms() {
  i=$1
  check_io_or_nio
  if [ -z "$BRIDGE_NAME" ]; then 
    network_name=$(grep "<name>" "$network_xml_path" | sed -n 's/.*<name>\(.*\)<\/name>.*/\1/p')
  else
    network_name=$mgmt_intf_name
  fi

  if [ -z "$VM_NAME" ]; then 
    vagrant_in_docker up "${network_name}-vm${i}" | tee -a "${LOG_FILE}"
  else
    vagrant_in_docker up "${VM_NAME}${i}" | tee -a "${LOG_FILE}"
  fi
 
  echo "Vagrant is UPPP" 
  serial_and_switch "$i" "$network_name"
  #  wait
}

function main() {

  echo "First argument: $NUM_VMS"
  # Check if the argument is provided and it is a number
  if [ -z "$NUM_VMS" ] || ! [[ "$NUM_VMS" =~ ^[0-9]+$ ]]; then
    echo "Usage: $0 <number_of_vms> [-nio] [-serials=<serials>]"
    exit 1
  fi
 
  if [ -n "$POOL_NAME" ] && { [ -z "$STANDALONE" ] || [ "$STANDALONE" -eq 0 ]; }; then
      if ! virsh pool-list --all | grep -q " $POOL_NAME "; then
	echo "The storage pool with the name: $POOL_NAME does not exist"
	exit 1
      fi
  else
      create_default_storage_pool
  fi
     	
  cp "${PWD}/templates/orch_network.xml" .
  cp "${PWD}/templates/Vagrantfile" .
  
  # Create the Vagrantfile
  echo "Creating Vagrantfile for $NUM_VMS $NAME $VERSION_ID VMs with custom SSH port forwarding starting at port $SSH_PORT..."

  vm_network_name=$(create_random_virtbr_net_name "$BOOT_EFI_URI")
  mgmt_intf_name="${vm_network_name}"
  echo "VM Network name will be used : $vm_network_name"
 
  echo "Network XML file: ${network_xml_file}"

  create_attach_network "${network_xml_file}"
  
  prepare_certificate_for_network "$vm_network_name"
  
  #start the vms
  #export VAGRANT_DEBUG=info
  rm -rf out/logs
  mkdir -p out/logs
  LOG_FILE="${log_file}"
  echo "$(date): Script started." | tee -a "$LOG_FILE"
  # exec > >(tee -a "$LOG_FILE") 2>&1

  sleep 5

  start_time_vms=$(date +%s)
  log_with_timestamp "Starting $NUM_VMS vms with vm-networkname $vm_network_name"
  
  for i in $(seq 1 "$NUM_VMS"); do
    sudo cp "${OVMF_PATH}/OVMF_CODE_4M.fd" "${OVMF_PATH}/OVMF_CODE_${network_name}-vm$i.fd"
    sudo cp "${OVMF_PATH}/OVMF_VARS_4M.fd" "${OVMF_PATH}/OVMF_VARS_${network_name}-vm$i.fd"
    spawn_vms "$i" &
    pids[i]=$!
  done
  
  if [ "$FLOW" == "-nio" ] && [ -z "$STATIC_CONFIG_SERIALS" ]; then
    config_serial_number "$NUM_VMS"
  fi
 
  for i in "${!pids[@]}"; do
     wait "${pids[i]}"
     return_values[i]=$?
     echo "Return value for vm-$i: ${return_values[i]}"

     if [ "${return_values[i]}" -eq 120 ]; then
        SUCCESS_VMS=$((SUCCESS_VMS + 1))
     elif [ "${return_values[i]}" -eq 102 ]; then
        STORAGE_FAIL_VMS=$((STORAGE_FAIL_VMS + 1))
     elif [ "${return_values[i]}" -eq 13 ]; then
        WORKFLOW_FAIL_VMS=$((WORKFLOW_FAIL_VMS + 1))
     elif [ "${return_values[i]}" -ge 2 ] && [ "${return_values[i]}" -le 12 ]; then
        IPXE_FAIL_VMS=$((IPXE_FAIL_VMS + 1))
     else
        OTH_FAIL_VMS=$((OTH_FAIL_VMS + 1))
     fi
  done
  echo "VM spawn return values:" "${return_values[@]}"

  ## ## wait for all provisioning done

  end_time_vms=$(date +%s)
  log_with_timestamp "Finished processing of all vms"
  time_taken_to_provision=$((end_time_vms - start_time_vms))
  log_with_timestamp "Time taken for flashing of all vms [$NUM_VMS]:  $time_taken_to_provision Sec"

  log_with_timestamp "\n ---- VM states ----\n" \
    "SUCCESS_VM: $SUCCESS_VMS\n" \
    "STORAGE_FAIL_VMS: $STORAGE_FAIL_VMS\n" \
    "WORKFLOW_FAIL_VMS: $WORKFLOW_FAIL_VMS\n" \
    "IPXE_FAIL_VMS: $IPXE_FAIL_VMS\n" \
    "OTH_FAIL_VMS: $OTH_FAIL_VMS"

  if [ "$CI_CONFIG" = "false" ]; then
  # wait until interrrupt/BACKGRND is not defined
    while [ -z "$RUN_IN_BACKGROUND" ]; do
     sleep 1
    done
    # wait until interrrupt
    while true; do
      sleep 1
    done
  fi
}

main "$@"
