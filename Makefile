# Makefile for Pulse - Docker Stack Manager TUI

# Application details
BINARY_NAME=pulse
VERSION=0.2.0
DESCRIPTION="A vibrant TUI for Docker stack management"

# Go parameters
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOMOD=$(GO) mod
GOTEST=$(GO) test

# Build directory
BUILD_DIR=./build
RELEASE_DIR=$(BUILD_DIR)/release

# Go version check
GO_VERSION=$(shell go version | cut -d ' ' -f 3)

# Compiler flags
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Color output
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

# Default target
all: setup build

# Setup dependencies and environment
setup:
	@echo "${BLUE}Setting up development environment...${NC}"
	$(GOMOD) tidy
	$(GOMOD) download
	$(GOMOD) verify
	@echo "${GREEN}Setup complete!${NC}"

# Build for current platform
build:
	@echo "${BLUE}Building binary for current platform...${NC}"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "${GREEN}Build complete! Binary: $(BUILD_DIR)/$(BINARY_NAME)${NC}"

# Build for multiple platforms
build-all: build-linux build-darwin build-windows
	@echo "${GREEN}All platform builds complete!${NC}"

# Build for Linux
build-linux:
	@echo "${BLUE}Building for Linux...${NC}"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux .
	@echo "${GREEN}Linux build complete!${NC}"

# Build for macOS
build-darwin:
	@echo "${BLUE}Building for macOS...${NC}"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin .
	@echo "${GREEN}macOS build complete!${NC}"

# Build for Windows
build-windows:
	@echo "${BLUE}Building for Windows...${NC}"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_windows.exe .
	@echo "${GREEN}Windows build complete!${NC}"

# Install the binary
install:
	@echo "${BLUE}Installing binary...${NC}"
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) .
	@echo "${GREEN}Installation complete! Binary: $(GOPATH)/bin/$(BINARY_NAME)${NC}"

# Clean build artifacts
clean:
	@echo "${BLUE}Cleaning build artifacts...${NC}"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "${GREEN}Clean complete!${NC}"

# Run tests
test:
	@echo "${BLUE}Running tests...${NC}"
	$(GOTEST) -v ./...

# Run the application
run:
	@echo "${BLUE}Running the application...${NC}"
	$(GO) run .

# Run the application in debug mode
run-debug:
	@echo "${BLUE}Running the application in debug mode...${NC}"
	$(GO) run . --debug

# Check code quality and run linters
lint:
	@echo "${BLUE}Running code quality checks...${NC}"
	golangci-lint run
	@echo "${GREEN}Lint complete!${NC}"

# Create a release tarball
release: clean build-all
	@echo "${BLUE}Creating release tarball...${NC}"
	@mkdir -p $(RELEASE_DIR)
	tar -czvf $(RELEASE_DIR)/$(BINARY_NAME)_$(VERSION).tar.gz -C $(BUILD_DIR) $(BINARY_NAME)_linux $(BINARY_NAME)_darwin $(BINARY_NAME)_windows.exe
	@echo "${GREEN}Release created: $(RELEASE_DIR)/$(BINARY_NAME)_$(VERSION).tar.gz${NC}"

# Help target
help:
	@echo "${YELLOW}Available targets:${NC}"
	@echo "  ${GREEN}all${NC}        - Setup and build the application"
	@echo "  ${GREEN}setup${NC}      - Download and verify dependencies"
	@echo "  ${GREEN}build${NC}      - Build for current platform"
	@echo "  ${GREEN}build-all${NC}  - Build for Linux, macOS, and Windows"
	@echo "  ${GREEN}install${NC}    - Install the binary to GOPATH"
	@echo "  ${GREEN}clean${NC}      - Remove build artifacts"
	@echo "  ${GREEN}test${NC}       - Run tests"
	@echo "  ${GREEN}run${NC}        - Run the application"
	@echo "  ${GREEN}run-debug${NC}  - Run the application with debug output"
	@echo "  ${GREEN}lint${NC}       - Run code quality checks"
	@echo "  ${GREEN}release${NC}    - Create a release tarball"

.PHONY: all setup test build build-all build-linux build-darwin build-windows install clean run run-debug lint release docker-dev help