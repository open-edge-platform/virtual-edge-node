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
terraform apply
```

## Variables

| Name | Description | Type | Default |
|------|-------------|------|---------|
| `vm_name` | Name of the virtual machine | `string` | `ubuntu-server` |
| `cpu_cores` | Number of CPU cores | `number` | `4` |
| `memory` | Memory in MB | `number` | `4096` |
| `disk_size` | Primary OS disk size in bytes | `number` | `42949672960` (40 GB) |
| `additional_disks` | Extra data disks (`name`, `size`) | `list(object)` | `[]` |
| `smbios_uuid` | SMBIOS UUID (auto-generated if blank) | `string` | `""` |
| `smbios_serial` | SMBIOS serial number (auto-generated if blank) | `string` | `""` |
| `smbios_product` | SMBIOS product name | `string` | `Ubuntu Server VM` |
| `default_user` | Default login username | `string` | `user` |
| `default_password` | Default login password | `string` | `user123` |
| `ssh_enabled` | Install and enable SSH server | `bool` | `true` |
| `ssh_authorized_keys` | SSH public keys for the default user | `list(string)` | `[]` |
| `ubuntu_image_url` | URL to the Ubuntu 24.04 cloud image | `string` | Ubuntu official |
| `libvirt_uri` | Libvirt connection URI | `string` | `qemu:///system` |
| `libvirt_pool_name` | Libvirt storage pool | `string` | `default` |
| `libvirt_network_name` | Libvirt network | `string` | `default` |
| `libvirt_firmware` | UEFI firmware path (blank = BIOS) | `string` | `""` |
| `vm_console` | Console type: `pty` or `file` | `string` | `pty` |
| `vm_autostart` | Auto-start VM on host boot | `bool` | `false` |

## Outputs

| Name | Description |
|------|-------------|
| `vm_name` | Name of the created VM |
| `vm_uuid` | SMBIOS UUID |
| `vm_serial` | SMBIOS serial number |
| `vm_ip` | IP address(es) of the VM |
| `vm_memory_mb` | Allocated memory in MB |
| `vm_cpu_cores` | Allocated CPU cores |
| `vm_disk_size_bytes` | Primary disk size in bytes |
| `ssh_connection` | Ready-to-use SSH command |

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
generate-ubuntu-vm/
├── terraform.tf                     # Provider configuration
├── variables.tf                     # Input variables
├── main.tf                          # VM, disk, cloud-init resources
├── outputs.tf                       # Output values
├── customize_domain.xsl.tftpl       # SMBIOS XML customization template
├── cloud_init_user.cfg.tftpl        # Cloud-init user-data template
├── cloud_init_network.cfg.tftpl     # Cloud-init network config template
├── terraform.tfvars.example         # Example variable values
└── README.md                        # This file
```
