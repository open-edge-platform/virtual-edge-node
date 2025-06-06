# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_version = ">= 1.9.5"

  required_providers {
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "1.19.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.3"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.7.1"
    }
  }
}

locals {
  boot_image_name = "${var.vm_name}-uefi-boot.img"
  vm_uuid         = length(var.smbios_uuid) > 0 ? var.smbios_uuid : random_uuid.vm_uuid.result
  vm_serial       = length(var.smbios_serial) > 0 ? var.smbios_serial : upper(random_id.vm_serial.hex)
}

resource "random_uuid" "vm_uuid" {}

resource "random_id" "vm_serial" {
  byte_length = 5
}

module "common" {
  source                  = "../common"
  boot_image_name         = local.boot_image_name
  tinkerbell_nginx_domain = var.tinkerbell_nginx_domain
}

# Ensure default storage pool exists before provisioning
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

resource "libvirt_volume" "uefi_boot_image" {
  depends_on = [null_resource.ensure_libvirt_pool, module.common]
  name       = "${var.vm_name}-vol"
  pool       = var.libvirt_pool_name
  source     = "../common/output/${local.boot_image_name}"
  format     = "raw"
}

resource "libvirt_domain" "node_vm" {
  name    = var.vm_name
  memory  = var.memory
  vcpu    = var.cpu_cores
  running = false

  firmware = var.libvirt_firmware

  cpu {
    mode = "host-model" # Use host-model to match the host CPU as closely as possible
  }

  disk {
    volume_id = libvirt_volume.uefi_boot_image.id
  }

  network_interface {
    network_name = var.libvirt_network_name
  }

  graphics {
    type = "vnc"
  }

  boot_device {
    dev = ["hd"]
  }

  tpm {
    model = var.tpm_enable ? "tpm-tis" : ""
  }

  xml {
    xslt = templatefile("${path.module}/customize_domain.xsl.tftpl", {
      smbios_product = var.smbios_product,
      smbios_serial  = local.vm_serial,
      vm_name        = var.vm_name,
      vm_uuid        = local.vm_uuid,
      vm_console     = var.vm_console
    })
  }
}

resource "null_resource" "update_libvirtvm_and_restart" {
  provisioner "local-exec" {
    command = <<-EOT
      # Resize the boot disk
      virsh vol-resize ${libvirt_volume.uefi_boot_image.name} --pool ${libvirt_volume.uefi_boot_image.pool} ${var.disk_size}G

      # Start the VM
      virsh start "${libvirt_domain.node_vm.name}"
    EOT
  }
}
