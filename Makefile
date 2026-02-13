ifeq ($(OS),Windows_NT)
  SHELL := cmd.exe
  .SHELLFLAGS := /C
  VERSION ?= $(shell git describe --tags --always --dirty 2>NUL || echo dev)
  MKDIR_BIN = if not exist bin mkdir bin
  RM_BIN = if exist bin rmdir /s /q bin
else
  VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
  MKDIR_BIN = mkdir -p bin
  RM_BIN = rm -rf bin
endif

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-wsl build-host test clean install

build: build-wsl build-host

build-wsl: export GOOS = linux
build-wsl: export GOARCH = amd64
build-wsl:
	@$(MKDIR_BIN)
	@echo Building wstart for linux/amd64...
	@go build $(LDFLAGS) -o bin/wstart ./cmd/wstart

build-host: export GOOS = windows
build-host: export GOARCH = amd64
build-host:
	@$(MKDIR_BIN)
	@echo Building wstart-host.exe for windows/amd64...
	@go build $(LDFLAGS) -o bin/wstart-host.exe ./cmd/wstart-host

test:
	go test ./...

clean:
	@$(RM_BIN)
	@echo Cleaned bin/

# Install from WSL.
install:
	@./install-wsl.sh
	@echo ""
	@echo "To install the host helper, run from PowerShell:"
	@echo "  .\\install-host.ps1"
