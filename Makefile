#!/usr/bin/make -f

BINARY := oasyced
BUILD_DIR := ./build
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "v0.1.0")
COMMIT := $(shell git log -1 --format='%H' 2>/dev/null || echo "unknown")
LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=oasyce \
           -X github.com/cosmos/cosmos-sdk/version.AppName=$(BINARY) \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
           -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

.PHONY: all build install test lint clean proto-gen docker-build docker-testnet

all: build

build:
	@echo "Building $(BINARY)..."
	@CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) ./cmd/oasyced

build-linux:
	@echo "Cross-compiling $(BINARY) for linux/amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '$(LDFLAGS) -s -w' -o $(BUILD_DIR)/$(BINARY)-linux ./cmd/oasyced
	@echo "Built $(BUILD_DIR)/$(BINARY)-linux ($$(du -h $(BUILD_DIR)/$(BINARY)-linux | cut -f1))"

install:
	@echo "Installing $(BINARY)..."
	@CGO_ENABLED=0 go install -ldflags '$(LDFLAGS)' ./cmd/oasyced

test:
	@echo "Running tests..."
	@CGO_ENABLED=0 go test ./... -v

lint:
	@echo "Running linter..."
	@golangci-lint run --timeout 5m

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

proto-gen:
	@echo "Generating protobuf files..."
	@scripts/protocgen.sh

docker-build:
	@echo "Building Docker image..."
	@docker build -t oasyce/chain:latest .

docker-testnet:
	@echo "Starting local testnet with docker-compose..."
	@docker-compose up -d

tidy:
	@echo "Tidying go modules..."
	@go mod tidy
