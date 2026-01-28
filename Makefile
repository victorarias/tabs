# Makefile for tabs

.PHONY: all build build-cli build-daemon build-server test install clean

# Version
VERSION ?= 0.1.0-dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Directories
BIN_DIR := bin
BUILD_DIR := build

all: build

build: build-cli build-daemon build-server

build-cli:
	@echo "Building tabs-cli..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-cli ./cmd/tabs-cli

build-daemon:
	@echo "Building tabs-daemon..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-daemon ./cmd/tabs-daemon

build-server:
	@echo "Building tabs-server..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/tabs-server ./cmd/tabs-server

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage:
	@echo "Generating coverage report..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

install: build
	@echo "Installing binaries..."
	cp $(BIN_DIR)/tabs-cli /usr/local/bin/tabs-cli
	cp $(BIN_DIR)/tabs-daemon /usr/local/bin/tabs-daemon

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
	@echo "  build          - Build all binaries"
	@echo "  build-cli      - Build tabs-cli only"
	@echo "  build-daemon   - Build tabs-daemon only"
	@echo "  build-server   - Build tabs-server only"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  install        - Install binaries to /usr/local/bin"
	@echo "  clean          - Remove build artifacts"
	@echo "  dev-daemon     - Run daemon in development mode"
	@echo "  dev-cli        - Run CLI in development mode"
	@echo "  docker-build   - Build Docker image"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy dependencies"
