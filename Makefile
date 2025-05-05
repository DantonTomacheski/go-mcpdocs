# Go parameters
BINARY_NAME=extract-data-go
OUTPUT_DIR=bin
MAIN_FILE=main.go

# Default target executed when you just run "make"
.DEFAULT_GOAL := build

.PHONY: all build run clean test help dev

all: build

## build: Compile the Go application
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${OUTPUT_DIR}
	go build -o ${OUTPUT_DIR}/${BINARY_NAME} ${MAIN_FILE}

## run: Build and run the Go application
run: build
	@echo "Running ${BINARY_NAME}..."
	./${OUTPUT_DIR}/${BINARY_NAME}

## dev: Run the Go application with live-reload using Air
dev:
	@echo "Starting development server with Air (live-reload)..."
	@air

## test: Run Go tests
test:
	@echo "Running tests..."
	go test ./... -v

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf ${OUTPUT_DIR}
	go clean

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
