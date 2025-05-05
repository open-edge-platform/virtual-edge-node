resource "random_integer" "vm_name_suffix" {
  min = 1000
  max = 100000
}

locals {
  boot_image_name = "${var.vm_name}-${random_integer.vm_name_suffix.result}-uefi-boot.img"
}

resource "null_resource" "generate_uefi_boot_image" {
  provisioner "local-exec" {
    command = <<EOT
set -o errexit
set -o nounset
set -o xtrace

OSTYPE=$${OSTYPE:-linux}

# Clean up any previous mount points
sudo umount ${path.module}/mnt 2>/dev/null || true
rmdir ${path.module}/mnt 2>/dev/null || true

# Clean up any previous output files
rm -rf ${path.module}/output 2>/dev/null || true

# Create the output directory
mkdir -p ${path.module}/output

# Create an empty disk image file
truncate -s 2M ${path.module}/output/${local.boot_image_name}

# Format the disk image with a FAT filesystem
mkfs.vfat ${path.module}/output/${local.boot_image_name}

# Create a temporary mount directory
mkdir -p ${path.module}/mnt

# Mount the disk image to the temporary mount directory
case "$OSTYPE" in
  darwin*)
    hdiutil attach -mountpoint ${path.module}/mnt ${path.module}/output/${local.boot_image_name} ;;
  *)
    sudo mount -o loop ${path.module}/output/${local.boot_image_name} ${path.module}/mnt ;;
esac

# Fetch the iPXE binary and save it to the mounted disk image
response_code=$(sudo -E curl \
  --write-out "%%{http_code}" \
  --verbose \
  --output ${path.module}/mnt/signed_ipxe.efi \
  --insecure \
  --location \
  https://${var.tinkerbell_nginx_domain}/tink-stack/signed_ipxe.efi)

if [ "$${response_code}" -ne 200 ]; then
  echo "Failed to download signed_ipxe.efi: Expected HTTP response code 200, got $response_code"
  exit 1
fi

# Create a startup script for UEFI to boot the iPXE binary
echo "fs0:\signed_ipxe.efi" | sudo tee ${path.module}/mnt/startup.nsh > /dev/null

# Clean up
case "$OSTYPE" in
  darwin*)
    hdiutil detach ${path.module}/mnt ;;
  *)
    sudo umount ${path.module}/mnt ;;
esac
rmdir ${path.module}/mnt
EOT
  }
}

resource "proxmox_virtual_environment_file" "upload_uefi_boot_image" {
  depends_on = [
    random_integer.vm_name_suffix,
    null_resource.generate_uefi_boot_image,
  ]

  content_type = "iso"
  datastore_id = var.datastore_id
  node_name    = var.proxmox_node_name

  source_file {
    path = "${path.module}/output/${local.boot_image_name}"
  }
}

resource "proxmox_virtual_environment_vm" "node_vm" {
  depends_on = [
    random_integer.vm_name_suffix,
    proxmox_virtual_environment_file.upload_uefi_boot_image,
  ]

  node_name = var.proxmox_node_name

  name        = "${var.vm_name}-${random_integer.vm_name_suffix.result}"
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
  }

  dynamic "tpm_state" {
    for_each = var.tpm_enable ? [1] : []

    content {
      datastore_id = var.vm_datastore_id
      version      = var.tpm_version
    }
  }
}
