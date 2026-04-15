MODULE   := github.com/trganda/codeql-development-toolkit
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-X $(MODULE)/cmd.Version=$(VERSION)"

BINARY   := qlt
BUILD_DIR := dist

.PHONY: all build install clean test lint

all: build

## build: compile the binary into dist/
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

## install: install the binary to $GOPATH/bin (or ~/go/bin)
install:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(shell go env GOPATH)/bin/$(BINARY) .

## test: run all Go tests
test:
	go test ./...

## lint: run go vet
lint:
	go vet ./...

## clean: remove build artefacts
clean:
	rm -rf $(BUILD_DIR)
