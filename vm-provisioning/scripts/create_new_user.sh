#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Function to create a new user
create_user() {
    local username=$1
    local password=$2

    # Create the user with a home directory and bash shell
    sudo useradd -m -s /bin/bash "$username"

    # Set the user's password
    echo "$username:$password" | sudo chpasswd

    # Add the user to the specified groups
    sudo usermod -aG sudo,kvm,docker,libvirt "$username"

    # Verify the user's group membership
    groups "$username"

    echo "User $username has been created and added to the specified groups."
}

# Function to run a script independently using nohup and disown
run_script_independently() {
    local username=$1
    local script_path=$2

    # Switch to the user's home directory
    sudo -u "$username" bash -c "cd ~ && nohup $script_path & disown"
}

# Main script
read -r -p "Enter the username to create: " username
read -r -sp "Enter the password for the new user: " password
echo

# Create the user
create_user "$username" "$password"
echo "User creation and setup complete."
