# Edge Node in a Container Makefile

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help
.PHONY: help build test lint

#### Variables ####

# Optionally include tool version checks, not used in Docker builds
TOOL_VERSION_CHECK ?= 1
#
# ##### Variables #####
#

# Project variables
PROJECT_NAME  := enic
BINARY_NAME   := enic
UTILS_NAME    := utils

# Versioning variables
VERSION          := $(shell cat VERSION)
ENA_VERSION      ?= $(shell cat EN_MANIFEST_VERSION)

# Params for docker uuids folder and uuid
DOCKER_UUIDS_FOLDER  = $(shell pwd)/uuids
DOCKER_UUID         ?= b9f27de1-40a4-4abf-9035-2a58eaf6d11f
DOCKER_VOL_PATH      = $(DOCKER_UUIDS_FOLDER)/$(DOCKER_UUID)
DOCKER_CREDS_FOLDER  = $(shell pwd)/creds

# Include shared makefile
include ../common.mk

ENIC_DOCKER_TAG              := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_IMG_NAME):$(VERSION)
UTILS_DOCKER_TAG             := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_IMG_NAME)-$(UTILS_NAME):$(VERSION)

# Path for HELM script
TOOLS_DIR     := ../tools

# YAML 
YAML_MAX_LENGTH := 99
YAML_IGNORE := vendor, .github/workflows, $(VENV_NAME), chart/

#### Development Targets ####
build: go-build ## Build local binaries

lint: license yamllint hadolint-custom go-lint helm-lint helm-version-check mdlint ## Run all lint tools

# https://github.com/hadolint/hadolint/
hadolint-custom: ## Check Dockerfiles with Hadolint
	hadolint --version
	hadolint Dockerfile --ignore DL3008 --ignore DL4006 --ignore DL3015
	hadolint Dockerfile.utils --ignore DL3008

#### Sub-targets ####

go-build: $(OUT_DIR) ## Run go build
	$(GOCMD) build $(COMMON_GOEXTRAFLAGS) -o $(OUT_DIR)/$(BINARY_NAME) cmd/main.go

docker-build: docker-build-enic docker-build-utils ##  Build ENiC docker image

docker-build-enic: bma_packages ##  Build ENiC docker image
	$(GOCMD) mod vendor
	cp ../common.mk ../version.mk .
	docker build $(DOCKER_BUILD_ARGS) \
                -t $(ENIC_DOCKER_TAG) \
                -f ./Dockerfile .
	@rm -rf vendor common.mk version.mk

docker-build-utils: ##  Build ENiC utils docker image
	cp ../common.mk ../version.mk .
	docker build $(DOCKER_BUILD_ARGS) \
                -t $(UTILS_DOCKER_TAG) \
                -f ./Dockerfile.utils .
	@rm -rf common.mk version.mk

HELM_CHART_ENIC = $(shell pwd)/chart
helm-package: ##  package the helm chart en-sim
	helm dep update $(HELM_CHART_ENIC)
	helm package --app-version=$(VERSION) --version=$(VERSION) --debug -u $(HELM_CHART_ENIC)

helm_version_enic = $(shell helm show chart $(HELM_CHART_ENIC) | yq e '.version' -)
helm_app_version_enic = $(shell helm show chart $(HELM_CHART_ENIC) | yq e '.appVersion' -)
helm-version-check: ##  validates that the version is the same in the VERSION file and in the helm-chart
	@echo "Verify that VERSION ($(VERSION)) matches Helm Chart Version ($(helm_version_enic)) and Helm Chart App Version ($(helm_app_version_enic))"
	@bash -c "diff -u <(echo $(VERSION)) <(echo $(helm_version_enic))"
	@bash -c "diff -u <(echo $(VERSION)) <(echo $(helm_app_version_enic))"

helm-build: bma_versions helm-package

helm-push: ##  Push helm charts.
	chartToPush=`find . -name enic-$(VERSION).tgz`; \
	chartVersion=`echo $$chartToPush | sed -n 's/.*$*-\(.*\).tgz/\1/p'`; \
	isChartDev=`echo $$chartVersion | grep -c dev || true`; \
	chartExist=`helm show chart oci://$(HELM_REGISTRY)/$(HELM_REPOSITORY)/enic --version $${chartVersion} || true`; \
	if [ -z "$$chartExist" ] || [ "$$isChartDev" -eq 1 ]; then \
		echo "Push new chart: chart=$* version=$${chartVersion}"; \
		helm push $${chartToPush} oci://$(HELM_REGISTRY)/$(HELM_REPOSITORY)/; \
	fi

$(DOCKER_CREDS_FOLDER): ##  Create credentials folder
	mkdir -p $(DOCKER_CREDS_FOLDER)

bma-manifest:
	oras pull "registry-rs.edgeorchestration.intel.com/edge-orch/en/files/ena-manifest:$(ENA_VERSION)"

bma_packages: $(VENV_NAME) bma-manifest
	python3 ./utils/bma_utils.py < ena-manifest.yaml

docker-push: ##  Push docker images
	docker push $(ENIC_DOCKER_TAG)
	docker push $(UTILS_DOCKER_TAG)

docker-clean: ##  Remove the running container
	docker rm -f enic enic-utils
	rm -rf $(DOCKER_VOL_PATH)
	sudo rm -rf $(DOCKER_CREDS_FOLDER)

docker-gen-uuid: ##  Creates dmi/uuid files for docker container
	bash ./scripts/fake-uuid.sh \
        $(shell pwd)/dmiTemplate/dmi \
        $(shell pwd)/dmiTemplate/dmi-dump \
        $(DOCKER_VOL_PATH) \
        $(DOCKER_UUID)

image-export: ##  Export ENiC docker image into tar
	docker save -o edge-node.tar.gz $(ENIC_DOCKER_TAG)
	docker save -o edge-node-utils.tar.gz $(UTILS_DOCKER_TAG)

KIND_CLUSTER_NAME="kind"
kind-load: ##  Kind load ENiC docker image
	kind load docker-image $(ENIC_DOCKER_TAG) --name=$(KIND_CLUSTER_NAME)
	kind load docker-image $(UTILS_DOCKER_TAG) --name=$(KIND_CLUSTER_NAME)

SH_FILES := $(shell find . -type f -name '*.sh')
lint: ##  Lint shell utils scripts
	shellcheck $(SH_FILES)

bma_versions: ##  Sets the currently downloaded BMA versions in the value file
	yq eval -i '.bma_versions.caddy_version = "$(shell cat ./chart/bma_values.yaml | yq .caddy_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.cluster_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .cluster_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.hardware_discovery_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .hardware_discovery_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.inbc_program_version = "$(shell cat ./chart/bma_values.yaml | yq .inbc_program_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.inbm_cloudadapter_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .inbm_cloudadapter_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.inbm_configuration_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .inbm_configuration_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.inbm_diagnostic_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .inbm_diagnostic_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.inbm_dispatcher_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .inbm_dispatcher_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.inbm_telemetry_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .inbm_telemetry_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.mqtt_version = "$(shell cat ./chart/bma_values.yaml | yq .mqtt_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.node_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .node_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.platform_observability_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .platform_observability_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.platform_telemetry_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .platform_telemetry_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.platform_update_agent_version = "$(shell cat ./chart/bma_values.yaml | yq .platform_update_agent_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.tpm_provision_version = "$(shell cat ./chart/bma_values.yaml | yq .tpm_provision_version)"' ./chart/values.yaml
	yq eval -i '.bma_versions.trtl_version = "$(shell cat ./chart/bma_values.yaml | yq .trtl_version)"' ./chart/values.yaml

apply-version: ##helm-reset-annotations helm-annotate	bma_versions ## @HELP apply version from the top level package.json to all sub-projectsare the same across the different projects
	yq eval -i '.version = "$(VERSION)"' $(HELM_CHART_ENIC)/Chart.yaml
	yq eval -i '.appVersion = "$(VERSION)"' $(HELM_CHART_ENIC)/Chart.yaml
