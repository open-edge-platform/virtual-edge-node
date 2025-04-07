#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# NIO Flow Configurations
# Check if PROJECT_API_USER is set, otherwise prompt the user
if [ -z "${PROJECT_API_USER}" ]; then
    read -p "Enter Project API Username: " PROJECT_API_USER
fi

# Check if PROJECT_API_PASSWORD is set, otherwise prompt the user
if [ -z "${PROJECT_API_PASSWORD}" ]; then
    read -sp "Enter Project API Password: " PROJECT_API_PASSWORD
    echo
fi

    # Export the variables for use in the script
export PROJECT_API_USER="${PROJECT_API_USER}"
export PROJECT_API_PASSWORD="${PROJECT_API_PASSWORD}"
