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

provider "kubectl" {
  config_path      = var.kubeconfig_path
  load_config_file = true
}
