# Project name
APP := gonduit

# Supported platforms
OSES := linux windows darwin freebsd
ARCHES := 386 amd64 arm64

# Output directory
BIN := bin

# Source file
SRC := cmd/main.go

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "init")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Build flags for optimization and size reduction
LDFLAGS := -s -w -extldflags=-static \
	-X 'pkg.Version=$(VERSION)' \
	-X 'pkg.Commit=$(COMMIT)' \
	-X 'pkg.BuildDate=$(BUILD_DATE)'

GCFLAGS := -trimpath
ASMFLAGS := -trimpath

# Default target
all: build-all

# Build all OS/ARCH combinations
build-all:
	@for os in $(OSES); do \
		for arch in $(ARCHES); do \
			[ "$$os" = "darwin" ] && [ "$$arch" = "386" ] && continue; \
			out="$(BIN)/$(APP)-$$os-$$arch"; \
			[ "$$os" = "windows" ] && out="$$out.exe"; \
			echo "Building $$os/$$arch -> $$out"; \
			GOOS=$$os GOARCH=$$arch go build \
				-ldflags="$(LDFLAGS)" \
				-gcflags="$(GCFLAGS)" \
				-asmflags="$(ASMFLAGS)" \
				-trimpath \
				-o $$out $(SRC) || exit 1; \
		done; \
	done

# Clean build artifacts
clean:
	rm -rf $(BIN)

.PHONY: all build-all clean