#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

# source variables from common variable file
source "${PWD}/scripts/common_vars.sh"

# Function to get the IP address of a VM
get_vm_ip() {
  local vm_name=$1
  local ip
  ip=$(virsh domifaddr "$vm_name" --source agent --interface --full | grep -oP '(\d{1,3}\.){3}\d{1,3}')
  echo "$ip"
}

# Function to delete a specific VM
delete_vm() {
  local vm_name=$1

  # Check if the specified VM exists
  if ! virsh dominfo "$vm_name" &> /dev/null; then
    echo "VM '$vm_name' does not exist."
    return
  fi

  # Get the IP address of the VM
  ip=$(get_vm_ip "$vm_name")
  echo "Destroying VM: $vm_name (IP: $ip)"
  virsh destroy "$vm_name"

  # Check for and delete snapshots
  snapshots=$(virsh snapshot-list "$vm_name" --name)
  for snapshot in $snapshots; do
    if [ -n "$snapshot" ]; then
      echo "Deleting snapshot: $snapshot for VM: $vm_name"
      virsh snapshot-delete "$vm_name" "$snapshot"
    fi
  done

  # Remove NVRAM file if it exists
  nvram_file=$(virsh dumpxml "$vm_name" | grep -oP '(?<=<nvram>).*?(?=</nvram>)')
  if [ -n "$nvram_file" ]; then
    echo "Removing NVRAM file: $nvram_file for VM: $vm_name"
    rm -f "$nvram_file"
  fi

  # Undefine the VM and remove all associated storage
  echo "Undefining VM: $vm_name"
  virsh undefine "$vm_name" --remove-all-storage

  echo "VM '$vm_name' has been cleaned up."
}

# Check if VM names are provided as arguments
if [ "$#" -gt 0 ]; then
  # Loop through each provided VM name and delete it
  for vm_name in "$@"; do
    delete_vm "$vm_name"
  done
else
  # No VM names provided, delete all VMs
  echo "Cleaning up all VMs."
  vms=$(virsh list --all --name)

  for vm in $vms; do
    if [ -n "$vm" ]; then
      delete_vm "$vm"
    fi
  done

  echo "All VMs have been cleaned up."
fi

# Get a list of all inactive networks starting with "orchvm-net-"
networks=$(virsh net-list --all | grep 'orchvm-net-' | grep -v ' active ' | awk '{print $1}')

# Loop through the list and remove each network
for net in $networks; do
    echo "Deleting inactive network: $net"
    virsh net-destroy "$net"
    virsh net-undefine "$net"
done
echo "All inactive networks starting with 'orchvm-net-' have been removed."

sudo bash -c "rm -rf ${BOOT_PATH}/*}_ca.der"
sudo bash -c "rm -rf ${OVMF_PATH}/OVMF_*-vm*.fd"
sudo bash -c "rm -rf ${BOOT_IMAGE}/*-vm*.qcow2"
sudo bash -c "rm -rf ${BOOT_IMAGE}/*-vm*.raw"

echo "All Vhdd and certs got cleaned up."
