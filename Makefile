SHELL := /usr/bin/env bash
GOBIN ?= $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.60.3
GOLANGCI_LINT_TOOLCHAIN ?= go1.24.9
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.cache/golangci-lint
GOCACHE_DIR ?= $(CURDIR)/.cache/go-build

.PHONY: lint lint-fix test ci install-golangci-lint

install-golangci-lint:
	@set -euo pipefail; \
	if [[ ! -x "$(GOLANGCI_LINT)" ]] || ! $(GOLANGCI_LINT) --version | grep -q "$(GOLANGCI_LINT_VERSION)"; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)"; \
		GOTOOLCHAIN="$(GOLANGCI_LINT_TOOLCHAIN)" GOBIN="$(GOBIN)" go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	else \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) already installed"; \
	fi

lint: install-golangci-lint
	@mkdir -p "$(GOLANGCI_LINT_CACHE)"
	@mkdir -p "$(GOCACHE_DIR)"
	GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE)" GOCACHE="$(GOCACHE_DIR)" $(GOLANGCI_LINT) run ./...

lint-fix: install-golangci-lint
	@mkdir -p "$(GOLANGCI_LINT_CACHE)"
	@mkdir -p "$(GOCACHE_DIR)"
	GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE)" GOCACHE="$(GOCACHE_DIR)" $(GOLANGCI_LINT) run --fix ./...

test:
	@mkdir -p "$(GOCACHE_DIR)"
	GOCACHE="$(GOCACHE_DIR)" go test ./...

ci: lint test
