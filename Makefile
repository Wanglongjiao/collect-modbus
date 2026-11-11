GO ?= go
BINARY := edge-modbus
OUT ?= output

APP_NAME ?= collect-modbus
VERSION ?= 0.0.160
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# -X 只能作用于包级 string 变量，路径必须与模块名一致
LDFLAGS ?= -s -w \
	-X collect-modbus/pkgs/env.AppName=$(APP_NAME) \
	-X collect-modbus/pkgs/env.Version=$(VERSION) \
	-X collect-modbus/pkgs/env.GitCommit=$(GIT_COMMIT) \
	-X collect-modbus/pkgs/env.BuildDate=$(BUILD_DATE)

.PHONY: all build test fmt vet lint clean amd64 arm64 armv7 win64

all: fmt vet test build

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(OUT)/$(BINARY) .

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else $(GO) vet ./...; fi

amd64:
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT)/$(BINARY)-linux-amd64 .

arm64:
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT)/$(BINARY)-linux-arm64 .

armv7:
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT)/$(BINARY)-linux-armv7 .

win64:
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT)/$(BINARY)-windows-amd64.exe .

clean:
	rm -rf $(OUT) logs
