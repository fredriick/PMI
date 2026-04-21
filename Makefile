.PHONY: help build test test-cli run clean

PROJECT_NAME := proxymesh
BUILD_DIR := bin
GO := go

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: build-gateway build-cli build-loadtest ## Build all binaries

build-gateway: ## Build the gateway binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/gateway ./cmd/gateway

build-cli: ## Build the CLI binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/proxymesh-cli ./cmd/cli

build-loadtest: ## Build the load testing binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/loadtest ./cmd/loadtest

test: ## Run all tests
	$(GO) test ./... -count=1

test-gateway: ## Run gateway tests
	$(GO) test ./gateway/... -v -count=1

test-matchmaker: ## Run matchmaker tests
	$(GO) test ./matchmaker/... -v -count=1

test-integration: ## Run integration tests (requires Redis)
	$(GO) test ./gateway/... -v -count=1 -run Integration

test-coverage: ## Generate coverage report
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out

benchmark: ## Run benchmarks
	$(GO) test ./... -bench=. -benchmem -run=^$ -benchtime=10s

run: build-gateway ## Run the gateway
	$(BUILD_DIR)/gateway

run-cli: build-cli ## Run the CLI tool
	$(BUILD_DIR)/proxymesh-cli --help

run-loadtest: build-loadtest ## Run load test (requires gateway running)
	$(BUILD_DIR)/loadtest -c 50 -d 30s

lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format Go code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

proto: ## Generate protobuf code (requires protoc)
	protoc --go_out=. --go_opt=module=proxymesh --go-grpc_out=. --go-grpc_opt=module=proxymesh internal/grpc/peer.proto

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean

demo: ## Quick demo setup (requires Redis)
	docker-compose up -d

bench: build-loadtest ## Run loadtest benchmark
	$(BUILD_DIR)/loadtest -c 20 -d 10s

watch: build-loadtest ## Run loadtest with 200 workers for 60s
	$(BUILD_DIR)/loadtest -c 200 -d 60s
