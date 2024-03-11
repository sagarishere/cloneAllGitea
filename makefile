# Makefile for building and running a Go project

# Project-specific settings
BINARY_NAME=cloneAllGitea
BUILD_DIR=./bin
SOURCE_DIR=.

# Go build commands
GO_BUILD=go build -ldflags "-s -w"
GO_CLEAN=go clean

# Makefile targets
.PHONY: all build clean run

all: build

build:
	mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(SOURCE_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

clean:
	$(GO_CLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
