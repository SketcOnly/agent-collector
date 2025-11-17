APP := yourapp
PKG := ./cmd/$(APP)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

BUILD_DIR := build
DIST_DIR := dist
DOCKER_IMAGE := yourapp:$(VERSION)

# ------------------------------
# Build Go binary
# ------------------------------
build: | $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP) $(PKG)

# ------------------------------
# Clean artifacts
# ------------------------------
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

# ------------------------------
# Release: build + cross compile + checksum
# ------------------------------
release: clean | $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-amd64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-arm64 $(PKG)
	sha256sum $(DIST_DIR)/* > $(DIST_DIR)/checksums.txt

# ------------------------------
# Docker image
# ------------------------------
docker: build
	docker build -t $(DOCKER_IMAGE) .

# ------------------------------
# Run locally
# ------------------------------
run: build
	./$(BUILD_DIR)/$(APP)

# ------------------------------
# Create directories
# ------------------------------
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(DIST_DIR):
	mkdir -p $(DIST_DIR)
