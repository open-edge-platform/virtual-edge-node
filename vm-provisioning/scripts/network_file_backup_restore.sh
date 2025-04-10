#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

# Assign arguments to variables
source "${PWD}/config"

function backup_network_file() {
  if [ -n "$BRIDGE_NAME" ]; then
     virsh net-list --all
     check_nw_int=$(sudo virsh net-list --all | awk '{print $1}' | grep -w "$BRIDGE_NAME")
      if [ -n "$check_nw_int" ]; then
         # Export the network configuration to an XML file
	 virsh net-dumpxml "$BRIDGE_NAME" > "${BRIDGE_NAME}.xml"
         sudo cp "${BRIDGE_NAME}.xml" "${BRIDGE_NAME}".xml_bkp
	 echo "Network file $BRIDGE_NAME copied to ${PWD}/${BRIDGE_NAME}.xml"
	 # This variable is declared in create_vm.sh
         # shellcheck disable=SC2034
         network_xml_file="${PWD}/${BRIDGE_NAME}.xml"
     else
         echo "Network $BRIDGE_NAME does not exist, create the network with name $BRIDGE_NAME"
         exit 1
     fi
  fi
}

function restore_network_file() {
  if [ -n "$BRIDGE_NAME" ]; then
    sudo virsh net-destroy "$BRIDGE_NAME"
    sudo virsh net-undefine "$BRIDGE_NAME"

    sudo virsh net-define "${BRIDGE_NAME}".xml_bkp
    sudo virsh net-start "${BRIDGE_NAME}"
    sudo systemctl restart libvirtd
    sudo systemctl daemon-reload
    echo "Successfully reset the $BRIDGE_NAME with backup file"
  fi
}

