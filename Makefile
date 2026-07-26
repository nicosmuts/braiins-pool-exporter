BINARY := braiins-pool-exporter
CMD := ./cmd/$(BINARY)
BUILD_DIR := bin

.DEFAULT_GOAL := help

.PHONY: help fmt fmt-check tidy vet test test-race lint build run check clean

help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format Go source.
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

fmt-check: ## Check Go formatting without changing files.
	@test -z "$$(gofmt -l .)"

tidy: ## Synchronize module dependencies.
	go mod tidy

vet: ## Run Go static analysis.
	go vet ./...

test: ## Run unit tests.
	go test ./...

test-race: ## Run unit tests with the race detector.
	go test -race ./...

lint: ## Run golangci-lint (must be installed separately).
	golangci-lint run

build: ## Build the exporter.
	go build -trimpath -o $(BUILD_DIR)/$(BINARY)$(if $(filter Windows_NT,$(OS)),.exe,) $(CMD)

run: ## Run the exporter locally.
	go run $(CMD)

check: fmt-check vet test test-race build ## Run all checks available in the Go toolchain.
	git diff --check

clean: ## Remove local build and test outputs.
	go clean -testcache
	$(RM) -r $(BUILD_DIR)
