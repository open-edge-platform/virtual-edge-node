#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

source "${PWD}/config"

# Define the storage pool path
STORAGE_POOL_PATH="/var/lib/libvirt/images/${POOL_NAME}"

# Function to create network and storage pool
create_resources() {
  # Create the network XML configuration
  cat <<EOF > "${BRIDGE_NAME}.xml"
<network xmlns:dnsmasq='http://libvirt.org/schemas/network/dnsmasq/1.0' connections='1'>
  <name>${BRIDGE_NAME}</name>
  <bridge name='virbr-${BRIDGE_NAME}' stp='on' delay='0'/>
  <forward mode='nat'/>
  <ip address='192.168.100.1' netmask='255.255.255.0'>
    <dhcp>
      <range start='192.168.100.2' end='192.168.100.254'/>
    </dhcp>
  </ip>
  <dnsmasq:options>
    <dnsmasq:option value='dhcp-vendorclass=set:efi-http,HTTPClient:Arch:00016'/>
    <dnsmasq:option value='dhcp-option-force=tag:efi-http,60,HTTPClient'/>
    <dnsmasq:option value='dhcp-match=set:ipxe,175'/>
    <dnsmasq:option value='dhcp-boot=tag:efi-http,"https://tinkerbell-haproxy.${CLUSTER}/tink-stack/signed_ipxe.efi"'/>
    <dnsmasq:option value='log-queries'/>
    <dnsmasq:option value='log-dhcp'/>
    <dnsmasq:option value='log-debug'/>
  </dnsmasq:options>
</network>
EOF

  # Define and start the new network
  virsh net-define "${BRIDGE_NAME}.xml"
  virsh net-start "${BRIDGE_NAME}"
  virsh net-autostart "${BRIDGE_NAME}"

  # Clean up the network XML file
  rm -f "${BRIDGE_NAME}.xml"

  # Create the storage pool directory
  sudo mkdir -p "${STORAGE_POOL_PATH}"

  # Create the storage pool XML configuration
  cat <<EOF > "${POOL_NAME}.xml"
<pool type='dir'>
  <name>${POOL_NAME}</name>
  <target>
    <path>${STORAGE_POOL_PATH}</path>
  </target>
</pool>
EOF

  # Define, build, and start the storage pool
  virsh pool-define "${POOL_NAME}.xml"
  virsh pool-build "${POOL_NAME}"
  virsh pool-start "${POOL_NAME}"
  virsh pool-autostart "${POOL_NAME}"

  # Clean up the storage pool XML file
  rm -f "${POOL_NAME}.xml"

  # List all networks and storage pools
  virsh net-list --all
  virsh pool-list --all

  echo "Network '${BRIDGE_NAME}' and storage pool '${POOL_NAME}' created and started successfully."
}

# Function to destroy network and storage pool
destroy_resources() {
  # Destroy and undefine the network
  virsh net-destroy "${BRIDGE_NAME}" 2>/dev/null
  virsh net-undefine "${BRIDGE_NAME}" 2>/dev/null

  # Destroy and undefine the storage pool
  virsh pool-destroy "${POOL_NAME}" 2>/dev/null
  virsh pool-undefine "${POOL_NAME}" 2>/dev/null

  # Remove the storage pool directory
  sudo rm -rf "${STORAGE_POOL_PATH}"

  # List all networks and storage pools
  virsh net-list --all
  virsh pool-list --all

  echo "Network '${BRIDGE_NAME}' and storage pool '${POOL_NAME}' destroyed successfully."
}

# Main script logic
case "$1" in
  create)
    create_resources
    ;;
  destroy)
    destroy_resources
    ;;
  *)
    echo "Usage: $0 {create|destroy}"
    exit 1
    ;;
esac
