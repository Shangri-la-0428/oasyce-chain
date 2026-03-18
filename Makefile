#!/usr/bin/make -f

BINARY := oasyced
BUILD_DIR := ./build
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "v0.1.0")
COMMIT := $(shell git log -1 --format='%H' 2>/dev/null || echo "unknown")
LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=oasyce \
           -X github.com/cosmos/cosmos-sdk/version.AppName=$(BINARY) \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
           -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

.PHONY: all build install test lint clean proto-gen

all: build

build:
	@echo "Building $(BINARY)..."
	@go build -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) ./cmd/oasyced

install:
	@echo "Installing $(BINARY)..."
	@go install -ldflags '$(LDFLAGS)' ./cmd/oasyced

test:
	@echo "Running tests..."
	@go test ./... -v

lint:
	@echo "Running linter..."
	@golangci-lint run --timeout 5m

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

proto-gen:
	@echo "Generating protobuf files..."
	@scripts/protocgen.sh

tidy:
	@echo "Tidying go modules..."
	@go mod tidy
