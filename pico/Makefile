# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

all: lint

lint:
	tflint --init --recursive
	tflint --recursive

dependency-check:
	@# Help: Runs dependency-check stage
	@echo "---MAKEFILE BUILD---"
	echo $@
	@echo "---END MAKEFILE Build---"

build:
	@# Help: Runs build stage
	@echo "---MAKEFILE BUILD---"
	echo $@
	@echo "---END MAKEFILE Build---"

docker-build:
	@# Help: Runs docker-build stage
	@echo "---MAKEFILE BUILD---"
	echo $@
	@echo "---END MAKEFILE Build---"
libvirt-installer:
	@echo "Installing Libvirt..."
	bash ../vm-provisioning/install_packages.sh
