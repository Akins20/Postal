# Postal — developer Makefile.
# `make help` lists targets. Most targets load variables from .env if present.

SHELL := /bin/bash

# Load .env (local dev) if it exists, exporting all vars to recipe shells.
ifneq (,$(wildcard ./.env))
include .env
export
endif

# Pinned tool versions, invoked via `go run` so contributors need no extra installs.
GOOSE      := go run github.com/pressly/goose/v3/cmd/goose@v3.22.1
SQLC       := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
GOLANGCILT := go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2

MIGRATIONS_DIR := db/migrations
DB_URL         := $(POSTAL_DATABASE_URL)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the postal binary into ./bin.
	@mkdir -p bin
	go build -o bin/postal ./cmd/postal

.PHONY: run
run: ## Run the API server (deps must be up: make up).
	go run ./cmd/postal serve

.PHONY: run-worker
run-worker: ## Run the worker role.
	go run ./cmd/postal worker

.PHONY: test
test: ## Run all tests with the race detector.
	go test -race ./...

.PHONY: fmt
fmt: ## Format code (gofmt + goimports).
	gofmt -w .
	go run golang.org/x/tools/cmd/goimports@latest -w -local github.com/Akins20/postal .

.PHONY: lint
lint: ## Run golangci-lint.
	$(GOLANGCILT) run ./...

.PHONY: check
check: ## Full Definition-of-Done check (fmt, vet, lint, file-length, race tests).
	./scripts/dev/check.sh

.PHONY: sqlc
sqlc: ## Generate type-safe Go from SQL (sqlc).
	$(SQLC) generate

.PHONY: tidy
tidy: ## Tidy and verify go modules.
	go mod tidy

.PHONY: up
up: ## Start local dependencies (Postgres, Redis, MinIO).
	docker compose up -d

.PHONY: down
down: ## Stop local dependencies.
	docker compose down

.PHONY: migrate
migrate: ## Apply all up migrations.
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration.
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

.PHONY: migrate-status
migrate-status: ## Show migration status.
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status
