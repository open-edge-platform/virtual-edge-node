<!-- SPDX-FileCopyrightText: 2026 Intel Corporation -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Generate Ubuntu VM — Libvirt Terraform Module

Terraform module to provision an **Ubuntu 24.04.4 Server** virtual machine on
a local libvirt/KVM host. The VM is bootstrapped with cloud-init to create a
default user, set a password, and enable SSH out of the box.

## Features

- **Ubuntu 24.04 LTS** cloud image (downloaded automatically)
- **Cloud-init** provisioning — user, password, SSH, hostname
- **Configurable resources** — CPU cores, memory, disk size
- **Additional disks** — attach extra data volumes as needed
- **SMBIOS identity** — control UUID, serial number, and product name (auto-generated if omitted)
- **SSH ready** — password authentication and optional authorized keys
- **UEFI support** — optional UEFI firmware for secure boot workflows
- **VNC console** — graphical access via VNC

## Prerequisites

- Libvirt with KVM support
- Terraform >= 1.9.5
- `xsltproc` — for SMBIOS XML customization
  - Linux: `apt install xsltproc`
  - macOS: `brew install libxslt`

## Quick Start

```shell
cd generate-ubuntu-vm

# Copy and edit the example variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your desired values

# Initialize and apply
terraform init
terraform apply --var-file=terraform.tfvars -auto-approve
```

## Variables

Refer to `terraform.tfvars` for the list of configurable variables and their default values.

## Outputs

Refer to `outputs.tf` for the list of output values.

## Update terraform.tfvars

Edit `terraform.tfvars` to customize the VM configuration before applying:

```hcl
# VM Configuration
vm_name   = "ubuntu-server-01"
cpu_cores = 4
memory    = 4096
disk_size = 40

smbios_serial  = "UBUNTUVM01"
smbios_product = "Ubuntu Server VM"

# User configuration
default_user     = "user"
default_password = "user"
ssh_enabled      = true

# Libvirt settings
libvirt_pool_name    = "default"
libvirt_network_name = "default"
```

## Usage Examples

### Minimal

```hcl
module "ubuntu_vm" {
  source = "./generate-ubuntu-vm"

  vm_name  = "my-ubuntu"
  memory   = 4096
  disk_size = 42949672960  # 40GB
}
```

### With custom identity and extra disks

```hcl
module "ubuntu_vm" {
  source = "./generate-ubuntu-vm"

  vm_name        = "dev-server"
  cpu_cores      = 8
  memory         = 8192
  disk_size      = 85899345920  # 80GB

  smbios_serial  = "DEVSERVER01"
  smbios_uuid    = "550e8400-e29b-41d4-a716-446655440000"

  additional_disks = [
    { name = "data", size = 53687091200 },   # 50GB
    { name = "logs", size = 21474836480 },   # 20GB
  ]

  default_user     = "user"
  default_password = "user123"
  ssh_enabled      = true

  ssh_authorized_keys = [
    "ssh-ed25519 AAAA... admin@workstation",
  ]
}
```

### UEFI boot

```hcl
module "ubuntu_vm" {
  source = "./generate-ubuntu-vm"

  vm_name          = "uefi-ubuntu"
  libvirt_firmware = "/usr/share/OVMF/OVMF_CODE_4M.fd"
}
```

## Connecting to the VM

After `terraform apply`, the SSH command is shown in the outputs:

```shell
# Get the SSH command
terraform output ssh_connection

# Or connect directly
ssh user@<vm-ip>
# Password: user123
```

### Console access

```shell
virsh console <vm_name>
# Escape with Ctrl + ]
```

## Destroy

```shell
terraform destroy
```

## File Structure

```
create-clean-vm/
├── terraform.tf                     # Provider configuration
├── variables.tf                     # Input variables
├── main.tf                          # VM, disk, cloud-init resources
├── outputs.tf                       # Output values
├── customize_domain.xsl.tftpl       # SMBIOS XML customization template
├── cloud_init_user.cfg.tftpl        # Cloud-init user-data template
├── cloud_init_network.cfg.tftpl     # Cloud-init network config template
├── terraform.tfvars                 # Variable values for deployment
├── terraform.tfvars.example         # Example variable values
└── README.md                        # This file
```
