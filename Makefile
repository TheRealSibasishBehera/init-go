# Simple Go Init System Makefile

BINARY_NAME=init
DOCKER_IMAGE=go-init

# Default target
.PHONY: all
all: clean build

# Build
.PHONY: build
build:
	@echo "Building..."
	go build -o $(BINARY_NAME) ./cmd/init

.PHONY: build-static
build-static:
	@echo "Building static binary..."
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o $(BINARY_NAME) ./cmd/init

# Docker Production
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-test
docker-test: docker-build
	@echo "Testing Docker container..."
	@CONTAINER_ID=$$(docker run -d $(DOCKER_IMAGE)); \
	sleep 2; \
	echo "Process tree:"; \
	docker exec $$CONTAINER_ID ps axjf; \
	echo "Cleaning up..."; \
	docker stop $$CONTAINER_ID; \
	docker rm $$CONTAINER_ID

# Docker Development
.PHONY: docker-dev-build
docker-dev-build:
	@echo "Building development Docker image..."
	docker build -f Dockerfile.dev -t $(DOCKER_IMAGE)-dev .

.PHONY: docker-dev-test
docker-dev-test: docker-dev-build
	@echo "Running tests in Docker..."
	docker run --rm -v "$$(pwd):/app" $(DOCKER_IMAGE)-dev make test

.PHONY: docker-dev-build-binary
docker-dev-build-binary: docker-dev-build
	@echo "Building binary in Docker..."
	docker run --rm -v "$$(pwd):/app" $(DOCKER_IMAGE)-dev make build-static

.PHONY: docker-dev-shell
docker-dev-shell: docker-dev-build
	@echo "Opening development shell..."
	docker run --rm -it -v "$$(pwd):/app" $(DOCKER_IMAGE)-dev sh

# Development
.PHONY: fmt
fmt:
	@echo "Formatting..."
	go fmt ./...

.PHONY: test
test:
	@echo "Testing..."
	go test ./...

.PHONY: tidy
tidy:
	@echo "Tidying modules..."
	go mod tidy

# Cleanup
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	docker rm -f init-test 2>/dev/null || true

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build                 - Build the binary"
	@echo "  build-static          - Build static binary for containers"
	@echo "  docker-build          - Build production Docker image"
	@echo "  docker-test           - Test production Docker container"
	@echo "  docker-dev-build      - Build development Docker image"
	@echo "  docker-dev-test       - Run tests in Docker container"
	@echo "  docker-dev-build-binary - Build binary in Docker container"
	@echo "  docker-dev-shell      - Open development shell in container"
	@echo "  fmt                   - Format code"
	@echo "  test                  - Run tests"
	@echo "  tidy                  - Tidy modules"
	@echo "  clean                 - Clean up"
	@echo "  help                  - Show this help"