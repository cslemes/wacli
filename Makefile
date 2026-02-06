.PHONY: help build build-api build-cli run-api test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: build-cli build-api ## Build both CLI and API

build-cli: ## Build the CLI binary
	go build -o bin/wacli cmd/wacli/*.go

build-api: ## Build the API server binary
	go build -o bin/wacli-api cmd/wacli-api/main.go

run-api: ## Run the API server (requires WACLI_API_KEYS env var)
	@if [ -z "$$WACLI_API_KEYS" ]; then \
		echo "Error: WACLI_API_KEYS environment variable is required"; \
		echo "Example: export WACLI_API_KEYS=your-secret-key"; \
		exit 1; \
	fi
	go run cmd/wacli-api/main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/

install: build ## Install binaries to GOPATH/bin
	cp bin/wacli $(GOPATH)/bin/
	cp bin/wacli-api $(GOPATH)/bin/
