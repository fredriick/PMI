.PHONY: help backup restore list-backups test lint build docker-build docker-run clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

backup: ## Backup Redis data
	@echo "Running Redis backup..."
	@bash scripts/redis-backup.sh

restore: ## Restore Redis from latest backup
	@echo "Finding latest backup..."
	@LATEST=$$(ls -1t backups/redis_backup_*.rdb.gz 2>/dev/null | head -1); \
	if [ -z "$$LATEST" ]; then echo "No backups found"; exit 1; fi; \
	echo "Restoring from $$LATEST..."; \
	RESTORE_FILE=$$(basename "$$LATEST") bash scripts/redis-restore.sh

restore-file: ## Restore specific backup (FILE=filename)
	@if [ -z "$(FILE)" ]; then echo "Usage: make restore-file FILE=redis_backup_20240101_120000.rdb.gz"; exit 1; fi; \
	RESTORE_FILE=$(FILE) bash scripts/redis-restore.sh

list-backups: ## List available backups
	@ls -lah backups/ 2>/dev/null || echo "No backups directory found"

test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

lint: ## Run linter
	go vet ./...
	gofmt -d .

build: ## Build binary
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/proxy-mesh .

docker-build: ## Build Docker image
	docker build -t proxymesh:latest .

docker-run: ## Run with Docker Compose
	docker compose up -d redis gateway

docker-run-monitoring: ## Run with monitoring stack
	docker compose --profile monitoring up -d

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f *.test
	rm -f **/*.test
