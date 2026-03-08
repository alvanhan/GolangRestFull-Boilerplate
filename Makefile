# =============================================================================
# File Management Service — Makefile
# =============================================================================
-include .env
export

# -----------------------------------------------------------------------------
# Variables
# -----------------------------------------------------------------------------
BINARY_NAME    := api
BUILD_DIR      := bin
CMD_PATH       := ./cmd/api/main.go
MIGRATIONS_DIR := migrations
DOCKER_COMPOSE := docker compose

DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

# Build metadata
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -w -s -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'

.PHONY: all help run build test test-unit migrate-up migrate-down seed \
        docker-up docker-down docker-logs lint tidy swag mock clean gen-secret

all: build

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------
help: ## Show all available targets
	@echo ""
	@echo "  File Management Service"
	@echo "  ========================"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# -----------------------------------------------------------------------------
# Development
# -----------------------------------------------------------------------------
run: ## Run the application with hot-reload support
	go run $(CMD_PATH)

build: ## Build binary to bin/api
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# -----------------------------------------------------------------------------
# Testing
# -----------------------------------------------------------------------------
test: ## Run all tests with race detector and coverage report
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | tail -n 1

test-unit: ## Run unit tests only (skip integration tests)
	go test -v -race -short ./...

# -----------------------------------------------------------------------------
# Database Migrations
# -----------------------------------------------------------------------------
migrate-up: ## Run all pending migrations
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down: ## Rollback the last applied migration
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

seed: ## Run seed data (002_seed.sql) into the running postgres container
	docker exec -i file-management-postgres psql -U $(DB_USER) -d $(DB_NAME) < $(MIGRATIONS_DIR)/002_seed.sql
	@echo "Seed data applied successfully"

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------
docker-up: ## Start all services in detached mode
	$(DOCKER_COMPOSE) up -d
	@echo "✓ Services started. API available at http://localhost:$${APP_PORT:-8080}"

docker-down: ## Stop and remove all service containers
	$(DOCKER_COMPOSE) down

docker-logs: ## Follow the api service logs
	$(DOCKER_COMPOSE) logs -f api

# -----------------------------------------------------------------------------
# Code Quality
# -----------------------------------------------------------------------------
lint: ## Run golangci-lint
	golangci-lint run --timeout 5m ./...

tidy: ## Tidy and verify go modules
	go mod tidy
	go mod verify
	@echo "✓ Modules tidied"

# -----------------------------------------------------------------------------
# Code Generation
# -----------------------------------------------------------------------------
swag: ## Generate Swagger/OpenAPI documentation
	swag init -g $(CMD_PATH) -o docs --parseDependency --parseInternal
	@echo "✓ Swagger docs generated in docs/"

mock: ## Generate mocks for all interfaces using mockery
	mockery --all --dir internal --output internal/mocks --outpkg mocks --with-expecter
	@echo "✓ Mocks generated in internal/mocks/"

# -----------------------------------------------------------------------------
# Utilities
# -----------------------------------------------------------------------------
clean: ## Remove build artifacts and coverage files
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "✓ Cleaned build artifacts"

gen-secret: ## Generate a cryptographically secure random secret key
	@openssl rand -hex 32
