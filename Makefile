# Project name
APP := gonduit

# Supported platforms
OSES := linux windows darwin
ARCHES := 386 amd64 arm64

# Output directory
BIN := bin

# Build selection
CMD ?= server
OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

# Source files
SRC_app := cmd/app/main.go
SRC_server := cmd/server/main.go
SRC = $(if $(filter app,$(CMD)),$(SRC_app),$(SRC_server))

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "init")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Build flags for optimization and size reduction
LDFLAGS := -s -w -extldflags=-static \
	-X 'shared/pkg.Version=$(VERSION)' \
	-X 'shared/pkg.Commit=$(COMMIT)' \
	-X 'shared/pkg.BuildDate=$(BUILD_DATE)'

GCFLAGS := -trimpath
ASMFLAGS := -trimpath

# Output file
OUT_BASE = $(BIN)/$(APP)-$(CMD)-$(OS)-$(ARCH)
OUT = $(if $(filter windows,$(OS)),$(OUT_BASE).exe,$(OUT_BASE))

# Build for selected OS/ARCH (defaults to current)
build:
	@echo "Building $(CMD) $(OS)/$(ARCH) -> $(OUT)"
	@GOOS=$(OS) GOARCH=$(ARCH) go build \
		-ldflags="$(LDFLAGS)" \
		-gcflags="$(GCFLAGS)" \
		-asmflags="$(ASMFLAGS)" \
		-trimpath \
		-o $(OUT) $(SRC)

# Build all OS/ARCH combinations for selected command
all:
	@for os in $(OSES); do \
		for arch in $(ARCHES); do \
			[ "$$os" = "darwin" ] && [ "$$arch" = "386" ] && continue; \
			out="$(BIN)/$(APP)-$(CMD)-$$os-$$arch"; \
			[ "$$os" = "windows" ] && out="$$out.exe"; \
			echo "Building $(CMD) $$os/$$arch -> $$out"; \
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

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "Error: protoc not found"; \
		exit 1; \
	fi
	@if ! command -v protoc-gen-go >/dev/null 2>&1; then \
		echo "Error: protoc-gen-go not found"; \
		exit 1; \
	fi
	@if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		echo "Error: protoc-gen-go-grpc not found"; \
		exit 1; \
	fi
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/shared/proto/gonduit.proto
	@echo "Protobuf code generated successfully"

.PHONY: build all clean app server proto

app: CMD=app
app: build

server: CMD=server
server: build
