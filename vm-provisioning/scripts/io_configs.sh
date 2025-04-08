#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# IO Flow Configurations
# Check if ONBOARDING_USERNAME is set, otherwise prompt the user
if [ -z "${ONBOARDING_USERNAME}" ]; then
    read -rp "Enter onboarding username: " ONBOARDING_USERNAME
fi

# Check if ONBOARDING_PASSWORD is set, otherwise prompt the user
if [ -z "${ONBOARDING_PASSWORD}" ]; then
    read -rsp "Enter onboarding password: " ONBOARDING_PASSWORD
    echo
fi

    # Export the variables for use in the script
export USERNAME_HOOK="${ONBOARDING_USERNAME}"
export PASSWORD_HOOK="${ONBOARDING_PASSWORD}"
