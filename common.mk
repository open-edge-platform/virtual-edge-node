# common.mk - common targets for virtual-edge-node repository

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Makefile Style Guide:
# - Help will be generated from ## comments at end of any target line
# - Use smooth parens $() for variables over curly brackets ${} for consistency
# - Continuation lines (after an \ on previous line) should start with spaces
#   not tabs - this will cause editor highlighting to point out editing mistakes
# - When creating targets that run a lint or similar testing tool, print the
#   tool version first so that issues with versions in CI or other remote
#   environments can be caught

# Optionally include tool version checks, not used in Docker builds
ifeq ($(TOOL_VERSION_CHECK), 1)
	include ../version.mk
endif

# GO variables
GOARCH	:= $(shell go env GOARCH)
GOCMD   := go

# Docker variables
IMG_VERSION             ?= $(shell git branch --show-current)
GIT_COMMIT              ?= $(shell git rev-parse HEAD)
DOCKER_IMG_NAME         := $(PROJECT_NAME)
DOCKER_VERSION          ?= $(shell git branch --show-current)
DOCKER_ENV              := DOCKER_BUILDKIT=1
OCI_REGISTRY            ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
OCI_REPOSITORY          ?= edge-orch/infra
HELM_REGISTRY           ?= $(OCI_REGISTRY)
HELM_REPOSITORY         ?= $(OCI_REPOSITORY)/charts
DOCKER_REGISTRY         ?= $(OCI_REGISTRY)
DOCKER_REPOSITORY       ?= $(OCI_REPOSITORY)
DOCKER_TAG              := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_IMG_NAME):$(VERSION)
DOCKER_TAG_BRANCH	    := $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY)/$(DOCKER_IMG_NAME):$(DOCKER_VERSION)

# Decides if we shall push image tagged with the branch name or not.
DOCKER_TAG_BRANCH_PUSH	?= true
LABEL_REPO_URL          ?= $(shell git remote get-url $(shell git remote | head -n 1))
LABEL_VERSION           ?= $(VERSION)
LABEL_REVISION          ?= $(GIT_COMMIT)
LABEL_BUILD_DATE        ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

#### Docker Config & Targets ####
# Docker variables
DOCKER_EXTRA_ARGS       ?= \
        --build-arg http_proxy="$(HTTP_PROXY)" --build-arg https_proxy="$(HTTPS_PROXY)" \
        --build-arg no_proxy="$(NO_PROXY)" --build-arg HTTP_PROXY="$(HTTP_PROXY)" \
        --build-arg HTTPS_PROXY="$(HTTPS_PROXY)" --build-arg NO_PROXY="$(NO_PROXY)"
DOCKER_LABEL_ARGS       ?= \
        --build-arg REPO_URL="$(LABEL_REPO_URL)" \
		--build-arg VERSION="$(LABEL_VERSION)" \
		--build-arg REVISION="$(LABEL_REVISION)" \
		--build-arg BUILD_DATE="$(LABEL_BUILD_DATE)"
DOCKER_BUILD_ARGS       ?= \
        ${DOCKER_LABEL_ARGS} \
        ${DOCKER_EXTRA_ARGS}

#### Variables ####
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

# Path variables
OUT_DIR	   := out
SECRETS_DIR := /var/run/secrets
SCRIPTS_DIR := ./ci_scripts

$(OUT_DIR): ## Create out directory
	mkdir -p $(OUT_DIR)

#### Python venv Target ####
VENV_NAME	:= venv_$(PROJECT_NAME)

$(VENV_NAME): requirements.txt
	python3 -m venv $@ ;\
  set +u; . ./$@/bin/activate; set -u ;\
  python -m pip install --upgrade pip ;\
  python -m pip install -r requirements.txt

#### Lint and Validator Targets ####
# https://github.com/koalaman/shellcheck
SH_FILES := $(shell find . -type f \( -name '*.sh' \) -print )
shellcheck: ## lint shell scripts with shellcheck
	shellcheck --version
	shellcheck -x -S style $(SH_FILES)

# https://pypi.org/project/reuse/
license: $(VENV_NAME) ## Check licensing with the reuse tool
	set +u; . ./$</bin/activate; set -u ;\
  	reuse --version ;\
  	reuse --root . lint 

YAML_MAX_LENGTH := 250
YAML_FILES := $(shell find . -type f \( -name '*.yaml' -o -name '*.yml' \) -print )
YAML_IGNORE ?= vendor, .github/workflows, $(VENV_NAME)
yamllint: $(VENV_NAME) ## lint YAML files
	. ./$</bin/activate; set -u ;\
  yamllint --version ;\
  yamllint -d '{extends: default, rules: {line-length: {max: $(YAML_MAX_LENGTH)}, truthy: disable}, ignore: [$(YAML_IGNORE)]}' -s $(YAML_FILES)

yaml-syntax-lint: $(VENV_NAME) ## Validate YAML files using a custom Python script
	@mkdir -p out
	@. ./$</bin/activate; set -u ;\
	python3 ./tools/yaml-syntax-check.py $(YAML_FILES) --ignore $(YAML_IGNORE) > out/yaml-syntax-report.log 2>&1
	@echo "Validation logs stored in out/yaml-syntax-report.log"

mdlint: ## link MD files
	markdownlint --version ;\
	markdownlint "**/*.md" -c ../.markdownlint.yml --ignore venv_virtualedgenode/

common-clean:
	rm -rf ${OUT_DIR} vendor

clean-venv:
	rm -rf "$(VENV_NAME)"

clean-all: clean clean-venv ## delete all built artifacts and downloaded tools

go-tidy: ## Run go mod tidy
	$(GOCMD) mod tidy

go-lint: $(OUT_DIR) ## Lint go code with golangci-lint
	golangci-lint --version
	golangci-lint run --config .golangci.yml

go-lint-fix: ## Apply automated lint/formatting fixes to go files
	golangci-lint run --fix --config .golangci.yml

# Security config for Go Builds - see:
#   https://readthedocs.intel.com/SecureCodingStandards/latest/compiler/golang/
# -trimpath: Remove all file system paths from the resulting executable.
# -gcflags="all=-m": Print optimizations applied by the compiler for review and verification against security requirements.
# -gcflags="all=-spectre=all" Enable all available Spectre mitigations
# -ldflags="all=-s -w" remove the symbol and debug info
# -ldflags="all=-X ..." Embed binary build stamping information
ifeq ($(GOARCH),arm64)
	# Note that arm64 (Apple, similar) does not support any spectre mititations.
  COMMON_GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w -X 'main.RepoURL=$(LABEL_REPO_URL)' -X 'main.Version=$(LABEL_VERSION)' -X 'main.Revision=$(LABEL_REVISION)' -X 'main.BuildDate=$(LABEL_BUILD_DATE)'"
else
  COMMON_GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X 'main.RepoURL=$(LABEL_REPO_URL)' -X 'main.Version=$(LABEL_VERSION)' -X 'main.Revision=$(LABEL_REVISION)' -X 'main.BuildDate=$(LABEL_BUILD_DATE)'"
endif

# https://github.com/slimm609/checksec.sh
# checks various security properties on executoables, such as RELRO, STACK CANARY, NX, PIE, etc.
checksec: go-build ## Check security properties on executables
	checksec --output=json --file=$(OUT_DIR)/$(BINARY_NAME)
	checksec --fortify-file=$(OUT_DIR)/$(BINARY_NAME)

#### Buf protobuf code generation tooling ###

APIPKG_DIR ?= pkg/api

buf-update: $(VENV_NAME) ## update buf modules
	set +u; . ./$</bin/activate; set -u ;\
  buf --version ;\
  pushd api; buf dep update; popd ;\
  buf build

buf-generate: $(VENV_NAME) ## compile protobuf files in api into code
	set +u; . ./$</bin/activate; set -u ;\
  buf --version ;\
  buf generate

buf-lint: $(VENV_NAME) ## Lint and format protobuf files
	buf --version
	buf format -d --exit-code
	buf lint

buf-lint-fix: $(VENV_NAME) ## Lint and when possible fix protobuf files
	buf --version
	buf format -d -w
	buf lint

#### Lint Targets ####
# https://github.com/hadolint/hadolint/
hadolint: ## Check Dockerfile with Hadolint
	hadolint --version
	hadolint Dockerfile

TOOLS_DIR     := ./tools

helm-lint: ## lint all helm charts
	$(TOOLS_DIR)/helmlint.sh

helm-clean: ## lint all helm charts, cleaning first
	$(TOOLS_DIR)/helmlint.sh clean

#### Help Target ####
help: ## Print help for each target
	@echo $(PROJECT_NAME) make targets
	@echo "Target               Makefile:Line    Description"
	@echo "-------------------- ---------------- -----------------------------------------"
	@grep -H -n '^[[:alnum:]_-]*:.* ##' $(MAKEFILE_LIST) \
    | sort -t ":" -k 3 \
    | awk 'BEGIN  {FS=":"}; {sub(".* ## ", "", $$4)}; {printf "%-20s %-16s %s\n", $$3, $$1 ":" $$2, $$4};'

rx-yaml-input-validate:
	@mkdir -p out
	@echo "Running YAML validation..."
	python3 ./tools/yaml_validator/yaml_validator.py ./ansible > out/rx-yaml-input-validate.log 2>&1
	@echo "Validation logs stored in out/rx-yaml-input-validate.log"
