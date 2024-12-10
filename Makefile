# Build System Configuration
.DEFAULT_GOAL := help

# OS specific commands and configurations
PASTE_BUFFER := $(HOME)/.makefile_buffer
PBCOPY := $(shell command -v pbcopy 2> /dev/null)
ifeq ($(PBCOPY),)
    # If pbcopy is not available, create a no-op command
    COPY_TO_CLIPBOARD = tee $(PASTE_BUFFER)
else
    # If pbcopy is available, use it with the buffer file
    COPY_TO_CLIPBOARD = tee $(PASTE_BUFFER) && cat $(PASTE_BUFFER) | pbcopy
endif

# Project configuration
PROJECT_NAME=project-name
SERVER_NAME=wsignd
CLIENT_NAME=wsignctl
VERSION?=0.0.1
REGISTRY?=registry.example.com

# Container configuration
# Check if podman is available, otherwise use docker
CONTAINER_ENGINE := $(shell command -v podman 2> /dev/null || echo docker)
COMPOSE_ENGINE := $(shell command -v podman-compose 2> /dev/null || echo docker-compose)

# Image names using variables
SERVER_IMAGE=$(REGISTRY)/$(SERVER_NAME):$(VERSION)
CLIENT_IMAGE=$(REGISTRY)/$(CLIENT_NAME):$(VERSION)
SERVER_IMAGE_LATEST=$(REGISTRY)/$(SERVER_NAME):latest
CLIENT_IMAGE_LATEST=$(REGISTRY)/$(CLIENT_NAME):latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOVET=$(GOCMD) vet

# Binary configuration
BINARY_OUTPUT_DIR=bin
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Binary paths
SERVER_PATH=$(BINARY_OUTPUT_DIR)/$(SERVER_NAME)
CLIENT_PATH=$(BINARY_OUTPUT_DIR)/$(CLIENT_NAME)

# Tools
GOLINT=golangci-lint
GOSEC=gosec

# Test parameters
TEST_OUTPUT_DIR=test-output
COVERAGE_FILE=coverage.out

# Docker/Podman context and compose files
BUILD_CONTEXT=.
COMPOSE_FILE=docker-compose.yml
COMPOSE_DEV_FILE=docker-compose.dev.yml

.PHONY: all clean test coverage lint sec-check vet fmt help install-tools run dev
.PHONY: build build-server build-client run-server run-client
.PHONY: docker-build docker-push docker-run docker-stop compose-up compose-down
.PHONY: build-images push-images x y

help: ## Display available commands
	@echo "Available Commands:"
	@echo
	@awk 'BEGIN {FS = ":.*##"; printf "  \033[36m%-20s\033[0m %s\n", "target", "description"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

$(BINARY_OUTPUT_DIR):
	mkdir -p $(BINARY_OUTPUT_DIR)

$(TEST_OUTPUT_DIR):
	mkdir -p $(TEST_OUTPUT_DIR)

clean: ## Clean build artifacts and containers
	$(GOCLEAN)
	rm -rf $(BINARY_OUTPUT_DIR)
	rm -rf $(TEST_OUTPUT_DIR)
	rm -f $(COVERAGE_FILE)
	$(COMPOSE_ENGINE) -f $(COMPOSE_FILE) down --volumes --remove-orphans

fmt: ## Format code
	@echo "==> Formatting code"
	@go fmt ./...

vet: fmt ## Run static analysis
	@echo "==> Running static analysis"
	$(GOVET) ./...

lint: vet ## Run linter
	@echo "==> Running linter"
	$(GOLINT) run

sec-check: ## Run security scan
	@echo "==> Running security scan"
	$(GOSEC) ./...

test: ## Run tests
	@echo "==> Running tests"
	$(GOTEST) -v -race ./...

coverage: ## Generate coverage report
	@echo "==> Generating coverage report"
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE)

build-server: $(BINARY_OUTPUT_DIR) ## Build server binary
	@echo "==> Building server"
	$(GOBUILD) $(LDFLAGS) -o $(SERVER_PATH) ./cmd/server

build-client: $(BINARY_OUTPUT_DIR) ## Build client binary
	@echo "==> Building client"
	$(GOBUILD) $(LDFLAGS) -o $(CLIENT_PATH) ./cmd/client

build: build-server build-client ## Build all binaries

install-tools: ## Install development tools
	@echo "==> Installing development tools"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

all: fmt vet lint sec-check test build ## Run full verification and build

# Development targets
run-server: build-server ## Run server
	@echo "==> Running server"
	$(SERVER_PATH)

run-client: build-client ## Run client
	@echo "==> Running client"
	$(CLIENT_PATH)

run: run-server ## Run server (default)

dev: ## Run with hot reload
	@echo "==> Starting development server"
	air -c .air.toml

# Container targets
build-images: ## Build container images
	@echo "==> Building container images with $(CONTAINER_ENGINE)"
	$(CONTAINER_ENGINE) build -t $(SERVER_IMAGE) -t $(SERVER_IMAGE_LATEST) -f build/server/Dockerfile $(BUILD_CONTEXT)
	$(CONTAINER_ENGINE) build -t $(CLIENT_IMAGE) -t $(CLIENT_IMAGE_LATEST) -f build/client/Dockerfile $(BUILD_CONTEXT)

push-images: ## Push container images to registry
	@echo "==> Pushing images to registry"
	$(CONTAINER_ENGINE) push $(SERVER_IMAGE)
	$(CONTAINER_ENGINE) push $(CLIENT_IMAGE)
	$(CONTAINER_ENGINE) push $(SERVER_IMAGE_LATEST)
	$(CONTAINER_ENGINE) push $(CLIENT_IMAGE_LATEST)

compose-up: ## Start compose environment
	@echo "==> Starting compose environment with $(COMPOSE_ENGINE)"
	$(COMPOSE_ENGINE) -f $(COMPOSE_FILE) up -d

compose-down: ## Stop compose environment
	@echo "==> Stopping compose environment"
	$(COMPOSE_ENGINE) -f $(COMPOSE_FILE) down --volumes --remove-orphans

compose-dev: ## Start development compose environment
	@echo "==> Starting development environment"
	$(COMPOSE_ENGINE) -f $(COMPOSE_FILE) -f $(COMPOSE_DEV_FILE) up -d

compose-logs: ## View compose logs
	@echo "==> Viewing compose logs"
	$(COMPOSE_ENGINE) -f $(COMPOSE_FILE) logs -f

compose-ps: ## List compose containers
	@echo "==> Listing compose containers"
	$(COMPOSE_ENGINE) -f $(COMPOSE_FILE) ps

# Quick productivity commands
x: ## Copy project tree structure to clipboard (ignoring git files)
	@echo "==> Copying tree structure to clipboard"
	@tree --gitignore | $(COPY_TO_CLIPBOARD)

y: ## Run all checks and copy output to clipboard while displaying
	@echo "==> Running all checks and copying output..."
	@{ make all 2>&1; } | $(COPY_TO_CLIPBOARD)

# Single container targets
docker-run-server: ## Run server container
	@echo "==> Running server container"
	$(CONTAINER_ENGINE) run -d --name $(SERVER_NAME) $(SERVER_IMAGE)

docker-run-client: ## Run client container
	@echo "==> Running client container"
	$(CONTAINER_ENGINE) run -it --rm $(CLIENT_IMAGE)

docker-stop: ## Stop all project containers
	@echo "==> Stopping containers"
	$(CONTAINER_ENGINE) stop $(SERVER_NAME) || true
	$(CONTAINER_ENGINE) rm $(SERVER_NAME) || true
