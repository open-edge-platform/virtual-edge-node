variable "proxmox_endpoint" {
  description = "Proxmox API endpoint"
  type        = string
}

variable "proxmox_username" {
  description = "Proxmox username"
  type        = string
  default     = "root@pam"
}

variable "proxmox_password" {
  description = "Proxmox password"
  type        = string
}

variable "proxmox_insecure" {
  description = "Allow insecure connection to Proxmox API"
  type        = bool
  default     = true
}

variable "cpu_cores" {
  description = "Number of CPU cores for the VM"
  type        = number
  default     = 16
}

variable "cpu_type" {
  description = "CPU type for the VM"
  type        = string
  default     = "host"
}

variable "memory_minimum" {
  description = "Minimum memory for the VM in MB. If 0 is set, it will be the same as dedicated memory."
  type        = number
  default     = 4096
}

variable "memory_dedicated" {
  description = "Dedicated memory for the VM in MB"
  type        = number
  default     = 16384
}

variable "disk_size" {
  description = "Disk size for the VM in GB"
  type        = number
  default     = 128
}

variable "datastore_id" {
  description = "Datastore ID for the ISO/IMG disks"
  type        = string
  default     = "local"
}

variable "network_bridge" {
  description = "Network bridge for the VM"
  type        = string
  default     = "vmbr0"
}

variable "network_model" {
  description = "Network model for the VM"
  type        = string
  default     = "virtio"
}

variable "smbios_serial" {
  description = "SMBIOS serial number for the VM. If blank, it will be auto-generated."
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

variable "proxmox_node_name" {
  description = "Name of the Proxmox node"
  type        = string
  default     = "pve"
}

variable "vm_name" {
  description = "Name of the virtual machine"
  type        = string
  default     = "pico-node"
}

variable "vm_description" {
  description = "Description of the virtual machine"
  type        = string
  default     = "DO NOT EDIT - Managed by Terraform - Source: https://github.com/intel-sandbox/personal.cthach.pico"
}

variable "vm_tags" {
  description = "Tags for the virtual machine"
  type        = list(string)
  default     = ["terraform", "pico"]
}

variable "vm_startup" {
  description = "Startup configuration for the virtual machine"
  type = object({
    up_delay   = string
    down_delay = string
  })
  default = {
    up_delay   = "60"
    down_delay = "60"
  }
}

variable "vm_datastore_id" {
  description = "Datastore ID for the VM disk"
  type        = string
  default     = "local-lvm"
}

variable "tinkerbell_nginx_domain" {
  description = "The domain of the Tinkerbell Nginx server"
  type        = string
}

variable "vm_operating_type" {
  description = "Operating type for the VM"
  type        = string
  default     = "l26"
}

variable "vga_display_type" {
  description = "VGA display type for the VM"
  type        = string
  default     = "qxl"
}

variable "boot_order" {
  description = "Boot order for the VM"
  type        = list(string)
  default     = ["scsi0"]
}

variable "scsi_hardware" {
  description = "SCSI hardware type for the VM"
  type        = string
  default     = "virtio-scsi-single"
}

variable "disk_interface" {
  description = "Disk interface type for the VM"
  type        = string
  default     = "scsi0"
}

variable "disk_aio" {
  description = "Asynchronous I/O type for the disk"
  type        = string
  default     = "io_uring"
}

variable "disk_cache_type" {
  description = "Disk cache type for the VM"
  type        = string
  default     = "none"
}

variable "disk_iothread" {
  description = "Enable I/O threading for the disk"
  type        = bool
  default     = true
}

variable "disk_backup" {
  description = "Enable backup for the disk"
  type        = bool
  default     = true
}

variable "disk_replicate" {
  description = "Enable replication for the disk"
  type        = bool
  default     = true
}

variable "tpm_enable" {
  description = "Enable TPM for the VM"
  type        = bool
  default     = true
}

variable "tpm_version" {
  description = "TPM version for the VM"
  type        = string
  default     = "v2.0"
}
