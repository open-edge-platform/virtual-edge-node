# Edge Node Simulator Makefile

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
# # Required Go Dependencies
GOLINT              := true
GOJUNITREPORT       := true
GOCOBERTURA         := true
PROTOCGENDOC        := true
BUF                 := true
PROTOCGENGO         := true
PROTOCGENGOGRPC     := true

# Project variables
PROJECT_NAME  := ensim
BINARY_NAME   := ensim

# Versioning variables
VERSION          := $(shell cat VERSION)

# Test variables
# Set TEST_TARGET to '<testname1>' or '<testname1\|testname2>' to run specific tests or use other regex '<testname.*>' - example:
#  TEST_TARGET='TestAuth.*' make test
TEST_TARGET      ?= .
# default TEST_PKG, overwrite to select particular test pakages
TEST_PKG         ?= $(shell go list ./... | grep -v "pkg/api" | grep -v "/test" | grep -v "cmd" | grep -v "pkg/en/onboard/proto" | tr '\n' ,)
TEST_COVER       := atomic ./pkg/...

# Include shared makefile
include ../common.mk

# Path for HELM script
TOOLS_DIR     := ../tools

# YAML 
YAML_MAX_LENGTH := 99
YAML_IGNORE := vendor, .github/workflows, $(VENV_NAME), charts/

out_custom:
	mkdir -p ./$(OUT_DIR)/ensim
	mkdir -p ./$(OUT_DIR)/ensim/server
	mkdir -p ./$(OUT_DIR)/ensim/client

#### Development Targets ####
generate: buf-generate ## generate all files

build: go-build ## Build local binaries

lint: license yamllint hadolint buf-lint go-lint helm-lint helm-version-check mdlint ## Run all lint tools

test: go-test ## Run all unit tests

#### Sub-targets ####

go-build: $(OUT_DIR) out_custom ## build resource manager binary
	$(GOCMD) build $(COMMON_GOEXTRAFLAGS) -o $(OUT_DIR)/ensim/server/main cmd/ensim/server/main.go && \
	$(GOCMD) build $(COMMON_GOEXTRAFLAGS) -o $(OUT_DIR)/ensim/client/main cmd/ensim/client/main.go

go-test: $(OUT_DIR) ## Run go test and calculate code coverage
	$(GOCMD) test -race -v -p 1 \
	-coverpkg=$(TEST_PKG) $(TEST_TARGET) \
	-coverprofile=$(OUT_DIR)/coverage.out \
	-covermode $(TEST_COVER) \
    -args -globalLogLevel=error
	| tee >(go-junit-report -set-exit-code > $(OUT_DIR)/report.xml)
	gocover-cobertura $(if $(TEST_IGNORE_FILES),-ignore-files $(TEST_IGNORE_FILES)) < $(OUT_DIR)/coverage.out > $(OUT_DIR)/coverage.xml
	$(GOCMD) tool cover -html=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(OUT_DIR)/coverage.out -o $(OUT_DIR)/function_coverage.log

docker-build: go-build ## @HELP Build ensim docker image
	$(GOCMD) mod vendor
	cp ../common.mk ../version.mk .
	docker build $(DOCKER_BUILD_ARGS) \
                -t $(DOCKER_TAG) \
                -f ./Dockerfile .
	@rm -rf vendor common.mk version.mk

docker-push: docker-build ## tag and push Docker image
	docker push $(DOCKER_TAG)

KIND_CLUSTER_NAME="kind"
kind-load: ## @HELP Kind load ensim docker image
	kind load docker-image $(DOCKER_TAG) --name=${KIND_CLUSTER_NAME}

HELM_CHART_EN_SIM = $(shell pwd)/charts/
helm-package: ## @HELP package the helm chart en-sim
	helm dep update $(HELM_CHART_EN_SIM)
	helm package --app-version=$(VERSION) --version=$(VERSION) --debug -u $(HELM_CHART_EN_SIM)

helm_version_ensim = $(shell helm show chart $(HELM_CHART_EN_SIM) | yq e '.version' -)
helm_app_version_ensim = $(shell helm show chart $(HELM_CHART_EN_SIM) | yq e '.appVersion' -)
helm-version-check: ## @HELP validates that the version is the same in the VERSION file and in the helm-chart
	@echo "Verify that VERSION ($(VERSION)) matches Helm Chart Version ($(helm_version_ensim)) and Helm Chart App Version ($(helm_app_version_ensim))"
	@bash -c "diff -u <(echo $(VERSION)) <(echo $(helm_version_ensim))"
	@bash -c "diff -u <(echo $(VERSION)) <(echo $(helm_app_version_ensim))"

helm-build: helm-package

helm-push: ## @HELP Push helm charts.
	chartToPush=`find . -name ensim-$(VERSION).tgz`; \
	chartVersion=`echo $$chartToPush | sed -n 's/.*$*-\(.*\).tgz/\1/p'`; \
	isChartDev=`echo $$chartVersion | grep -c dev || true`; \
	chartExist=`helm show chart oci://$(HELM_REGISTRY)/$(HELM_REPOSITORY)/ensim --version $${chartVersion} || true`; \
	if [ -z "$$chartExist" ] || [ "$$isChartDev" -eq 1 ]; then \
		echo "Push new chart: chart=$* version=$${chartVersion}"; \
		helm push $${chartToPush} oci://$(HELM_REGISTRY)/$(HELM_REPOSITORY)/; \
	fi


PROJECT                 ?=      ""
CA_PATH                 ?=      ""
SIM_ADDRESS             ?=      ""
ORCH_FQDN               ?=      ""
int-tests: int-test-day0 int-test-day1 int-test-day2

int-test-day0:
	ginkgo -v -r --fail-fast --race \
                --label-filter="infra-tests-day0" \
                ./test/infra \
                -- -project=$(PROJECT) -caFilepath=$(CA_PATH) -simAddress=$(SIM_ADDRESS) -clusterFQDN=$(ORCH_FQDN)

int-test-day1:
	ginkgo -v -r \
                --label-filter="infra-tests-day1" \
                ./test/infra \
                -- -project=$(PROJECT) -caFilepath=$(CA_PATH) -simAddress=$(SIM_ADDRESS) -clusterFQDN=$(ORCH_FQDN)

int-test-day2:
	ginkgo -v -r \
                --label-filter="infra-tests-day2" \
                ./test/infra \
                -- -project=$(PROJECT) -caFilepath=$(CA_PATH) -simAddress=$(SIM_ADDRESS) -clusterFQDN=$(ORCH_FQDN)
