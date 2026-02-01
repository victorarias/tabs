# Makefile for tabs

.PHONY: all build build-ui build-cli build-daemon build-server build-local test test-unit test-integration test-golden test-golden-update install clean dev dev-ui dev-api

# Version
VERSION ?= 0.1.0-dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Directories
BIN_DIR := bin
BUILD_DIR := build
PREFIX ?= $(HOME)/.local

all: build

build: build-cli build-daemon build-server build-local

build-ui:
	@echo "Building UI..."
	cd ui && pnpm install && pnpm run build

build-cli: build-ui
	@echo "Building tabs-cli..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-cli ./cmd/tabs-cli

dev-ui:
	cd ui && pnpm run dev

dev-api:
	go run ./cmd/tabs-cli ui

dev:
	@echo "Starting Go API on :3787 and Vite on :3000..."
	@echo "Open http://localhost:3000 for hot-reload dev"
	@trap 'kill 0' EXIT; \
		go run ./cmd/tabs-cli ui & \
		cd ui && pnpm run dev

build-daemon:
	@echo "Building tabs-daemon..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-daemon ./cmd/tabs-daemon

build-server:
	@echo "Building tabs-server..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-server ./cmd/tabs-server

build-local:
	@echo "Building tabs-local..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-local ./cmd/tabs-local

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage:
	@echo "Generating coverage report..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit:
	@echo "Running unit tests..."
	go test -v -race ./internal/daemon/... -run 'Test[^Integration]'

test-integration:
	@echo "Running integration tests..."
	go test -v -race ./internal/daemon/... -run 'TestIntegration'

test-golden:
	@echo "Running golden file tests..."
	go test -v ./internal/daemon/... -run 'Golden'

test-golden-update:
	@echo "Updating golden files..."
	go test -v ./internal/daemon/... -run 'Golden' -update

install: build
	@echo "Installing binaries to $(PREFIX)/bin..."
	@mkdir -p $(PREFIX)/bin
	install -m 755 $(BIN_DIR)/tabs-cli $(PREFIX)/bin/tabs-cli
	install -m 755 $(BIN_DIR)/tabs-daemon $(PREFIX)/bin/tabs-daemon

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Development helpers
dev-daemon:
	go run ./cmd/tabs-daemon

dev-cli:
	go run ./cmd/tabs-cli

# Docker
docker-build:
	docker build -t tabs-server:$(VERSION) -f Dockerfile .

docker-push:
	docker tag tabs-server:$(VERSION) yourorg/tabs-server:$(VERSION)
	docker push yourorg/tabs-server:$(VERSION)

# Linting
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Tidy dependencies
tidy:
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build              - Build all binaries"
	@echo "  build-cli          - Build tabs-cli only"
	@echo "  build-daemon       - Build tabs-daemon only"
	@echo "  build-server       - Build tabs-server only"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only"
	@echo "  test-golden        - Run golden file tests"
	@echo "  test-golden-update - Update golden files"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  install            - Install binaries to PREFIX/bin (default: ~/.local/bin)"
	@echo "  clean              - Remove build artifacts"
	@echo "  dev-daemon         - Run daemon in development mode"
	@echo "  dev-cli            - Run CLI in development mode"
	@echo "  docker-build       - Build Docker image"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  tidy               - Tidy dependencies"
