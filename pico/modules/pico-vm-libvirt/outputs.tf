# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

output "vm_name_and_serial" {
  description = "VM names and serial numbers associated with the nodes"
  value = [
    for i in range(var.vm_count) : {
      name   = libvirt_domain.node_vm[i].name
      serial = local.vm_serials[i]
      uuid   = local.vm_uuids[i]
    }
  ]
}

output "vm_names" {
  description = "List of all VM names"
  value       = [for vm in libvirt_domain.node_vm : vm.name]
}

output "vm_serials" {
  description = "List of all VM serial numbers"
  value       = local.vm_serials
}

# Map format for easy reference
output "vms_map" {
  description = "Map of VM details keyed by VM name"
  value = {
    for i in range(var.vm_count) : libvirt_domain.node_vm[i].name => {
      serial = local.vm_serials[i]
      uuid   = local.vm_uuids[i]
      index  = i + 1
    }
  }
}

# Generate CSV content for Edge Infrastructure Manager
output "edge_manager_csv" {
  description = "CSV content for Edge Infrastructure Manager bulk import"
  value = join("\n", concat(
    ["Serial,UUID,OSProfile,Site,Secure,RemoteUser,Metadata,Error - do not fill"],
    [for i in range(var.vm_count) : "${local.vm_serials[i]},${local.vm_uuids[i]},,,,,,"]
  ))
}
