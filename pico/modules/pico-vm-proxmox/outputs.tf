output "vm_name" {
  value = proxmox_virtual_environment_vm.node_vm.name
}

output "vm_id" {
  value = proxmox_virtual_environment_vm.node_vm.id
}

output "vm_serial" {
  value = proxmox_virtual_environment_vm.node_vm.smbios[0].serial
}

output "vm_uuid" {
  value = proxmox_virtual_environment_vm.node_vm.smbios[0].uuid
}

output "tinkerbell_nginx_domain" {
  value = var.tinkerbell_nginx_domain
}
