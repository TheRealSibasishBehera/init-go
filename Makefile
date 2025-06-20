# Simple Go Init System Makefile

BINARY_NAME=init
DOCKER_IMAGE=go-init

# Default target
.PHONY: all
all: clean build

# Build
.PHONY: build
build:
	@echo "🔨 Building..."
	go build -o $(BINARY_NAME) ./cmd/init

.PHONY: build-static
build-static:
	@echo "🔨 Building static binary..."
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o $(BINARY_NAME) ./cmd/init

# Docker
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-test
docker-test: docker-build
	@echo "🧪 Testing Docker container..."
	@CONTAINER_ID=$$(docker run -d $(DOCKER_IMAGE)); \
	sleep 2; \
	echo "📋 Process tree:"; \
	docker exec $$CONTAINER_ID ps axjf; \
	echo "🧹 Cleaning up..."; \
	docker stop $$CONTAINER_ID; \
	docker rm $$CONTAINER_ID

# Development
.PHONY: fmt
fmt:
	@echo "🎨 Formatting..."
	go fmt ./...

.PHONY: test
test:
	@echo "🧪 Testing..."
	go test ./...

.PHONY: tidy
tidy:
	@echo "🧹 Tidying modules..."
	go mod tidy

# Cleanup
.PHONY: clean
clean:
	@echo "🧹 Cleaning..."
	rm -f $(BINARY_NAME)
	docker rm -f init-test 2>/dev/null || true

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-static - Build static binary for containers"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-test  - Test Docker container"
	@echo "  fmt          - Format code"
	@echo "  test         - Run tests"
	@echo "  tidy         - Tidy modules"
	@echo "  clean        - Clean up"
	@echo "  help         - Show this help"