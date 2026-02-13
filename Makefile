VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Cross-platform commands
ifeq ($(OS),Windows_NT)
    RM = if exist bin rmdir /S /Q bin
    MKBIN = if not exist bin mkdir bin
else
    RM = rm -rf bin
    MKBIN = mkdir -p bin
endif

.PHONY: build build-wsl build-host test clean install

build: build-wsl build-host

build-wsl: export GOOS = linux
build-wsl: export GOARCH = amd64
build-wsl:
	@$(MKBIN)
	@echo "Building wstart (linux/amd64)..."
	@go build $(LDFLAGS) -o bin/wstart ./cmd/wstart

build-host: export GOOS = windows
build-host: export GOARCH = amd64
build-host:
	@$(MKBIN)
	@echo "Building wstart-host.exe (windows/amd64)..."
	@go build $(LDFLAGS) -o bin/wstart-host.exe ./cmd/wstart-host

test:
	go test ./...

clean:
	@$(RM)
	@echo "Cleaned bin/"

# Install from WSL. Runs both install scripts.
install:
	@./install-wsl.sh
	@echo ""
	@echo "To install the host helper, run from PowerShell:"
	@echo "  .\\install-host.ps1"
