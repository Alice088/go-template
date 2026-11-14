SHELL := /bin/bash

# Load .env so local targets share the same configuration as the application.
# The file is optional; `make build`, `make test` and `make lint` work without it.
-include .env
export

BINARY  ?= bin/api
MAIN    ?= ./cmd/api
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)
MIGRATIONS_DIR ?= migrations

SWAG_VERSION    ?= v1.16.6
MIGRATE_VERSION ?= v4.19.0

.DEFAULT_GOAL := help

.PHONY: help run build test test-cover coverage lint fmt vet tidy check swagger migrate-up migrate-down migrate-version docker-build docker-up docker-down

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## Run the API locally
	go run $(MAIN)

build: ## Build the API binary into bin/
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

test: ## Run all tests with the race detector
	go test -race -count=1 ./...

test-cover: ## Run tests and write bin/coverage.out
	go test -race -count=1 -coverprofile=bin/coverage.out ./...
	go tool cover -func=bin/coverage.out | tail -1

coverage: test-cover ## Open the coverage report in a browser
	go tool cover -html=bin/coverage.out

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format code (gofumpt + goimports via golangci-lint)
	golangci-lint fmt

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

check: fmt vet lint test ## Run the full local check suite

swagger: ## Regenerate the OpenAPI docs into docs/
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g $(MAIN) -o docs --parseInternal

migrate-up: ## Apply all pending migrations
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down: ## Roll back the last migration
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-version: ## Print the current migration version
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

docker-build: ## Build the docker image
	docker build --build-arg VERSION=$(VERSION) -t go-template-backend:local .

docker-up: ## Start the stack in the background
	docker compose up --build -d

docker-down: ## Stop the stack and keep the database volume
	docker compose down
