BINARY  := powerline-go
BIN_DIR := bin

# Build a fully static, dependency-free binary: CGO_ENABLED=0 drops the libc
# linkage that os/user would otherwise pull in, so the result runs on any
# x86-64 Linux machine regardless of glibc/musl. GOOS/GOARCH pin the target so
# the artifact is identical no matter what host it is built on.
GOOS       := linux
GOARCH     := amd64
BUILD_FLAGS := -trimpath
LDFLAGS    := -s -w

.DEFAULT_GOAL := build

.PHONY: build test vet fmt install preview clean help

## build: compile a static x64 Linux binary into ./bin (default target)
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) .

## test: run the test suite
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go source in place
fmt:
	gofmt -w .

## install: install a static binary into $GOBIN (or $GOPATH/bin)
install:
	CGO_ENABLED=0 go install $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' .

## preview: build, then regenerate the preview using ./generatePreview.sh
preview: build
	PATH="$(CURDIR)/$(BIN_DIR):$$PATH" ./generatePreview.sh

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## help: list the available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'
