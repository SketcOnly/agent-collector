APP := agent-collector
PKG := ../cmd/
VERSION := $(shell git describe --tags --always --dirty --match 'v*' 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# 支持环境变量覆盖构建目录，增强灵活性
BUILD_DIR ?= build
DOCKER_IMAGE := agent-collector:$(VERSION)

# 优化链接参数，增加-buildid=none进一步减小二进制体积
LDFLAGS := -s -w \
	-buildid=none \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# ------------------------------
# 日志函数（保持原有风格，统一输出格式）
# ------------------------------
define log
	echo "[$(shell date +'%Y-%m-%d %H:%M:%S')] $(1)"
endef

# ------------------------------
# 声明伪目标，避免与文件重名导致目标失效
# ------------------------------
.PHONY: build clean release docker run

# ------------------------------
# 构建本地二进制（增加错误检查，失败时终止）
# ------------------------------
build: | $(BUILD_DIR)
	$(call log, "🚀 Building $(APP) with verbose output...")
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP) $(PKG) || { \
		$(call log, "❌ Build failed"); \
		exit 1; \
	}
	$(call log, "✅ Build complete: $(BUILD_DIR)/$(APP)")

# ------------------------------
# 清理产物（确保目录下次构建可复用）
# ------------------------------
clean:
	$(call log, "🧹 Cleaning build artifacts...")
	rm -rf $(BUILD_DIR)
	$(call log, "✅ Clean complete")

# ------------------------------
# 发布Linux ARM64版本（增加错误检查，统一日志）
# ------------------------------
release: clean
	$(call log, "🌍 Building release for Linux ARM64...")
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
	go build -x -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP)-linux-arm64 $(PKG) || { \
		$(call log, "❌ Cross-compile failed"); \
		exit 1; \
	}
	$(call log, "🔑 Generating checksum...")
	sha256sum $(BUILD_DIR)/$(APP)-linux-arm64 > $(BUILD_DIR)/checksums.txt || { \
		$(call log, "❌ Checksum generation failed"); \
		exit 1; \
	}
	$(call log, "✅ Release build complete: $(BUILD_DIR)/$(APP)-linux-arm64")

# ------------------------------
# 构建Docker镜像（修复依赖路径，统一日志）
# ------------------------------
docker: $(BUILD_DIR)
	$(call log, "🐳 Building Docker image $(DOCKER_IMAGE)...")
	docker build -t $(DOCKER_IMAGE) . || { \
		$(call log, "❌ Docker build failed"); \
		exit 1; \
	}
	$(call log, "✅ Docker image ready: $(DOCKER_IMAGE)")

# ------------------------------
# 本地运行（修复依赖路径，统一日志）
# ------------------------------
run: $(BUILD_DIR)
	$(call log, "🏃 Running $(APP)...")
	./$(BUILD_DIR)/$(APP) || { \
		$(call log, "❌ Runtime failed"); \
		exit 1; \
	}

# ------------------------------
# 创建构建目录（确保存在）
# ------------------------------
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)