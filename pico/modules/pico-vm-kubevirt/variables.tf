# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

variable "kubeconfig_path" {
  type        = string
  description = "Path to the kubeconfig file."
  default     = "~/.kube/config"
}

variable "upload_proxy_url" {
  description = "The URL of the upload proxy for virtctl image-upload."
  type        = string
  default     = "https://cdi-uploadproxy:31001"
}

variable "cpu_cores" {
  description = "Number of CPU cores for the VM"
  type        = number
  default     = 16
}

variable "cpu_type" {
  description = "CPU type for the VM"
  type        = string
  default     = "host-model"
}

variable "memory_minimum" {
  description = "Minimum memory for the VM to request from the scheduler."
  type        = string
  default     = "8Gi"
}

variable "memory_limit" {
  description = "Memory limit for the VM. This is the maximum memory that the VM can use."
  type        = string
  default     = "16Gi"
}

variable "disk_size" {
  description = "Disk size for the VM. The minimum size for EMT is 110GB."
  type        = string
  default     = "110Gi"
}

variable "network_model" {
  description = "Network model for the VM"
  type        = string
  default     = "virtio"
}

variable "smbios_serial" {
  description = "SMBIOS serial number for the VM."
  type        = string
}

variable "smbios_uuid" {
  description = "SMBIOS UUID for the VM. If blank, it will be auto-generated."
  type        = string
  default     = ""
}

variable "vm_name" {
  description = "Name of the virtual machine"
  type        = string
  default     = "pico-node"
}

variable "vm_namespace" {
  description = "Namespace for the virtual machine"
  type        = string
  default     = "default"
}

variable "disk_cache_type" {
  description = "Disk cache type for the VM"
  type        = string
  default     = "none"
}

variable "tpm_enable" {
  description = "Enable TPM for the VM"
  type        = bool
  default     = true
}

variable "tinkerbell_haproxy_domain" {
  description = "The domain of the Tinkerbell HAProxy server"
  type        = string
}
