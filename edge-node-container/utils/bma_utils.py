# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

import yaml
import sys
import argparse
import os
import subprocess

def parse(string):
    all = yaml.safe_load(string)
    packages = all['packages']
    v = [{'package': d['name'], 'version': d['version']} for d in packages]
    print(v)
    return v

def download(bmas):
    full_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "bma_packages")
    print(bmas)
    for deb in bmas:
        # oras pull debs from OCI repositories
        filename = "{package}:{version}".format(**deb)
        command = f'oras pull "registry-rs.edgeorchestration.intel.com/edge-orch/en/deb/{filename}" -o {full_path}'
        subprocess.run(command, shell=True, check=True)
    subprocess.run(command, shell=True, check=True)

def bma_values(bmas):
    full_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "chart", "bma_values.yaml")
    values = {}
    for deb in bmas:
        key = f"{deb['package'].replace('-', '_')}_version"
        values[key] = deb['version']
    values["caddy_version"] = "2.7.6"
    with open(full_path, 'w') as ymlfile:
        dumpdata = yaml.dump(values)
        ymlfile.write(dumpdata)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    args = parser.parse_args()

    bmas = parse(sys.stdin.read())
    download(bmas)
    bma_values(bmas)
