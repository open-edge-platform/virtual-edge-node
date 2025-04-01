#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Update package list
# Add Docker's official GPG key:


# Check dmesg for specific virtualization-related strings
if command -v systemd-detect-virt &>/dev/null; then
  env_type=$(systemd-detect-virt)
  if [ "$env_type" == "none" ]; then
    echo "Bare Metal continuing install"
  else
    echo "Running in a VM: $env_type"
  fi
else
  echo "systemd-detect-virt not found. Install or try another method."
fi

sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings

if ! command -v docker &>/dev/null; then
  echo "Docker could not be found, attempting to install Docker..."

  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  sudo chmod a+r /etc/apt/keyrings/docker.asc

  # Add the repository to Apt sources:
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" |
    sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  sudo apt autoremove -y
  sudo groupadd docker
  sudo usermod -aG docker "${USER}"


  # Define your proxy settings
  HTTP_PROXY="${http_proxy:-}"
  HTTPS_PROXY="${https_proxy:-}"
  NO_PROXY="${noproxy:-}"
  FTP_PROXY="${ftp_proxy:-}"
  
  # Create or modify the Docker systemd service file to include the proxy settings
  sudo mkdir -p /etc/systemd/system/docker.service.d

  # Create a file named http-proxy.conf
  # Create a file named http-proxy.conf
  cat <<EOF | sudo tee /etc/systemd/system/docker.service.d/http-proxy.conf >/dev/null
[Service]
Environment="HTTP_PROXY=$HTTP_PROXY"
Environment="HTTPS_PROXY=$HTTPS_PROXY"
Environment="FTP_PROXY=$FTP_PROXY"
Environment="NO_PROXY=$NO_PROXY"
EOF

  # Reload the systemd daemon to apply the changes
  sudo systemctl daemon-reload

  # Restart the Docker service to use the proxy settings
  sudo systemctl restart docker
  sleep 2
#    newgrp docker
fi

#docker pull hello-world

#TODO detect ubuntu 22.04 or 24.04
# based on that install softwares

# Install specific packages
sudo apt-get install -y qemu qemu-kvm libvirt-dev

# Install additional tools
sudo apt-get install -y libvirt-daemon-system libvirt-clients pesign virt-manager ovmf expect minicom socat xterm efitools

sudo systemctl start libvirtd
sudo systemctl enable libvirtd
sleep 3
sudo usermod -aG libvirt "$USER"
sudo usermod -aG kvm "$USER"

# Backup the original configuration file
sudo cp /etc/libvirt/libvirtd.conf /etc/libvirt/libvirtd.conf.bak

# Update the configuration file
sudo sed -i 's/^#unix_sock_group = "libvirt"/unix_sock_group = "libvirt"/' /etc/libvirt/libvirtd.conf
sudo sed -i 's/^#unix_sock_rw_perms = "0770"/unix_sock_rw_perms = "0770"/' /etc/libvirt/libvirtd.conf

# Ensure the settings are present in the file if they were not commented out
grep -q '^unix_sock_group = "libvirt"' /etc/libvirt/libvirtd.conf || echo 'unix_sock_group = "libvirt"' | sudo tee -a /etc/libvirt/libvirtd.conf
grep -q '^unix_sock_rw_perms = "0770"' /etc/libvirt/libvirtd.conf || echo 'unix_sock_rw_perms = "0770"' | sudo tee -a /etc/libvirt/libvirtd.conf

sudo systemctl restart libvirtd
# Disable apparmor profiles for libvirt
sudo ln -sf /etc/apparmor.d/usr.sbin.libvirtd /etc/apparmor.d/disable/
sudo ln -sf /etc/apparmor.d/usr.lib.libvirt.virt-aa-helper /etc/apparmor.d/disable/
sudo apparmor_parser -R /etc/apparmor.d/usr.sbin.libvirtd
sudo apparmor_parser -R /etc/apparmor.d/usr.lib.libvirt.virt-aa-helper

sudo systemctl restart libvirtd
sleep 2
sudo systemctl reload apparmor
sleep 2
# Verify installations and display versions
echo "Installed applications and their versions:"
dpkg -l | grep -E 'qemu|libvirt-daemon-system|ebtables|libguestfs-tools|libxslt-dev|libxml2-dev'

# Check KVM support
echo "Checking KVM support..."
if kvm-ok; then
    echo "KVM acceleration is supported on this system."
else
    echo "KVM acceleration is not supported or not enabled. Please check your BIOS/UEFI settings."
fi