PROJECT_DIR := $(CURDIR)/
BIN_DIR := $(PROJECT_DIR)/bin

.PHONY: all
all: help

## General

.PHONY: help
help:
	@echo "Choose one of the following target"
	@echo
	@echo "fmt            Run go fmt against code."
	@echo "vet            Run go vet against code."
	@echo "build          Build all binaries."
	@echo "install        Install all binaries."

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/ ./cmd/...

.PHONY: install
install:
	go install ./cmd/update-actions
	go install ./cmd/update-gomod
