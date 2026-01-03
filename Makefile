.PHONY: build test run clean docker-build docker-run help

# Binary name
BINARY_NAME=gameday-sim

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Docker parameters
DOCKER_IMAGE=gameday-sim
DOCKER_TAG=latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the application
	$(GOBUILD) -o $(BINARY_NAME) -v

run: build ## Build and run the application
	./$(BINARY_NAME)

test: ## Run tests
	$(GOTEST) -v ./tests/...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./tests/...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Remove binary and build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f simulation_results.json

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

lint: fmt vet ## Run formatters and linters

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run --rm -v $(PWD)/config.yaml:/root/config.yaml $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-run-custom: ## Run Docker container with custom config (use CONFIG=path/to/config.yaml)
	docker run --rm -v $(CONFIG):/root/config.yaml $(DOCKER_IMAGE):$(DOCKER_TAG)

all: clean deps lint test build ## Run all tasks (clean, deps, lint, test, build)

.DEFAULT_GOAL := help
