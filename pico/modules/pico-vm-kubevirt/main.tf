# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

resource "random_integer" "vm_name_suffix" {
  min = 1000
  max = 100000
}

locals {
  full_vm_name    = "${var.vm_name}-${random_integer.vm_name_suffix.result}"
  boot_image_name = "${local.full_vm_name}-uefi-boot.img"
}

module "common" {
  source                    = "../common"
  boot_image_name           = local.boot_image_name
  tinkerbell_haproxy_domain = var.tinkerbell_haproxy_domain
}

resource "null_resource" "upload_uefi_boot_image" {
  depends_on = [
    module.common
  ]

  provisioner "local-exec" {
    command = <<EOT
virtctl image-upload \
  dv \
  ${var.vm_name}-${random_integer.vm_name_suffix.result}-disk \
  --access-mode=ReadWriteOnce \
  --force-bind \
  --image-path=../common/output/${local.boot_image_name} \
  --size=${var.disk_size} \
  --uploadproxy-url=${var.upload_proxy_url} \
  --insecure \
  --wait-secs=60 \
EOT
  }
}

resource "kubectl_manifest" "vm" {
  depends_on = [
    null_resource.upload_uefi_boot_image
  ]

  yaml_body = <<-YAML
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: ${local.full_vm_name}
  namespace: ${var.vm_namespace}
spec:
  runStrategy: Always
  template:
    metadata:
      labels:
        kubevirt.io/domain: ${local.full_vm_name}
    spec:
      domain:
        cpu:
          model: ${var.cpu_type}
        devices:
          disks:
          - name: rootdisk
            cache: ${var.disk_cache_type}
          interfaces:
          - name: default
            model: ${var.network_model}
            masquerade: {}
        firmware:
          serial: ${var.smbios_serial}
          uuid: ${var.smbios_uuid}
          bootloader:
            efi:
              secureBoot: false
        machine:
          type: q35
        devices:
          autoattachGraphicsDevice: false
          tpm:
            enabled: ${var.tpm_enable}
        resources:
          requests:
            memory: ${var.memory_minimum}
            cpu: ${var.cpu_cores}
          limits:
            memory: ${var.memory_limit}
            cpu: ${var.cpu_cores}
        clock:
          utc: {}
      volumes:
      - name: rootdisk
        dataVolume:
          name: ${var.vm_name}-${random_integer.vm_name_suffix.result}-disk
  YAML
}

// This is needed because destroying the VM does not automatically delete the DataVolume.
resource "null_resource" "delete_data_volume" {
  # Store the necessary values as triggers to access them during destroy
  triggers = {
    vm_name        = var.vm_name
    vm_name_suffix = random_integer.vm_name_suffix.result
    vm_namespace   = var.vm_namespace
  }

  # This will only run when the resource is destroyed
  provisioner "local-exec" {
    when    = destroy
    command = <<EOT
kubectl delete --force dv ${self.triggers.vm_name}-${self.triggers.vm_name_suffix}-disk -n ${self.triggers.vm_namespace} --ignore-not-found=true
EOT
  }
}
