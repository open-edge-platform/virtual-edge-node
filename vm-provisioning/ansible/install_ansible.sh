#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0


# This script will install Ansible on an Ubuntu server.

# Ensure the locale is set to UTF-8
export LANG=C.UTF-8
export LC_ALL=C.UTF-8

# Update the system and install required packages
echo "Updating the system and installing required packages..."
sudo apt-get update
sudo apt-get install -y software-properties-common

# Add Ansible's official PPA (Personal Package Archive)
echo "Adding Ansible's official PPA..."
sudo apt-add-repository --yes --update ppa:ansible/ansible

# Install Ansible
echo "Installing Ansible..."
sudo apt-get install -y ansible

# Verify the installation
ansible --version

# Check if the locale is set to UTF-8
locale

echo "Ansible has been installed successfully and is ready to use."
