#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Define the directory where the script is located
SCRIPT_PATH="$(dirname "$0")"

# Ensure the script is running from the correct directory
cd "$SCRIPT_PATH" || { echo "Failed to change directory to $SCRIPT_PATH"; exit 1; }

# Set permissions for specific directories and files
echo "Setting permissions..."

# Recursively set 600 permissions for files with specific extensions
find . -type f \
    \( -name "*.txt" -o -name "*.md" -o -name "*.jpg" -o -name "*.png" \
    -o -name "*.logs" \) -exec chmod 600 {} \;

# Recursively set 700 permissions for files with specific extensions
find . -type f \
    \( -name "*.sh" -o -name "*.yml" -o -name "*.py" -o -name "*.xml" \
    -o -name "*.crt" -o -name "*.mk" -o -name "*.ps*" -o -name "*.exp" \) \
    -exec chmod 700 {} \;

# Set 700 permissions for files without extensions
find . -type f ! -name "*.*" -exec chmod 700 {} \;

# Set 640 permissions for all directories
find . -type d -exec chmod 700 {} \;

echo "Permissions set successfully."

# End of script