# Zero Maintenance Philosophy:
# - Dependencies are vendored locally (`go mod vendor`) to ensure reproducible builds forever.
# - CI tools (goreleaser) are pinned to specific major versions to prevent breaking changes.
# - Please maintain this "self-contained" approach.

.PHONY: build build-all test lint pre-commit release release-simulate clean install uninstall tag help

# Default target
.DEFAULT_GOAL := help

# Handle "make tag v0.1.0"
ifeq (tag,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "tag"
  TAG_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(TAG_ARGS):;@:)
  # Assign the first argument to v if v is not already set
  v ?= $(firstword $(TAG_ARGS))
endif

build: ## Build the application for the current system only (incremental builds)
	go build -o panforge ./cmd/panforge

build-all: ## Build for all target systems (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo >&2 "goreleaser is not installed. Please install it with: go install github.com/goreleaser/goreleaser/v2@latest"; exit 1; }
	goreleaser build --snapshot --clean

install: ## Install the application
	go install ./cmd/panforge

test: ## Run all tests
	go test -v ./...

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "golangci-lint is not installed. Please install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run

pre-commit: ## Install git pre-commit hooks
	@command -v lefthook >/dev/null 2>&1 || { echo >&2 "lefthook is not installed. Please install it with: go install github.com/evilmartians/lefthook@latest"; exit 1; }
	lefthook install

release: ## Create a release and push it to GitHub (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo >&2 "goreleaser is not installed. Please install it with: go install github.com/goreleaser/goreleaser/v2@latest"; exit 1; }
	goreleaser release --clean

release-simulate: ## Create a simulated release (for local testing) and build for all target systems (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo >&2 "goreleaser is not installed. Please install it with: go install github.com/goreleaser/goreleaser/v2@latest"; exit 1; }
	goreleaser release --snapshot --clean

clean: ## Clean up build artifacts and remove the binary
	rm -f panforge
	rm -rf dist

uninstall: ## Uninstall the application
	go clean -i ./cmd/panforge

tag: ## Create a new git tag and push it. Usage: make tag v=v1.0.0 or make tag v1.0.0
	@command -v git >/dev/null 2>&1 || { echo >&2 "git is not installed. Please install it first."; exit 1; }
	@if [ -z "$(v)" ]; then \
		echo "Error: version argument 'v' is required."; \
		echo "Usage: make tag v=v1.0.0 or make tag v1.0.0"; \
		echo ""; \
		echo "Current version: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'none')"; \
		exit 1; \
	fi
	@tag_name="$(v)"; \
	case "$$tag_name" in \
		v*) ;; \
		*) \
			tag_name="v$$tag_name"; \
			echo "Warning: Version '$(v)' did not start with 'v'. Using '$$tag_name' instead."; \
			echo "Reason: Go modules require semantic version tags to start with 'v'."; \
			;; \
	esac; \
	if ! echo "$$tag_name" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$$'; then \
		echo "Error: '$$tag_name' is not a valid semantic version (expected format: vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD])."; \
		exit 1; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Git working tree has uncommitted or unstaged changes."; \
		echo "Please commit or stash your changes before tagging a release."; \
		exit 1; \
	fi; \
	if git rev-parse "$$tag_name" >/dev/null 2>&1; then \
		echo "Error: Tag '$$tag_name' already exists locally."; \
		exit 1; \
	fi; \
	if git ls-remote --tags origin "refs/tags/$$tag_name" 2>/dev/null | grep -q "refs/tags/$$tag_name"; then \
		echo "Error: Tag '$$tag_name' already exists on remote 'origin'."; \
		exit 1; \
	fi; \
	current_version="$$(git describe --tags --abbrev=0 2>/dev/null || true)"; \
	if [ -n "$$current_version" ]; then \
		if [ "$$tag_name" = "$$current_version" ]; then \
			echo "Error: Version '$$tag_name' is identical to the current version ($$current_version)."; \
			exit 1; \
		fi; \
		highest_version="$$(printf "%s\n%s\n" "$$current_version" "$$tag_name" | sort -V | tail -n 1)"; \
		if [ "$$highest_version" != "$$tag_name" ]; then \
			echo "Error: Version '$$tag_name' must be greater than current version '$$current_version'."; \
			exit 1; \
		fi; \
	fi; \
	echo "Creating tag '$$tag_name'..."; \
	git tag -a $$tag_name -m "Release $$tag_name" || exit 1; \
	echo "Pushing tag '$$tag_name' to origin..."; \
	if ! git push origin $$tag_name; then \
		echo "Push failed. Rolling back and removing local tag '$$tag_name'..."; \
		git tag -d $$tag_name; \
		exit 1; \
	fi; \
	echo "Successfully created and pushed tag '$$tag_name'."

help: ## Show this help message
	@echo 'Usage: make [target] ...'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
