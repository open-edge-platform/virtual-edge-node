# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

variable "tinkerbell_nginx_domain" {
  description = "The domain of the Tinkerbell Nginx server"
  type        = string
}

variable "boot_image_name" {
  description = "The name of the boot image file to be generated."
  type        = string
}

variable "pxe_boot" {
  description = "Skip local image generation step if pxe-boot enabled"
  type        = bool
}
