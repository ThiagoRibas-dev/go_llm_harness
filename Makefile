# GoHarness build & lint tasks.
# The app itself is pure Go (zero runtime deps); these targets wire up the
# HTML/JS lint and cross-compilation used by CI.

BIN_DIR     := bin
GO          := go
GOFLAGS     := -ldflags="-s -w"
GOFLAGS_DBG :=
NODE        ?= node

.PHONY: all help lint lint-go lint-html test build build-dbg clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

all: lint test build ## Run all checks and build production binaries

##@ Quality

lint: lint-go lint-html ## Run all linters

lint-go: ## go vet
	$(GO) vet ./...

lint-html: ## Lint the embedded web console (HTML tag balance, IDs, handlers, JS syntax)
	$(NODE) scripts/lint-html.js

test: ## Run Go tests
	$(GO) test ./...

##@ Build

build: ## Cross-compile stripped release binaries to bin/
	GOOS=linux   GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/agent_linux   ./src
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/agent.exe     ./src
	GOOS=darwin  GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/agent_mac_amd ./src
	GOOS=darwin  GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/agent_mac_arm ./src

build-dbg: ## Build an unstripped binary for local debugging
	$(GO) build $(GOFLAGS_DBG) -o $(BIN_DIR)/agent ./src

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
