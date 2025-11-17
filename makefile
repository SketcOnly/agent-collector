APP := agent-collector
PKG := ./cmd/
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

BUILD_DIR := build
DOCKER_IMAGE := yourapp:$(VERSION)


# ------------------------------
# Helper for logging
# ------------------------------
define log
	@echo "[$(shell date +'%Y-%m-%d %H:%M:%S')] $(1)"
endef


# ------------------------------
# Build Go binary
# ------------------------------
build: | $(BUILD_DIR)
	$(call log, "🚀 Building $(APP) with verbose output...")
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP) $(PKG)
	$(call log, "✅ Build complete: $(BUILD_DIR)/$(APP)")

# ------------------------------
# Clean artifacts
# ------------------------------
clean:
	$(call log, "🧹 Cleaning build artifacts...")
	rm -rf $(BUILD_DIR)
	$(call log, "✅ Clean complete")

# ------------------------------
# Release: build + cross compile + checksum
# ------------------------------
release: clean
	$(call log, "🌍 Building release for Linux ARM64...")
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -x  -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP)-linux-arm64 $(PKG)
	$(call log, "🔑 Generating checksum...")
	sha256sum $(BUILD_DIR)/$(APP)-linux-arm64 > $(BUILD_DIR)/checksums.txt
	$(call log, "✅ Release build complete: $(BUILD_DIR)/$(APP)-linux-arm64")

# ------------------------------
# Docker image
# ------------------------------
docker: build
	@echo "🐳 Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image ready: $(DOCKER_IMAGE)"

# ------------------------------
# Run locally
# ------------------------------
run: build
	@echo "🏃 Running $(APP)..."
	./$(BUILD_DIR)/$(APP)

# ------------------------------
# Create directories
# ------------------------------
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)
