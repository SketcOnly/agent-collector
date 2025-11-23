# ==============================================================================
# 简单版 Makefile - 编译 Go 程序并传输文件
# ==============================================================================

SHELL := /bin/zsh  # macOS 默认 shell

# ===================== 配置 =====================
BUILD_DIR ?= build/
BINARY_NAME ?= agent-collector-arm64
GO_SRC_DIR ?= ./cmd/
KEY_FILE ?= ~/.ssh/id_rsa
DEST_DIR ?= /home/sketc
FILE1 ?= $(BUILD_DIR)$(BINARY_NAME)
FILE2 ?= configs/config.yaml
FILES ?= $(FILE1) $(FILE2)

# 目标主机列表
HOSTS ?= \
  sketc@10.32.9.134

# Go 架构配置
GOOS ?= linux
GOARCH ?= arm64
GOFLAGS ?= -x -v -ldflags "-w -s"

# ===================== 日志函数 =====================
define log_info
	@echo "[INFO] $$(date +%Y-%m-%dT%H:%M:%S%z) - $(1)"
endef

define log_success
	@echo "[SUCCESS] $$(date +%Y-%m-%dT%H:%M:%S%z) - $(1)"
endef

define log_error
	@echo "[FAILED] $$(date +%Y-%m-%dT%H:%M:%S%z) - $(1)"
endef

# ===================== 1. 编译 =====================
.PHONY: compile
compile:
	$(call log_info, "START 编译 Go 程序 GOOS=$(GOOS), GOARCH=$(GOARCH)")
	@mkdir -p $(BUILD_DIR)
	@echo "Running: GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -o $(FILE1) $(GO_SRC_DIR)main.go"
	@cd $(GO_SRC_DIR) && GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -o ../$(FILE1) main.go || { \
		$(call log_error, "Go 编译失败"); exit 1; }
	$(call log_success, "编译完成 $(FILE1)")

# ===================== 2. SCP 多文件传输 =====================
define scp_host
user_ip=$(1); port=$(2); \
$(call log_info, "TRANSFER 到 $$user_ip 端口 $$port 开始"); \
scp -i $(KEY_FILE) -P $$port -o StrictHostKeyChecking=no $(FILES) $$user_ip:$(DEST_DIR) || { \
	$(call log_error, "传输到 $$user_ip 失败"); exit 1; }; \
$(call log_success, "传输到 $$user_ip 完成")
endef

.PHONY: deploy
deploy: compile
	@for f in $(FILES); do \
		if [ ! -f $$f ]; then \
			$(call log_error, "文件不存在: $$f"); exit 1; \
		fi; \
	done
	$(call log_info, "START 传输 $(words $(HOSTS)) 台主机")
	@for host in $(HOSTS); do \
		user_ip=$${host%%:*}; port=$${host##*:}; \
		if [ "$$user_ip" = "$$port" ]; then port=22; fi; \
		$(call scp_host,$$user_ip,$$port); \
	done
	$(call log_success, "所有主机传输完成")

# ===================== 3. 总目标 =====================
.PHONY: all
all: compile deploy

# ===================== 4. 清理 =====================
.PHONY: clean
clean:
	$(call log_info, "START 清理 build 目录")
	@rm -rf $(BUILD_DIR)
	$(call log_success, "清理完成")
