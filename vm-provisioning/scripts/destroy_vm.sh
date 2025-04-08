#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

source "${PWD}/config"
# Assign arguments to variables
source "${PWD}/scripts/common_vars.sh"
source "${PWD}/scripts/network_file_backup_restore.sh"

# List all VMs and filter those starting with "vm-provisioning"
VM_PREFIX="vm-provisioning"
VM_LIST=$(virsh list --all --name | grep "^${VM_PREFIX}")

# Check if any VMs were found
if [ -z "$VM_LIST" ]; then
    echo "No VMs found with prefix '${VM_PREFIX}'."
    exit 0
fi
pkill -9 minicom || true
# Iterate over each VM and delete it
for vm_name in $VM_LIST; do
    nw_name=$(virsh domiflist "$vm_name" | sed -n '3p' | awk '{print $3}')
    nw_names+=("$nw_name")
    echo "Processing VM: $vm_name"
    # Destroy the VM if it is running
    if virsh list --name | grep -q "^${vm_name}$"; then
        echo "Destroying VM: $vm_name"
        virsh destroy "$vm_name"
    fi
    # Undefine the VM, including NVRAM if applicable
    echo "Undefining VM: $vm_name"
    virsh undefine "$vm_name" --nvram
done
if [ -n "$STANDALONE" ]; then
  echo "standalone mode $STANDALONE"
  restore_network_file "${nw_names[@]}"
fi
sudo rm -rf /tmp/console*.sock
sudo find "${BOOT_IMAGE}"/ -name 'vm-provisioning*' -exec rm -rf {} +
sudo ls -l "${BOOT_IMAGE}"/
virsh list --all
virsh net-list --all
