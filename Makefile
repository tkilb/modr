SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

BINDIR ?= $(HOME)/bin
INSTALL_BIN := $(BINDIR)/modr
BIN := ./bin/modr
LEGACY_BIN := ./bin/rimmod
LEGACY_INSTALL_BIN := $(BINDIR)/rimmod

.PHONY: help build test status apply apply-dry-run update update-dry-run \
        tidy tidy-dry-run import import-dry-run order deck-push deck-push-dry-run deck-check \
        deck-pull deck-pull-dry-run dev dev-dry-run \
        install install-dry-run uninstall uninstall-dry-run \
        sync-deck sync-deck-dry-run sync-deck-check import-deck import-deck-dry-run \
        build-load-order build-mod

##@ Build & Test
build: ## Compile the modr and rimmod CLIs
	@mkdir -p bin
	@go build -o $(BIN) ./cmd/modr
	@go build -o $(LEGACY_BIN) ./cmd/rimmod

test: ## Run the Go test suite
	@go test ./...

$(BIN):
	@$(MAKE) build

##@ Installation
install: $(BIN) ## Install modr CLI to $(BINDIR)/modr (default: ~/bin/modr)
ifeq ($(findstring n,$(firstword -$(MAKEFLAGS))),n)
	@echo "  [DRY-RUN] Would install $(abspath $(BIN)) -> $(INSTALL_BIN)"
	@echo "  [DRY-RUN] Would symlink $(INSTALL_BIN) -> $(LEGACY_INSTALL_BIN)"
else
	@mkdir -p $(BINDIR)
	@ln -sf $(abspath $(BIN)) $(INSTALL_BIN)
	@ln -sf $(INSTALL_BIN) $(LEGACY_INSTALL_BIN)
	@echo "Installed $(INSTALL_BIN) -> $(abspath $(BIN))"
	@echo "Linked compatibility alias $(LEGACY_INSTALL_BIN) -> $(INSTALL_BIN)"
endif

install-dry-run: ## Preview installing modr CLI
	@echo "  [DRY-RUN] Would install $(abspath $(BIN)) -> $(INSTALL_BIN)"
	@echo "  [DRY-RUN] Would symlink $(INSTALL_BIN) -> $(LEGACY_INSTALL_BIN)"

uninstall: ## Remove installed modr and rimmod CLIs from $(BINDIR)
ifeq ($(findstring n,$(firstword -$(MAKEFLAGS))),n)
	@echo "  [DRY-RUN] Would remove $(INSTALL_BIN) and $(LEGACY_INSTALL_BIN)"
else
	@rm -f $(INSTALL_BIN) $(LEGACY_INSTALL_BIN)
	@echo "Removed $(INSTALL_BIN) and $(LEGACY_INSTALL_BIN)"
endif

uninstall-dry-run: ## Preview removing installed rimmod CLI
	@echo "  [DRY-RUN] Would remove $(INSTALL_BIN)"

##@ Core Workflows
apply: $(BIN) ## Reconcile local game (fetch missing, link, sort, write config)
	@$(BIN) apply $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

apply-dry-run: $(BIN) ## Preview local reconciliation without modifying filesystem
	@$(BIN) apply --dry-run

update: $(BIN) ## Check upstream mod updates, refresh lockfile, and re-apply
	@$(BIN) update $(if $(MOD),--mod $(MOD),) $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

update-dry-run: $(BIN) ## Preview upstream mod updates
	@$(BIN) update $(if $(MOD),--mod $(MOD),) --dry-run

tidy: $(BIN) ## Prune unreferenced lock entries, vendor cache, and dead symlinks
	@$(BIN) tidy $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

tidy-dry-run: $(BIN) ## Preview tidy cleanup without deleting files
	@$(BIN) tidy --dry-run

import: $(BIN) ## Import a new mod by URL (e.g. make import URL="https://...")
	@$(BIN) import $(URL) $(if $(SLUG),--slug $(SLUG),) $(if $(APPLY),--apply,) $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

import-dry-run: $(BIN) ## Preview importing a new mod by URL
	@$(BIN) import $(URL) $(if $(SLUG),--slug $(SLUG),) --dry-run

##@ Steam Deck Sync
deck-push: $(BIN) ## Rsync manifests/mods to Steam Deck and apply remotely
	@$(BIN) deck push $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

deck-push-dry-run: $(BIN) ## Preview sync to Steam Deck
	@$(BIN) deck push --dry-run

deck-check: $(BIN) ## Test SSH connectivity to Steam Deck
	@$(BIN) deck push --check

deck-pull: $(BIN) ## Interactive TUI to import Workshop subscriptions from Steam Deck
	@$(BIN) deck pull $(if $(COPY),--copy,) $(if $(ALL),--all,) $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

deck-pull-dry-run: $(BIN) ## Preview importing mods from Steam Deck
	@$(BIN) deck pull --dry-run

##@ Private Mod Development
dev: $(BIN) ## Compile custom mod and auto-apply (Usage: make dev MOD=<name>)
	@$(BIN) dev build $(if $(MOD),$(MOD),organic-constructs) $(if $(findstring n,$(firstword -$(MAKEFLAGS))),--dry-run,)

dev-dry-run: $(BIN) ## Preview compiling custom mod
	@$(BIN) dev build $(if $(MOD),$(MOD),organic-constructs) --dry-run

##@ Diagnostics & Inspection
order: $(BIN) ## Inspect computed topological mod load order
	@$(BIN) order

status: $(BIN) ## Inspect monorepo health, active symlinks, and Deck status
	@$(BIN) status

help: $(BIN) ## Display rimmod CLI help and reference
	@$(BIN) help

##@ Compatibility Aliases
sync-deck: deck-push
sync-deck-dry-run: deck-push-dry-run
sync-deck-check: deck-check
import-deck: deck-pull
import-deck-dry-run: deck-pull-dry-run
build-load-order: build
build-mod: dev
