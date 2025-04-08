#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Assign value to common variables
export network_xml_file="${PWD}/orch_network.xml"
export BOOT_PATH="/var/lib/libvirt/boot"
export BOOT_IMAGE="/var/lib/libvirt/images"
export OVMF_PATH="/usr/share/OVMF"
export log_file="out/logs/console.log"