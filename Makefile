# Makefile for radius-accounting-server
# This is the standard approach for Go projects

# export env
ifneq (,$(wildcard .env))
  include .env
  export
endif

# Variables
BINARY_NAME=radius-accounting-server
DOCKER_COMPOSE=docker compose
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build variables
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

.PHONY: help
help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2 } /^##@/ { printf "\n${YELLOW}%s${NC}\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: run
run: ## Run the application locally (requires .env file)
	@if [ ! -f .env ]; then \
		echo "${RED}Error: .env file not found${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Starting application...${NC}"
	$(DOCKER_COMPOSE) up

.PHONY: run-detached
run-detached: ## Run the application in background
	@if [ ! -f .env ]; then \
		echo "${RED}Error: .env file not found${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Starting application in background...${NC}"
	$(DOCKER_COMPOSE) up -d

.PHONY: stop
stop: ## Stop all running services
	@echo "${YELLOW}Stopping services...${NC}"
	$(DOCKER_COMPOSE) down
	docker rm -f radclient-test 2>/dev/null || true

.PHONY: logs
logs: ## Show logs from all services
	$(DOCKER_COMPOSE) logs -f

.PHONY: shell
shell: ## Start interactive radclient shell
	@echo "${GREEN}Starting interactive shell...${NC}"
	$(DOCKER_COMPOSE) up -d redis controlplane logger
	@sleep 2
	@docker rm -f radclient-test 2>/dev/null || true
	@docker run -it --rm \
		--name radclient-test \
		--network radius-network \
		-e RADIUS_SHARED_SECRET="$${RADIUS_SHARED_SECRET}" \
		-e RADIUS_SERVER=radius-controlplane \
		-v ./examples:/requests \
		--entrypoint /bin/bash \
		radclient-test

##@ Testing

.PHONY: test
test: ## Run all tests with coverage
	@echo "${GREEN}Running all tests with coverage...${NC}"
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@$(MAKE) coverage-report

.PHONY: test-unit
test-unit: ## Run unit tests with coverage
	@echo "${GREEN}Running unit tests...${NC}"
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) \
		$$(go list ./internal/...)
	@echo "${GREEN}✓ Unit tests passed${NC}"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "${GREEN}Running integration tests...${NC}"
	@echo "Starting test dependencies..."
	$(DOCKER_COMPOSE) -f docker-compose.integration.yml up -d redis
	@sleep 2
	$(DOCKER_COMPOSE) -f docker-compose.integration.yml up --build --exit-code-from tests tests
	$(DOCKER_COMPOSE) -f docker-compose.integration.yml down
	@echo "${GREEN}✓ Integration tests passed${NC}"

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "${YELLOW}Running tests with race detection...${NC}"
	$(GOTEST) -race ./...

.PHONY: test-short
test-short: ## Run only short tests
	$(GOTEST) -short ./...

.PHONY: coverage
coverage: ## Generate coverage report
	@echo "${GREEN}Generating coverage report...${NC}"
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $$(go list ./... | grep -v /cmd/)
	@$(MAKE) coverage-report

.PHONY: coverage-report
coverage-report: ## Display coverage report
	@if [ -f $(COVERAGE_FILE) ]; then \
		echo "${GREEN}=== Coverage Report ===${NC}"; \
		$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total Coverage: " $$3}'; \
	else \
		echo "${RED}No coverage file found. Run 'make coverage' first.${NC}"; \
	fi

.PHONY: coverage-html
coverage-html: ## Generate HTML coverage report
	@if [ -f $(COVERAGE_FILE) ]; then \
		$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML); \
		echo "${GREEN}Coverage report generated: $(COVERAGE_HTML)${NC}"; \
	else \
		echo "${RED}No coverage file found. Run 'make coverage' first.${NC}"; \
	fi

##@ Build

.PHONY: build
build: ## Build the application binaries
	@echo "${GREEN}Building binaries...${NC}"
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o bin/radius-controlplane ./cmd/radius-controlplane
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o bin/radius-logger ./cmd/radius-controlplane-logger
	@echo "${GREEN}✓ Build complete${NC}"

.PHONY: build-docker
build-docker: ## Build Docker images
	@echo "${GREEN}Building Docker images...${NC}"
	docker build -t radius-controlplane:$(VERSION) -f cmd/radius-controlplane/Dockerfile .
	docker build -t radius-logger:$(VERSION) -f cmd/radius-controlplane-logger/Dockerfile .
	@echo "${GREEN}✓ Docker build complete${NC}"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	$(GOCLEAN)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	rm -rf bin/
	@echo "${GREEN}✓ Clean complete${NC}"

##@ Code Quality

.PHONY: lint
lint: ## Run linters
	@if [ -x "$(HOME)/go/bin/golangci-lint" ]; then \
		PATH="$(HOME)/go/bin:$$PATH" golangci-lint run; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "${YELLOW}golangci-lint not installed. Install with:${NC}"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

.PHONY: fmt
fmt: ## Format code
	@echo "${GREEN}Formatting code...${NC}"
	$(GOCMD) fmt ./...
	@echo "${GREEN}✓ Formatting complete${NC}"

.PHONY: vet
vet: ## Run go vet
	@echo "${GREEN}Running go vet...${NC}"
	$(GOCMD) vet ./...
	@echo "${GREEN}✓ Vet complete${NC}"

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	@echo "${GREEN}Tidying modules...${NC}"
	$(GOMOD) tidy
	@echo "${GREEN}✓ Modules tidied${NC}"

.PHONY: mod-download
mod-download: ## Download go modules
	@echo "${GREEN}Downloading modules...${NC}"
	$(GOMOD) download
	@echo "${GREEN}✓ Modules downloaded${NC}"

##@ CI/CD

.PHONY: ci
ci: ## Run CI pipeline (lint, test, build)
	@echo "${GREEN}Running CI pipeline...${NC}"
	@$(MAKE) lint
	@$(MAKE) vet
	@$(MAKE) test
	@$(MAKE) build
	@echo "${GREEN}✓ CI pipeline complete${NC}"

.PHONY: pre-commit
pre-commit: ## Run pre-commit checks
	@echo "${GREEN}Running pre-commit checks...${NC}"
	@$(MAKE) fmt
	@$(MAKE) vet
	@$(MAKE) test-unit
	@echo "${GREEN}✓ Pre-commit checks passed${NC}"

##@ Utilities

.PHONY: deps
deps: ## Check dependencies
	@echo "${GREEN}Checking dependencies...${NC}"
	@$(GOCMD) list -m all

.PHONY: version
version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"


# Default target
.DEFAULT_GOAL := help