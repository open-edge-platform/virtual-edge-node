# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SUBPROJECTS := vm-provisioning edge-node-simulator edge-node-container

.DEFAULT_GOAL := help
.PHONY: all clean clean-all help lint test build license

all: lint mdlint build
	@# Help: Runs build, lint, test stages for all subprojects


#### Python venv Target ####
VENV_DIR := venv_ven

$(VENV_DIR): requirements.txt ## Create Python venv
	python3 -m venv $@ ;\
  set +u; . ./$@/bin/activate; set -u ;\
  python -m pip install --upgrade pip ;\
  python -m pip install -r requirements.txt

dependency-check: $(VENV_DIR)

license:
	@echo "---LICENSE CHECK---"
	@for dir in $(SUBPROJECTS); do $(MAKE) -C $$dir license; done
	@echo "---END LICENSE CHECK---"

lint:
	@# Help: Runs lint stage in all subprojects
	@echo "---MAKEFILE LINT---"
	@for dir in $(SUBPROJECTS); do $(MAKE) -C $$dir lint; done
	@echo "---END MAKEFILE LINT---"

build:
	@# Help: Runs build stage in all subprojects
	@echo "---MAKEFILE BUILD---"
	for dir in $(SUBPROJECTS); do $(MAKE) -C $$dir build; done
	@echo "---END MAKEFILE Build---"

mdlint:
	@echo "---MAKEFILE LINT README---"
	@for dir in $(SUBPROJECTS); do $(MAKE) -C $$dir mdlint; done
	@echo "---END MAKEFILE LINT README---"

clean:
	@# Help: Runs clean stage in all subprojects
	@echo "---MAKEFILE CLEAN---"
	for dir in $(SUBPROJECTS); do $(MAKE) -C $$dir clean; done
	@echo "---END MAKEFILE CLEAN---"

clean-all:
	@# Help: Runs clean-all stage in all subprojects
	@echo "---MAKEFILE CLEAN-ALL---"
	for dir in $(SUBPROJECTS); do $(MAKE) -C $$dir clean-all; done
	@echo "---END MAKEFILE CLEAN-ALL---"

help:	
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
