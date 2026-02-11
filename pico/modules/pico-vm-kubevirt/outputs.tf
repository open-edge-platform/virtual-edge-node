# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

// TODO: Add serial and uuid outputs for the VM
# output "vm_serial" {
#   description = "SMBIOS serial number of the virtual machine"
#   value       = kubectl_manifest.vm.object.spec.template.spec.domain.firmware.serial
#   depends_on  = [kubectl_manifest.vm]
# }

# output "vm_uuid" {
#   description = "SMBIOS UUID of the virtual machine"
#   value       = kubectl_manifest.vm.object.spec.template.spec.domain.firmware.uuid
#   depends_on  = [kubectl_manifest.vm]
# }

output "vm_name" {
  description = "Name of the virtual machine"
  value       = local.full_vm_name
}

output "tinkerbell_haproxy_domain" {
  value = var.tinkerbell_haproxy_domain
}

output "data_volume_name" {
  description = "Name of the data volume created for the VM"
  value       = "${local.full_vm_name}-disk"
}
