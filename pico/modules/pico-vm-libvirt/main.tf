# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

locals {
  boot_image_name = "${var.vm_name}-uefi-boot.img"
  # Create lists for multiple VMs
  vm_uuids   = [for i in range(var.vm_count) : length(var.smbios_uuid) > 0 ? var.smbios_uuid : random_uuid.vm_uuid[i].result]
  # Generate random serials using random_string instead of random_id
  vm_serials = [for i in range(var.vm_count) : length(var.smbios_serial) > 0 ? var.smbios_serial : random_string.vm_serial[i].result]
  vm_names   = [for i in range(var.vm_count) : "${var.vm_name}-${format("%02d", i + 1)}"]
}

resource "random_uuid" "vm_uuid" {
  count = var.vm_count
}

# Generate CSV file for Edge Infrastructure Manager
variable "edge_os_profile" {
  description = "OS Profile for Edge Infrastructure Manager"
  type        = string
  default     = ""
}

variable "edge_site" {
  description = "Site for Edge Infrastructure Manager"
  type        = string
  default     = ""
}

variable "edge_secure" {
  description = "Secure setting for Edge Infrastructure Manager"
  type        = string
  default     = ""
}

# Then update the CSV generation:
resource "local_file" "edge_manager_csv" {
  depends_on = [libvirt_domain.node_vm]
  
  filename = "${path.module}/edge_nodes_import.csv"
  content = join("\n", concat(
    ["Serial,UUID,OSProfile,Site,Secure,RemoteUser,Metadata,Error - do not fill"],
    [for i in range(var.vm_count) : "${local.vm_serials[i]},${local.vm_uuids[i]},${var.edge_os_profile},${var.edge_site},${var.edge_secure},,"]
  ))
}

# Output the CSV file path
output "csv_file_path" {
  description = "Path to the generated CSV file for Edge Infrastructure Manager"
  value       = local_file.edge_manager_csv.filename
}

# Use random_string instead of random_id to match your bash script behavior
resource "random_string" "vm_serial" {
  count   = var.vm_count
  length  = 5
  special = false
  upper   = false  # Only lowercase
  lower   = true
  numeric = true
}

module "common" {
  count = contains(var.boot_order, "network") && length(var.boot_order) == 1 ? 0 : 1
  source                  = "../common"
  boot_image_name         = local.boot_image_name
  tinkerbell_nginx_domain = var.tinkerbell_nginx_domain != "" ? var.tinkerbell_nginx_domain : "dummy.local"
  boot_order              = var.boot_order
}

# Ensure default storage pool exists before provisioning
resource "null_resource" "ensure_libvirt_pool" {
  count = contains(var.boot_order, "network") && length(var.boot_order) == 1 ? 0 : 1

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
  count = (contains(var.boot_order, "network") && length(var.boot_order) == 1) ? 0 : var.vm_count
  depends_on = [null_resource.ensure_libvirt_pool, module.common]
  name       = "${local.vm_names[count.index]}-vol"
  pool       = var.libvirt_pool_name
  source     = "../common/output/${local.boot_image_name}"
  format     = "raw"
}

resource "libvirt_domain" "node_vm" {
  count   = var.vm_count
  name    = local.vm_names[count.index]
  memory  = var.memory
  vcpu    = var.cpu_cores
  running = false
  type    = "qemu"  # Use QEMU emulation instead of KVM for nested virtualization compatibility

  firmware = var.libvirt_firmware

  cpu {
    mode = "host-model" # Use host-model to match the host CPU as closely as possible
  }

  dynamic "disk" {
    for_each = (contains(var.boot_order, "network")) && length(var.boot_order) == 1 ? [] : [1]
    content {
      volume_id = libvirt_volume.uefi_boot_image[count.index].id
    }
  }

  network_interface {
    network_name = var.libvirt_network_name
  }

  graphics {
    type = "vnc"
  }

  boot_device {
    dev = var.boot_order
  }

  tpm {
    model = var.tpm_enable ? "tpm-tis" : ""
  }

  xml {
    xslt = templatefile("${path.module}/customize_domain.xsl.tftpl", {
      smbios_product = var.smbios_product,
      smbios_serial  = local.vm_serials[count.index],
      vm_name        = local.vm_names[count.index],
      vm_uuid        = local.vm_uuids[count.index],
      vm_console     = var.vm_console
    })
  }
}

resource "null_resource" "update_libvirtvm_and_restart" {
  count = var.vm_count

  provisioner "local-exec" {
    command = <<-EOT
      ${(contains(var.boot_order, "network")) && length(var.boot_order) == 1 ? "" : "# Resize the boot disk\n      virsh vol-resize ${libvirt_volume.uefi_boot_image[count.index].name} --pool ${libvirt_volume.uefi_boot_image[count.index].pool} ${var.disk_size}G"}

      # Start the VM
      virsh start "${libvirt_domain.node_vm[count.index].name}"
    EOT
  }
}
