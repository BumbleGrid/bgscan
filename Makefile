# Kind clusters and manifests under testdata/clusters/
# Requires: kind, kubectl, bash

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
TESTDATA_DIR := $(ROOT)/testdata
SCENARIOS_SH := $(TESTDATA_DIR)/scenarios.sh

CLUSTER_SCENARIO_DIRS := $(wildcard $(TESTDATA_DIR)/clusters/*)
SCENARIOS := $(sort $(foreach d,$(CLUSTER_SCENARIO_DIRS),$(if $(wildcard $(d)/scenario.env),$(notdir $(d)),)))

.PHONY: help kind-list \
	kind-apply kind-apply-recreate kind-delete kind-delete-cluster \
	kind-apply-all kind-delete-all kind-delete-cluster-all \
	FORCE

.DEFAULT_GOAL := help

help:
	@echo "Kind provisioning via $(TESTDATA_DIR)"
	@echo ""
	@echo "Discover scenarios:"
	@echo "  make kind-list"
	@echo ""
	@echo "Provision cluster + apply manifests:"
	@echo "  make kind-apply SCENARIO=simple"
	@echo "  make kind-apply SCENARIO=complex"
	@echo "  make kind-apply-recreate SCENARIO=<name>"
	@echo ""
	@echo "Shortcuts (current: $(words $(SCENARIOS)) scenarios):"
	@if [ -n "$(strip $(SCENARIOS))" ]; then \
		for s in $(SCENARIOS); do echo "  make kind-apply-$$s | kind-apply-recreate-$$s | kind-delete-$$s | kind-delete-cluster-$$s"; done; \
	else \
		echo "  (none — add testdata/clusters/<name>/ with scenario.env + manifests/)"; \
	fi
	@echo ""
	@echo "All scenarios (sequential):"
	@echo "  make kind-apply-all"
	@echo "  make kind-delete-all"
	@echo "  make kind-delete-cluster-all   # delete manifests + destroy all scenario clusters"
	@echo ""
	@echo "Paths: $(TESTDATA_DIR)/clusters/simple, $(TESTDATA_DIR)/clusters/complex"

kind-list:
	@"$(SCENARIOS_SH)" list

kind-apply:
	@test -n "$(SCENARIO)" || (echo "usage: make kind-apply SCENARIO=simple|complex"; exit 1)
	@"$(SCENARIOS_SH)" apply "$(SCENARIO)"

kind-apply-recreate:
	@test -n "$(SCENARIO)" || (echo "usage: make kind-apply-recreate SCENARIO=simple|complex"; exit 1)
	@"$(SCENARIOS_SH)" apply "$(SCENARIO)" --recreate

kind-delete:
	@test -n "$(SCENARIO)" || (echo "usage: make kind-delete SCENARIO=simple|complex"; exit 1)
	@"$(SCENARIOS_SH)" delete "$(SCENARIO)"

kind-delete-cluster:
	@test -n "$(SCENARIO)" || (echo "usage: make kind-delete-cluster SCENARIO=simple|complex"; exit 1)
	@"$(SCENARIOS_SH)" delete "$(SCENARIO)" --delete-cluster

kind-apply-all:
	@set -e; for s in $(SCENARIOS); do "$(SCENARIOS_SH)" apply "$$s"; done

kind-delete-all:
	@set -e; for s in $(SCENARIOS); do "$(SCENARIOS_SH)" delete "$$s"; done

kind-delete-cluster-all:
	@set -e; for s in $(SCENARIOS); do "$(SCENARIOS_SH)" delete "$$s" --delete-cluster; done

define SCENARIO_KIND_RULES
.PHONY: kind-apply-$(1) kind-apply-recreate-$(1) kind-delete-$(1) kind-delete-cluster-$(1)

kind-apply-$(1):
	@"$(SCENARIOS_SH)" apply "$(1)"

kind-apply-recreate-$(1):
	@"$(SCENARIOS_SH)" apply "$(1)" --recreate

kind-delete-$(1):
	@"$(SCENARIOS_SH)" delete "$(1)"

kind-delete-cluster-$(1):
	@"$(SCENARIOS_SH)" delete "$(1)" --delete-cluster

endef
$(foreach s,$(SCENARIOS),$(eval $(call SCENARIO_KIND_RULES,$(s))))

FORCE:
