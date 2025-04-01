#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0


sudo apt remove --purge libvirt-daemon-system libvirt-clients vagrant virt-manager ovmf expect minicom socat xterm -y
sudo apt remove --purge qemu qemu-kvm libvirt-dev qemu-kvm -y
sudo apt-get purge docker-ce docker-ce-cli containerd.io -y
sudo rm -rf /var/lib/docker
sudo rm -rf /var/lib/containerd
sudo rm /etc/apt/sources.list.d/docker.list
sudo apt-get autoremove -y

sudo unlink /etc/apparmor.d/disable/usr.lib.libvirt.virt-aa-helper
sudo unlink /etc/apparmor.d/disable/usr.sbin.libvirtd