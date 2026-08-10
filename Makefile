.PHONY: help run build test test-race vet fmt fmt-check install-hooks mock tidy

APP_NAME := sbterm-server

GO_FILES := $(shell find . -name '*.go' -not -path './.git/*')

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-14s %s\n", $$1, $$2}'

run: ## Run the server
	go run ./cmd/server

build: ## Build the binary into bin/
	go build -o bin/$(APP_NAME) ./cmd/server

test: ## Run all tests
	go test ./...

test-race: ## Run all tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go source files with gofmt
	gofmt -w $(GO_FILES)

fmt-check: ## Fail if any Go file is not gofmt-formatted
	@unformatted=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt: the following files need formatting (run 'make fmt'):"; \
		echo "$$unformatted"; \
		exit 1; \
	fi; \
	echo "gofmt: all files formatted"

install-hooks: ## Install git hooks (core.hooksPath -> .githooks)
	chmod +x .githooks/pre-commit
	git config core.hooksPath .githooks
	@echo "git hooks installed (core.hooksPath = $$(git config core.hooksPath))"

mock: ## Generate mocks with uber-go/mock (go generate)
	go generate ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy
