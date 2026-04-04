# VM Configuration
vm_name   = "ubuntu-server-01"
cpu_cores = 4
memory    = 4096
disk_size = 40

# Optional: additional data disks
# additional_disks = [
#   { name = "data1", size = 20 },
#   { name = "data2", size = 10 },
# ]

smbios_serial  = "UBUNTUVM01"
smbios_product = "Ubuntu Server VM"

# User configuration
default_user     = "user"
default_password = "user"
ssh_enabled      = true

# Optional: add your SSH public keys
# ssh_authorized_keys = [
#   "ssh-rsa AAAA... your-key-comment",
# ]

# Libvirt settings
libvirt_pool_name    = "edge"
libvirt_network_name = "edge"
# libvirt_uri        = "qemu:///system"
# libvirt_firmware   = "/usr/share/OVMF/OVMF_CODE_4M.fd"  # uncomment for UEFI boot
