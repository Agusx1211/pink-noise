.PHONY: all build build-linux build-mac build-all clean run test docker-build docker-linux

BINARY_NAME=pink-noise
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build

all: build

build:
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pink-noise

build-linux:
	@echo "Use 'make docker-linux' for cross-platform Linux builds"
	@echo "Or run 'make build' on a Linux system"

docker-linux:
	docker run --rm -v "$(PWD)":/app -w /app golang:1.24-alpine sh -c "\
		apk add --no-cache alsa-lib-dev build-base && \
		go mod download && \
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o build/$(BINARY_NAME)-linux-amd64 ./cmd/pink-noise && \
		CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-musl-gcc go build -ldflags='-s -w' -o build/$(BINARY_NAME)-linux-arm64 ./cmd/pink-noise \
	"

build-mac:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pink-noise
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pink-noise

build-all: build-mac
	@echo "For Linux builds, use: make docker-linux"

clean:
	rm -rf $(BUILD_DIR)

run:
	go run ./cmd/pink-noise

test:
	go test -v ./...

docker-build:
	docker build -t pink-noise:$(VERSION) .

.DEFAULT_GOAL := build
