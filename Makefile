.PHONY: build test lint clean install release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

# Build binary
build:
	go build -ldflags="$(LDFLAGS)" -o ip2cc ./cmd/ip2cc/

# Run tests
test:
	go test -v -race -cover ./...

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f ip2cc
	rm -rf dist/

# Install to GOPATH/bin
install:
	go install -ldflags="$(LDFLAGS)" ./cmd/ip2cc/

# Build for all platforms
release: clean
	mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/ip2cc_linux_amd64 ./cmd/ip2cc/
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/ip2cc_linux_arm64 ./cmd/ip2cc/
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/ip2cc_darwin_amd64 ./cmd/ip2cc/
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/ip2cc_darwin_arm64 ./cmd/ip2cc/
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/ip2cc_windows_amd64.exe ./cmd/ip2cc/
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/ip2cc_windows_arm64.exe ./cmd/ip2cc/
	cd dist && sha256sum * > checksums.txt

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run the update command (for development)
dev-update:
	go run ./cmd/ip2cc/ update --concurrency 4

# Run a test lookup (for development)
dev-lookup:
	go run ./cmd/ip2cc/ 8.8.8.8

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  clean    - Clean build artifacts"
	@echo "  install  - Install to GOPATH/bin"
	@echo "  release  - Build for all platforms"
	@echo "  coverage - Generate coverage report"
