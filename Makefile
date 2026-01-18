# Zero Maintenance Philosophy:
# - Dependencies are vendored locally (`go mod vendor`) to ensure reproducible builds forever.
# - CI tools (goreleaser) are pinned to specific major versions to prevent breaking changes.
# - Please maintain this "self-contained" approach.

.PHONY: build build-all test release release-simulate clean install uninstall help

# Default target
.DEFAULT_GOAL := help

build: ## Build the application for the current system only (incremental builds)
	go build -o panforge ./cmd/panforge

build-all: ## Build for all target systems (requires goreleaser)
	goreleaser build --snapshot --clean

install: ## Install the application
	go install ./cmd/panforge

test: ## Run all tests
	go test -v ./...

release: ## Create a release and push it to GitHub (requires goreleaser)
	goreleaser release --clean

release-simulate: ## Create a simulated release (for local testing) and build for all target systems (requires goreleaser)
	goreleaser release --snapshot --skip-publish --clean

clean: ## Clean up build artifacts and remove the binary
	rm -f panforge
	rm -rf dist

uninstall: ## Uninstall the application
	go clean -i ./cmd/panforge

tag: ## Create a new git tag and push it. Usage: make tag v=v1.0.0
	@if [ -z "$(v)" ]; then \
		echo "Error: version argument 'v' is required."; \
		echo "Usage: make tag v=v1.0.0"; \
		echo ""; \
		echo "Current version: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'none')"; \
		exit 1; \
	fi
	git tag -a $(v) -m "Release $(v)"
	git push origin $(v)

help: ## Show this help message
	@echo 'Usage: make [target] ...'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
