# SPDX-FileCopyrightText: 2026 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

locals {
  vm_uuid   = length(var.smbios_uuid) > 0 ? var.smbios_uuid : random_uuid.vm_uuid.result
  vm_serial = length(var.smbios_serial) > 0 ? var.smbios_serial : upper(random_id.vm_serial.hex)
}

resource "random_uuid" "vm_uuid" {}

resource "random_id" "vm_serial" {
  byte_length = 5
}

############################
# Ensure storage pool exists
############################

resource "null_resource" "ensure_libvirt_pool" {
  provisioner "local-exec" {
    command = <<-EOT
      set -e
      if ! virsh pool-info ${var.libvirt_pool_name} > /dev/null 2>&1; then
        echo "Creating libvirt pool: ${var.libvirt_pool_name}"
        virsh pool-define-as ${var.libvirt_pool_name} dir --target /var/lib/libvirt/images
        virsh pool-build ${var.libvirt_pool_name}
        virsh pool-start ${var.libvirt_pool_name}
        virsh pool-autostart ${var.libvirt_pool_name}
      else
        echo "Libvirt pool ${var.libvirt_pool_name} already exists."
      fi
    EOT
  }
}

############################
# Ubuntu base image volume
############################

resource "libvirt_volume" "ubuntu_base" {
  depends_on = [null_resource.ensure_libvirt_pool]
  name       = "${var.vm_name}-base.qcow2"
  pool       = var.libvirt_pool_name
  source     = var.ubuntu_image_url
  format     = "qcow2"
}

############################
# Primary OS disk (resized)
############################

resource "libvirt_volume" "os_disk" {
  depends_on     = [libvirt_volume.ubuntu_base]
  name           = "${var.vm_name}-os.qcow2"
  pool           = var.libvirt_pool_name
  base_volume_id = libvirt_volume.ubuntu_base.id
  size           = var.disk_size * 1073741824  # Convert GB to bytes
  format         = "qcow2"
}

############################
# Additional data disks
############################

resource "libvirt_volume" "additional_disk" {
  for_each = { for idx, d in var.additional_disks : d.name => d }

  name   = "${var.vm_name}-${each.key}.qcow2"
  pool   = var.libvirt_pool_name
  size   = each.value.size * 1073741824  # Convert GB to bytes
  format = "qcow2"
}

############################
# Cloud-Init
############################

resource "libvirt_cloudinit_disk" "cloud_init" {
  name = "${var.vm_name}-cloudinit.iso"
  pool = var.libvirt_pool_name

  user_data = templatefile("${path.module}/cloud_init_user.cfg.tftpl", {
    username            = var.default_user
    password            = var.default_password
    ssh_enabled         = var.ssh_enabled
    ssh_authorized_keys = var.ssh_authorized_keys
    hostname            = var.vm_name
  })

  network_config = templatefile("${path.module}/cloud_init_network.cfg.tftpl", {})
}

############################
# Domain (VM)
############################

resource "libvirt_domain" "ubuntu_vm" {
  name      = var.vm_name
  memory    = var.memory
  vcpu      = var.cpu_cores
  autostart = var.vm_autostart

  firmware = length(var.libvirt_firmware) > 0 ? var.libvirt_firmware : null

  cpu {
    mode = "host-model"
  }

  cloudinit = libvirt_cloudinit_disk.cloud_init.id

  # Primary OS disk
  disk {
    volume_id = libvirt_volume.os_disk.id
  }

  # Additional disks
  dynamic "disk" {
    for_each = libvirt_volume.additional_disk
    content {
      volume_id = disk.value.id
    }
  }

  network_interface {
    network_name   = var.libvirt_network_name
    wait_for_lease = true
  }

  graphics {
    type        = "vnc"
    listen_type = "address"
  }

  xml {
    xslt = templatefile("${path.module}/customize_domain.xsl.tftpl", {
      smbios_product = var.smbios_product
      smbios_serial  = local.vm_serial
      vm_name        = var.vm_name
      vm_uuid        = local.vm_uuid
      vm_console     = var.vm_console
    })
  }
}
