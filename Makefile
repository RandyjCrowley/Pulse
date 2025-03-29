# Makefile for Docker Stack Manager

# Go parameters
BINARY_NAME=pulse
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOMOD=$(GO) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Build directory
BUILD_DIR=./build

# Go version check
GO_VERSION=$(shell go version | cut -d ' ' -f 3)

# Compiler flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: setup test build

# Setup dependencies and environment
setup:
	@echo "Setting up development environment..."
	$(GOMOD) tidy
	$(GOMOD) download
	$(GOMOD) verify

# Build for current platform
build:
	@echo "Building binary for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux .

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin .

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_windows.exe .

# Install the binary
install:
	@echo "Installing binary..."
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Run the application
run:
	@echo "Running the application..."
	$(GO) run .

# Check code quality and run linters
lint:
	@echo "Running code quality checks..."
	golangci-lint run

# Create a release tarball
release: clean build-all
	@echo "Creating release tarball..."
	@mkdir -p $(BUILD_DIR)/release
	tar -czvf $(BUILD_DIR)/release/$(BINARY_NAME)_$(GO_VERSION).tar.gz -C $(BUILD_DIR) $(BINARY_NAME)_linux $(BINARY_NAME)_darwin $(BINARY_NAME)_windows.exe

# Help target
help:
	@echo "Available targets:"
	@echo "  all        - Setup, and build the application"
	@echo "  setup      - Download and verify dependencies"
	@echo "  build      - Build for current platform"
	@echo "  build-all  - Build for Linux, macOS, and Windows"
	@echo "  install    - Install the binary to GOPATH"
	@echo "  clean      - Remove build artifacts"
	@echo "  run        - Run the application"
	@echo "  lint       - Run code quality checks"
	@echo "  release    - Create a release tarball"

.PHONY: all setup test build build-all build-linux build-darwin build-windows install clean run lint release help