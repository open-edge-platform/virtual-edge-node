# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

############################
# VM Configuration
############################

variable "vm_name" {
  description = "Name of the virtual machine"
  type        = string
  default     = "ubuntu-server"
}

variable "cpu_cores" {
  description = "Number of CPU cores for the VM"
  type        = number
  default     = 4
}

variable "memory" {
  description = "Memory for the VM in MB"
  type        = number
  default     = 4096
}

variable "disk_size" {
  description = "Primary disk size for the VM in GB"
  type        = number
  default     = 40
}

variable "additional_disks" {
  description = "List of additional disks to attach (size in GB)"
  type = list(object({
    name = string
    size = number
  }))
  default = []
}

############################
# SMBIOS / Identity
############################

variable "smbios_uuid" {
  description = "SMBIOS UUID for the VM. If blank, it will be auto-generated."
  type        = string
  default     = ""
}

variable "smbios_serial" {
  description = "SMBIOS serial number for the VM. If blank, it will be auto-generated."
  type        = string
  default     = ""
}

variable "smbios_product" {
  description = "SMBIOS product name for the VM"
  type        = string
  default     = "Ubuntu Server VM"
}

############################
# Cloud-Init / User Config
############################

variable "default_user" {
  description = "Default username for the VM"
  type        = string
  default     = "user"
}

variable "default_password" {
  description = "Default password for the VM user"
  type        = string
  default     = "user"
  sensitive   = true
}

variable "ssh_enabled" {
  description = "Enable SSH server on the VM"
  type        = bool
  default     = true
}

variable "ssh_authorized_keys" {
  description = "List of SSH public keys to add to the default user"
  type        = list(string)
  default     = []
}

############################
# Ubuntu Image
############################

variable "ubuntu_image_url" {
  description = "URL to the Ubuntu 24.04 cloud image"
  type        = string
  default     = "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
}

############################
# Libvirt Settings
############################

variable "libvirt_uri" {
  description = "Libvirt connection URI"
  type        = string
  default     = "qemu:///system"
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
  description = "The UEFI firmware path for the VM (leave empty for BIOS boot)"
  type        = string
  default     = ""
}

variable "vm_console" {
  description = "Console type: pty (interactive) or file (log to file)"
  type        = string
  default     = "pty"
  validation {
    condition     = contains(["pty", "file"], var.vm_console)
    error_message = "The vm_console variable must be either 'pty' or 'file'."
  }
}

variable "vm_autostart" {
  description = "Automatically start the VM when the host boots"
  type        = bool
  default     = false
}
