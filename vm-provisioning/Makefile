# VM Provisioning project

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: lint license help

all: lint

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 0

##### Variables #####

# Project variables
PROJECT_NAME := virtualedgenode
BINARY_NAME  := $(PROJECT_NAME)

# Code versions, tags, and so on
VERSION       := $(shell cat VERSION)
VERSION_MAJOR := $(shell cut -c 1 VERSION)
IMG_NAME      := ${PROJECT_NAME}
IMG_VERSION   ?= $(shell git branch --show-current)
GIT_COMMIT    ?= $(shell git rev-parse HEAD)

# Yamllint variables
YAML_FILES           := $(shell find . -path './venv_virtualedgenode' -path './vendor' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
YAML_IGNORE          := vendor, .github/workflows

# Plantuml variables
PLANTUML_JAR := plantuml.jar
PLANTUML_URL := https://sourceforge.net/projects/plantuml/files/plantuml.jar/download

# Include shared makefile
include ../common.mk

##### CI Targets #####

plant-download:
	@echo "Downloading $(PLANTUML_JAR)..."
	curl -L -o $(PLANTUML_JAR) $(PLANTUML_URL)

sequence.png: docs/ven-nio-flow-seq.puml plantuml.jar
	java -jar plantuml.jar $< -o .

lint: license shellcheck yamllint mdlint
	@$(MAKE) yaml-syntax-lint YAML_FILES="$(YAML_FILES)" YAML_IGNORE="$(YAML_IGNORE)"
	@$(MAKE) rx-yaml-input-validate
dependency-check:
	./install_packages.sh

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

#### Sub-targets ####

clean: common-clean # Clean generated files
	rm -rf ${OUT_DIR}
