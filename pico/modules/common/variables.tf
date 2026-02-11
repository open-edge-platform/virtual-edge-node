# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

variable "tinkerbell_haproxy_domain" {
  description = "The domain of the Tinkerbell HAProxy server"
  type        = string
}

variable "boot_image_name" {
  description = "The name of the boot image file to be generated."
  type        = string
}

variable "boot_order" {
  description = "Skip local image generation step if boot order network."
  type        = list(string)
}
