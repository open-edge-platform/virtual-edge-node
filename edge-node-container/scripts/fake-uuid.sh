#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Script Name: fake-uuid.sh
# Description: This script creates a folder as the output directory ($3).
# Inside the created folder, the script copies the dmi/dmi-dump reference template files ($1 and $2)
# and overwrites them using the provided uuid ($4), it also writes in this folder a new file named
# uuid containing the provided uuid ($4).
# Then, these files (dmi/uuid) can be imported as specific volume targets in a docker container,
# so that they can fake the source of information for the dmidecode tool.
# If a uuid ($4) is not specified (e.g., ""), then the script generates a new uuid using the uuidgen tool.

dmi=$1          # Stores the path to the dmi file template.
dmiDump=$2      # Stores the path to the dmi-dump file template.
outDir=$3       # Stores the path to the output of this script, the folder to output dmi/dmi-dump/uuid files.
tId=$4          # Input uuid to be used to generate new dmi/dmi-dump/uuid files.

# Checks if uuid variable is set and not empty.
if [[ -z ${tId+x} || -n "$tId" ]];
then
    echo "uuid ${tId}";
else
    tId=$(uuidgen)
    echo "uuid is unset, generated ${tId}";
fi

# Validates uuid format.
pattern='^\{?[A-Z0-9a-z]{8}-[A-Z0-9a-z]{4}-[A-Z0-9a-z]{4}-[A-Z0-9a-z]{4}-[A-Z0-9a-z]{12}\}?$'
if [[ "$tId" =~ $pattern ]]; then
    echo "valid uuid format";
else
    echo "invalid uuid format";
    exit 1;
fi

# Creates output folder to store dmi/dmi-dump/uuid files.
mkdir -p "${outDir}"

# Copy template dmi/dmi-dump files to output folder.
cp "${dmi}" "${outDir}"
cp "${dmiDump}" "${outDir}"

dmiOut=${outDir}/$(basename "${dmi}")
dmiDumpOut=${outDir}/$(basename "${dmiDump}")

# Overwrites dmi/dmi-dump files with input/provided uuid.
tmpId=$(echo "${tId}" | sed -ne 's@-@@gp'|sed 's@\([0-9a-zA-Z]\{2\}\)@\\x\1@g')
echo -ne "${tmpId}" | dd of="${dmiOut}" bs=1 seek=74 conv=notrunc
serial=$(echo "${tId}" |awk -F- '{print $1$2}')
echo -ne "${serial}" | dd of="${dmiOut}" bs=1 seek=120 conv=notrunc
dd if="${dmiOut}" of="${dmiDumpOut}" bs=1 seek=32 count=256 conv=notrunc

# Outputs the overwritten uuid and sn into a file.
uuid=$(dmidecode -s system-uuid --from-dump "${dmiDumpOut}")
echo "${uuid}"
echo "${uuid}" > "${outDir}"/uuid
sn=$(dmidecode -s system-serial-number --from-dump "${dmiDumpOut}")
echo "${sn}"
echo "${sn}" > "${outDir}"/sn