# Pico Terraform Module 👌

This repository contains Terraform modules for provisioning virtual Edge Nodes for the Open Edge Platform.

Pico means "small" in Spanish, and this module is designed to be lightweight and efficient, making it ideal for creating
Edge Nodes with minimal resource usage very quickly.

## Features

- Lightweight: Designed for minimal resource usage 🪶
- Fast: Provisions an Edge Node E2E with Tiber OS and all agents in minutes ⚡️
- Easy to use: Simple configuration with Terraform 🧘
- Highly configurable: Customize CPU, memory, disk size, and more 🔧
- Scalable: Easily scale up or down based on your needs 📈
- Cross platform: Deploy from Linux and Mac OS 🖥️
- Multiple onboarding options: Supports both interactive and non-interactive onboarding methods 🔄
- All features of a hardware-backed Edge Node: Including dynamic OS provisioning, agents, and Kubernetes 🚀

## Screenshots

![Pico Node in Orchestrator UI](/static/node_details.png)

## Requirements

- `dosfstools` package
  - Linux: `apt install dosfstools`
  - Mac OS: `brew install dosfstools`
- `curl` package
  - Linux: `apt install curl`
  - Mac OS: `brew install curl`
- Terraform v1.9.5 or later
- Proxmox Virtual Environment (PVE) with API access
  - Use <= v8.3.0 because v8.4.0 has a bug with detecting the boot image.
- An Open Edge Platform Orchestrator with a Tinkerbell Nginx URL
- **Note:** Ensure that the `no_proxy` environment variable is set correctly for your network configuration. This
  variable should be set to include any necessary domains to bypass the proxy.

## Usage

### Interactive CLI

```shell
# Change directory to the Proxmox module
cd modules/pico-vm-proxmox

# Initialize the module
terraform init

# Apply the configuration
terraform apply
```

This will prompt you for the required variables. You can also provide them via a `terraform.tfvars` file or as environment variables.

### Terraform

To use this module, include it in your Terraform configuration as follows:

```hcl
module "pico_vm" {
    source = "./modules/pico-vm-proxmox"

    vm_name           = "example-vm"
    vm_description    = "Example VM created with Pico module"
    datastore_id      = "local-lvm"
    proxmox_node_name = "pve-node"

    cpu_cores        = 16
    memory_dedicated = 16384
    disk_size        = "128G"

    smbios_serial    = "1234-5678-9012"
    smbios_uuid      = "abcd-efgh-ijkl-mnop"
    smbios_product   = "PicoVM"

    network_bridge = "vmbr0"
    network_model  = "virtio"

    tinkerbell_nginx_domain = "your-nginx-url"
}
```

## Outputs

- `vm_name`: The name of the created virtual machine
- `vm_id`: The ID of the created virtual machine
- `vm_serial`: The SMBIOS serial number of the virtual machine
- `vm_uuid`: The SMBIOS UUID of the virtual machine
- `tinkerbell_nginx_domain`: The Tinkerbell Nginx URL for the virtual machine

## Contributing

Contributions are welcome! Please submit an issue or pull request for any improvements or bug fixes.
