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
	$(MKBIN)
	go build $(LDFLAGS) -o bin/wstart ./cmd/wstart

build-host: export GOOS = windows
build-host: export GOARCH = amd64
build-host:
	$(MKBIN)
	go build $(LDFLAGS) -o bin/wstart-host.exe ./cmd/wstart-host

test:
	go test ./...

clean:
	$(RM)

# Install pre-built binaries. Run from within WSL.
install:
	@test -f bin/wstart || (echo "bin/wstart not found. Run 'make build' first." && exit 1)
	@test -f bin/wstart-host.exe || (echo "bin/wstart-host.exe not found. Run 'make build' first." && exit 1)
	mkdir -p ~/.local/bin
	cp bin/wstart ~/.local/bin/
	@WINAPPDATA=$$(cmd.exe /C 'echo %LOCALAPPDATA%' 2>/dev/null | tr -d '\r'); \
	if [ -n "$$WINAPPDATA" ]; then \
		WSLPATH=$$(wslpath "$$WINAPPDATA"); \
		mkdir -p "$$WSLPATH/wstart"; \
		cp bin/wstart-host.exe "$$WSLPATH/wstart/"; \
		echo "Installed wstart to ~/.local/bin/"; \
		echo "Installed wstart-host.exe to $$WINAPPDATA\\wstart\\"; \
	else \
		echo "WARNING: Could not detect Windows LOCALAPPDATA."; \
		echo "Copy bin/wstart-host.exe to %%LOCALAPPDATA%%\\wstart\\ manually."; \
	fi
