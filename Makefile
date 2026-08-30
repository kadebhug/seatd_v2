SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

.PHONY: help bootstrap check generate fmt lint test build start start-frontends clean

help: ## List supported local commands.
	@awk 'BEGIN {FS = ":.*## "; printf "Seatd development commands:\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

bootstrap: ## Prepare a clean checkout for local development.
	@./scripts/bootstrap.sh

check: ## Run repository structure and policy checks.
	@./scripts/check-repository.sh

generate: ## Regenerate contract-derived clients.
	@./scripts/generate-clients.sh

fmt: ## Format all projects (hooks are added with application skeletons).
	@./scripts/run-workspace-command.sh fmt

lint: ## Lint all projects (hooks are added with application skeletons).
	@./scripts/run-workspace-command.sh lint

test: ## Test all projects (hooks are added with application skeletons).
	@./scripts/run-workspace-command.sh test

build: ## Build all projects (hooks are added with application skeletons).
	@./scripts/run-workspace-command.sh build

start: ## Start the api, worker, and realtime backends.
	@./scripts/start-backends.sh

start-frontends: ## Start the web and guest frontends.
	@./scripts/start-frontends.sh

clean: ## Remove generated local build output.
	@./scripts/run-workspace-command.sh clean
