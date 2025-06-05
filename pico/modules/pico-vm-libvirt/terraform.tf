# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

terraform {

  required_providers {

    libvirt = {
      source  = "dmacvicar/libvirt"
      version = "~> 0.8.3"
    }

    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.3"
    }
  }
}


