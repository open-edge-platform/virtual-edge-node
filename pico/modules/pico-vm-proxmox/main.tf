resource "random_integer" "vm_name_suffix" {
  min = 1000
  max = 100000
}

locals {
  full_vm_name    = "${var.vm_name}-${random_integer.vm_name_suffix.result}"
  boot_image_name = "${local.full_vm_name}-uefi-boot.img"
}

module "common" {
  source                  = "../common"
  boot_image_name         = local.boot_image_name
  tinkerbell_nginx_domain = var.tinkerbell_nginx_domain
}

resource "proxmox_virtual_environment_file" "upload_uefi_boot_image" {
  depends_on   = [module.common]
  content_type = "iso"
  datastore_id = var.datastore_id
  node_name    = var.proxmox_node_name

  source_file {
    path = "../common/output/${local.boot_image_name}"
  }
}

resource "proxmox_virtual_environment_vm" "node_vm" {
  depends_on = [
    random_integer.vm_name_suffix,
    proxmox_virtual_environment_file.upload_uefi_boot_image,
  ]

  node_name = var.proxmox_node_name

  name        = local.full_vm_name
  description = var.vm_description
  tags        = var.vm_tags
  agent {
    enabled = false
  }
  stop_on_destroy = true
  startup {
    up_delay   = var.vm_startup.up_delay
    down_delay = var.vm_startup.down_delay
  }

  bios = "ovmf"
  smbios {
    serial  = var.smbios_serial
    uuid    = var.smbios_uuid
    product = var.smbios_product
  }
  operating_system {
    type = var.vm_operating_type
  }

  vga {
    type = var.vga_display_type
  }

  cpu {
    cores = var.cpu_cores
    type  = var.cpu_type
  }

  memory {
    dedicated = var.memory_dedicated
    floating  = var.memory_minimum
  }

  scsi_hardware = var.scsi_hardware

  disk {
    datastore_id = var.vm_datastore_id
    file_id      = proxmox_virtual_environment_file.upload_uefi_boot_image.id
    interface    = var.disk_interface
    size         = var.disk_size
    aio          = var.disk_aio
    cache        = var.disk_cache_type
    iothread     = var.disk_iothread
    backup       = var.disk_backup
    replicate    = var.disk_replicate
  }

  efi_disk {
    datastore_id      = var.vm_datastore_id
    type              = "4m"
    pre_enrolled_keys = false
  }

  boot_order = var.boot_order

  network_device {
    bridge  = var.network_bridge
    enabled = true
    model   = var.network_model
    vlan_id = var.network_vlan_id
  }

  dynamic "tpm_state" {
    for_each = var.tpm_enable ? [1] : []

    content {
      datastore_id = var.vm_datastore_id
      version      = var.tpm_version
    }
  }

  kvm_arguments = "-chardev file,id=char0,path=/tmp/serial.${var.vm_name}.log -serial chardev:char0"
}
