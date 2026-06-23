SHELL := /usr/bin/env bash

APP_NAME ?= bc-sandbox
BUFFALO ?= buffalo
DOCKER_COMPOSE ?= docker compose
GO_ENV ?= development
POSTGRES_SERVICE ?= postgres
TEST_DB_NAME ?= bc_sandbox_test

ifneq (,$(wildcard .env))
include .env
export
endif

POSTGRES_USER ?= bc_sandbox
POSTGRES_PASSWORD ?= bc_sandbox
POSTGRES_DB ?= bc_sandbox

.DEFAULT_GOAL := help

.PHONY: help deps dev build format test test-actions test-models clean \
	infra-up infra-up-tools infra-down infra-restart infra-ps infra-logs \
	db-create db-create-test db-migrate db-drop db-drop-test db-reset db-seed

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install Go and JS dependencies
	go mod download
	yarn install

dev: ## Run the Buffalo development server
	GO_ENV=$(GO_ENV) $(BUFFALO) dev

build: ## Build production assets and binary
	yarn build
	GO_ENV=production $(BUFFALO) build --static -o bin/$(APP_NAME)

format: ## Format Go source files
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

test: ## Run all Go tests
	GO_ENV=test go test ./...

test-actions: ## Run action tests only
	GO_ENV=test go test ./actions

test-models: ## Run model tests only
	GO_ENV=test go test ./models

clean: ## Remove generated build artifacts
	rm -rf bin tmp public/assets coverage coverage.data

infra-up: ## Start local infrastructure
	$(DOCKER_COMPOSE) up -d postgres kafka kafka-init cassandra cassandra-init

infra-up-tools: ## Start local infrastructure plus optional UIs
	$(DOCKER_COMPOSE) --profile tools up -d

infra-down: ## Stop local infrastructure
	$(DOCKER_COMPOSE) down

infra-restart: infra-down infra-up ## Restart local infrastructure

infra-ps: ## Show infrastructure status
	$(DOCKER_COMPOSE) ps

infra-logs: ## Follow infrastructure logs
	$(DOCKER_COMPOSE) logs -f

db-create: ## Create configured Buffalo databases
	$(BUFFALO) pop create -a

db-create-test: ## Create the local test database in Docker Postgres
	$(DOCKER_COMPOSE) exec -T $(POSTGRES_SERVICE) createdb -U $(POSTGRES_USER) $(TEST_DB_NAME)

db-migrate: ## Run database migrations
	$(BUFFALO) pop migrate

db-drop: ## Drop configured Buffalo databases
	$(BUFFALO) pop drop -a

db-drop-test: ## Drop the local test database in Docker Postgres
	$(DOCKER_COMPOSE) exec -T $(POSTGRES_SERVICE) dropdb --if-exists -U $(POSTGRES_USER) $(TEST_DB_NAME)

db-reset: db-drop db-create db-migrate ## Recreate and migrate databases

db-seed: ## Run database seed task
	$(BUFFALO) task db:seed
