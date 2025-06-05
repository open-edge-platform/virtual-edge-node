# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

output "vm_name_and_serial" {
  description = "VM name and serial number associated with the node"
  value = {
    name   = libvirt_domain.node_vm.name
    serial = local.vm_serial
    uuid   = local.vm_uuid
  }
}
