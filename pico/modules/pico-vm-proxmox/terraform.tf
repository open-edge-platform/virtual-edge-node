terraform {
  required_version = ">= 1.9.5"

  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = "~> 0.73.2"
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

provider "proxmox" {
  endpoint = var.proxmox_endpoint
  username = var.proxmox_username
  password = var.proxmox_password
  insecure = var.proxmox_insecure

  random_vm_id_start = var.proxmox_random_vm_id_start
  random_vm_id_end = var.proxmox_random_vm_id_end
  random_vm_ids = var.proxmox_random_vm_ids

  ssh {
    agent = true
    dynamic "node" {
      for_each = var.proxmox_endpoint_ssh != "" ? [1] : []
      content {
      name    = var.proxmox_node_name
      address = var.proxmox_endpoint_ssh
      }
    }
  }
}
