# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

resource "null_resource" "generate_uefi_boot_image" {
  provisioner "local-exec" {
    environment = {
      PXE_BOOT = var.pxe_boot
    }
    command = <<EOT
set -o errexit
set -o nounset
set -o xtrace

OSTYPE=$${OSTYPE:-linux}

# Define mount directory
MOUNT_DIR="${path.module}/mnt"

# Clean up any previous mount points
if mountpoint -q "$MOUNT_DIR"; then
  sudo umount "$MOUNT_DIR"
fi
if [ -d "$MOUNT_DIR" ]; then
  rmdir "$MOUNT_DIR"
fi

# Create the output directory if it doesn't exist
mkdir -p ${path.module}/output

# Create an empty disk image file
truncate -s 4M ${path.module}/output/${var.boot_image_name}

# Create a GPT partition table and an EFI System Partition (ESP)
sudo parted --script ${path.module}/output/${var.boot_image_name} mklabel gpt
sudo parted --script ${path.module}/output/${var.boot_image_name} mkpart ESP fat32 2MiB 100%
sudo parted --script ${path.module}/output/${var.boot_image_name} set 1 esp on

# Map the partition to a loop device
loop_device=$(sudo losetup --find --show --partscan ${path.module}/output/${var.boot_image_name})

# Format the partition with a FAT filesystem
sudo mkfs.vfat $${loop_device}p1

# Create a temporary mount directory
mkdir -p ${path.module}/mnt

# Mount the partition to the temporary mount directory
case "$OSTYPE" in
  darwin*)
    hdiutil attach -mountpoint ${path.module}/mnt $${loop_device}p1 ;;
  *)
    sudo mount $${loop_device}p1 ${path.module}/mnt ;;
esac

if [ "$PXE_BOOT" != "true" ]; then

  # Create EFI directory structure
  sudo mkdir -p ${path.module}/mnt/EFI/BOOT
  
  # Fetch the iPXE binary and save it to the mounted disk image
  response_code=$(sudo -E curl \
    --write-out "%%{http_code}" \
    --verbose \
    --output ${path.module}/mnt/EFI/BOOT/BOOTX64.EFI \
    --insecure \
    --location \
    https://${var.tinkerbell_nginx_domain}/tink-stack/signed_ipxe.efi)
  
  if [ "$${response_code}" -ne 200 ]; then
    echo "Failed to download signed_ipxe.efi: Expected HTTP response code 200, got $response_code"
    exit 1
  fi
  
  # Create a startup script for UEFI to boot the iPXE binary
  echo "fs0:\EFI\BOOT\BOOTX64.EFI" | sudo tee ${path.module}/mnt/startup.nsh > /dev/null

fi
# Clean up
case "$OSTYPE" in
  darwin*)
    hdiutil detach ${path.module}/mnt ;;
  *)
    sudo umount ${path.module}/mnt ;;
esac
rmdir ${path.module}/mnt
sudo losetup -d $${loop_device}
EOT
  }
}
