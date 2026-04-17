output "vm_name" {
  description = "Name of the created virtual machine"
  value       = libvirt_domain.ubuntu_vm.name
}

output "vm_uuid" {
  description = "SMBIOS UUID of the virtual machine"
  value       = local.vm_uuid
}

output "vm_serial" {
  description = "SMBIOS serial number of the virtual machine"
  value       = local.vm_serial
}

output "vm_ip" {
  description = "IP address of the virtual machine (from first network interface)"
  value       = libvirt_domain.ubuntu_vm.network_interface[0].addresses
}

output "vm_memory_mb" {
  description = "Memory allocated to the VM in MB"
  value       = var.memory
}

output "vm_cpu_cores" {
  description = "Number of CPU cores allocated"
  value       = var.cpu_cores
}

output "vm_disk_size_gb" {
  description = "Primary OS disk size in GB"
  value       = var.disk_size
}

output "ssh_connection" {
  description = "SSH connection command (once IP is available)"
  value       = length(libvirt_domain.ubuntu_vm.network_interface[0].addresses) > 0 ? "ssh ${var.default_user}@${libvirt_domain.ubuntu_vm.network_interface[0].addresses[0]}" : "Waiting for IP..."
}
