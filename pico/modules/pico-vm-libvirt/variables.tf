# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

variable "cpu_cores" {
  description = "Number of CPU cores for the VM"
  type        = number
  default     = 8
}

variable "memory" {
  description = "Dedicated memory for the VM in MB"
  type        = number
  default     = 8192
}

variable "disk_size" {
  description = "Disk size for the VM in GB"
  type        = number
  default     = 110
}

variable "smbios_serial" {
  description = "List of serial numbers for VMs"
  type        = string
  default     = ""
}

variable "smbios_uuid" {
  description = "SMBIOS UUID for the VM. If blank, it will be auto-generated."
  type        = string
  default     = ""
}

variable "smbios_product" {
  description = "SMBIOS product name for the VM"
  type        = string
  default     = "Pico Node"
}

variable "vm_name" {
  description = "Name of the virtual machine"
  type        = string
  default     = "pico-node-libvirt"
}

variable "vm_console" {
  description = "Enable Console port access or logs save to file i.e pty or file"
  type        = string
  default     = "pty"
  validation {
    condition     = contains(["pty", "file"], var.vm_console)
    error_message = "The vm_console variable must be either 'pty' or 'file'."
  }
}

variable "tinkerbell_nginx_domain" {
  description = "The domain of the Tinkerbell Nginx server"
  type        = string
}

variable "tpm_enable" {
  description = "Enable TPM for the VM"
  type        = bool
  default     = true
}

variable "libvirt_pool_name" {
  description = "The name of the libvirt storage pool"
  type        = string
  default     = "default"
}

variable "libvirt_network_name" {
  description = "The name of the libvirt network"
  type        = string
  default     = "default"
}

variable "libvirt_firmware" {
  description = "The firmware to use for the VM"
  type        = string
  default     = "/usr/share/OVMF/OVMF_CODE_4M.fd"
}

variable "boot_order" {
  type = list(string)
  default = ["hd"]
}

