.PHONY: help run-api run-ws run-ingest run-stream build test test-race vet fmt fmt-check install-hooks mock tidy

MODULES := apps/api apps/ws apps/ingest apps/stream apps/core libs/pkg libs/marketdata libs/proto libs/stockbit

GO_FILES := $(shell find apps libs -name '*.go')

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-14s %s\n", $$1, $$2}'

run-api: ## Run the REST api
	go run ./apps/api/cmd/server

run-ws: ## Run the datafeed websocket publisher
	go run ./apps/ws/cmd/ws

run-ingest: ## Run the questdb ingester
	go run ./apps/ingest/cmd/ingest

run-stream: ## Run the websocket fan-out stream service
	go run ./apps/stream/cmd/stream

run-core: ## Run the core auth service
	go run ./apps/core/cmd/server

build: ## Build every binary into bin/
	mkdir -p bin
	go build -o bin/sbterm-api ./apps/api/cmd/server
	go build -o bin/sbterm-ws ./apps/ws/cmd/ws
	go build -o bin/sbterm-ingest ./apps/ingest/cmd/ingest
	go build -o bin/sbterm-stream ./apps/stream/cmd/stream
	go build -o bin/sbterm-core ./apps/core/cmd/server

test: ## Run all tests in the workspace
	@for d in $(MODULES); do \
		echo "==> $$d"; \
		(cd $$d && go test ./...) || exit 1; \
	done

test-race: ## Run all tests with the race detector
	@for d in $(MODULES); do \
		echo "==> $$d"; \
		(cd $$d && go test -race ./...) || exit 1; \
	done

vet: ## Run go vet on the workspace
	@for d in $(MODULES); do \
		echo "==> $$d"; \
		(cd $$d && go vet ./...) || exit 1; \
	done

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
	cd apps/api && go generate ./...

tidy: ## Tidy every module's go.mod and go.sum
	go work sync
	@for d in $(MODULES); do \
		echo "==> $$d"; \
		(cd $$d && go mod tidy) || exit 1; \
	done
