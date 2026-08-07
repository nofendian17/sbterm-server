.PHONY: help run build test test-race vet mock tidy

APP_NAME := sbterm-server

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

mock: ## Generate mocks with uber-go/mock (go generate)
	go generate ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy
