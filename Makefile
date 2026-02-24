# Makefile for Tecla - The Git Ecosystem Explorer

# Variables
BINARY_NAME=tecla
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
DIST_DIR=dist
PREFIX ?= $(HOME)/.local/bin

# Linker flags for versioning
LDFLAGS=-ldflags "-X 'github.com/gi4nks/tecla/cmd.Version=${VERSION}' \
                 -X 'github.com/gi4nks/tecla/cmd.GitCommit=${COMMIT}' \
                 -X 'github.com/gi4nks/tecla/cmd.BuildDate=${DATE}' \
                 -s -w"

# Build Targets
.PHONY: all build run clean tidy test install dist docker help

all: build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) main.go

run: build
	./$(BINARY_NAME)

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR) bin coverage

tidy:
	go mod tidy

test:
	go test -v ./...

# Tecla might not have integration tests in the same path yet, but we keep the target for consistency
test-integration:
	go test -v ./gitinfo/...

test-all: test test-integration

# Cross-compilation targets for distribution
dist: clean
	mkdir -p $(DIST_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Docker image build
docker:
	docker build -t tecla:$(VERSION) .
	docker tag tecla:$(VERSION) tecla:latest

# Local installation
install: build
	@mkdir -p $(PREFIX)
	@cp $(BINARY_NAME) $(PREFIX)/$(BINARY_NAME)

help:
	@echo "Makefile for Tecla"
	@echo "Usage:"
	@echo "  make build           Build the binary"
	@echo "  make run             Build and run the binary"
	@echo "  make clean           Remove binaries and build artifacts"
	@echo "  make test            Run unit tests"
	@echo "  make test-all        Run all tests"
	@echo "  make dist            Cross-compile for all supported platforms"
	@echo "  make install         Install the binary locally"
