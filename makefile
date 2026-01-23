# 定义变量
GOOS = linux
GOARCH = arm64
SRC = ./cmd/main.go
OUTPUT = build/agent-collector-arm64
SSH_KEY = ~/.ssh/id_rsa
REMOTE_USER = sketc
REMOTE_HOST = 10.32.9.134
REMOTE_PATH = /home/sketc/
REMOTE_FILE = $(REMOTE_PATH)$(notdir $(OUTPUT))

# 颜色定义
GREEN = \033[0;32m
RED = \033[0;31m
YELLOW = \033[0;33m
RESET = \033[0m

# 默认目标
.PHONY: all
all: clean compile memory-check deploy

# 清理目标
.PHONY: clean
clean:
	@echo "$(YELLOW)[INFO] Cleaning build directory...$(RESET)"
	rm -rf build/*
	@echo "$(GREEN)[SUCCESS] Build directory cleaned.$(RESET)"

# 编译目标
.PHONY: compile
compile:
	@echo "$(YELLOW)[INFO] Compiling project...$(RESET)"
	@sleep 3s
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -x -v -o $(OUTPUT) $(SRC)
	@echo "$(GREEN)[SUCCESS] Compilation completed.$(RESET)"

# 动态内存预估提示（极简写法）
.PHONY: memory-check
memory-check:
	@echo "$(YELLOW)[INFO] Starting binary memory detection...$(RESET)"
	@sleep 2s

	# 1. 二进制文件大小检测
	@echo "$(YELLOW)[INFO] Binary file size analysis...$(RESET)"
	@if [ -f $(OUTPUT) ]; then \
		du -h $(OUTPUT) | awk '{printf "$(GREEN)[INFO] Binary file size (human): %s (%s)$(RESET)\n", $$1, $$2}'; \
		du -k $(OUTPUT) | awk '{printf "$(GREEN)[INFO] Exact binary size (bytes): %d$(RESET)\n", $$1 * 1024}'; \
	else \
		echo "$(RED)[ERROR] Binary file $(OUTPUT) not found!$(RESET)"; \
		exit 1; \
	fi

	# 2. 静态内存分析
	@echo "$(YELLOW)[INFO] Static memory layout analysis (macOS compatible)...$(RESET)"
	@command -v go >/dev/null 2>&1; GO_EXISTS=$$?
	@if [ -z "$$GO_EXISTS" ]; then \
		echo "$(YELLOW)[WARNING] Go environment not found, skipping static analysis.$(RESET)"; \
	else \
		if [ $$GO_EXISTS -eq 0 ]; then \
			go tool objdump -h >/dev/null 2>&1; OBJDUMP_EXISTS=$$?; \
			if [ $$OBJDUMP_EXISTS -eq 0 ]; then \
				echo "$(GREEN)===== Binary Section Info (objdump) =====$(RESET)"; \
				go tool objdump -s $(OUTPUT) | awk ' \
					/\.text/ {text += strtonum("0x" $$3)} \
					/\.data/ {data += strtonum("0x" $$3)} \
					/\.bss/ {bss += strtonum("0x" $$3)} \
					END { \
						printf "$(GREEN)[INFO] .text (code) size: %d bytes$(RESET)\n", text; \
						printf "$(GREEN)[INFO] .data (init data) size: %d bytes$(RESET)\n", data; \
						printf "$(GREEN)[INFO] .bss (uninit data) size: %d bytes$(RESET)\n", bss; \
						printf "$(GREEN)[INFO] Total static memory: %d bytes$(RESET)\n", text+data+bss; \
					}'; \
			else \
				echo "$(YELLOW)[WARNING] go tool objdump not available, skip section analysis.$(RESET)"; \
				file $(OUTPUT) | awk '{printf "$(GREEN)[INFO] Binary info: %s$(RESET)\n", $$0}'; \
			fi; \
		fi; \
	fi

	# 3. 动态内存预估提示（极简写法）
	@echo "$(YELLOW)[INFO] Dynamic memory usage preview...$(RESET)"
	@if [ -f $(OUTPUT) ]; then \
		echo "$(GREEN)[INFO] Deploy first, then run on remote:$(RESET)"; \
		echo "$(GREEN)ssh -i $(SSH_KEY) $(REMOTE_USER)@$(REMOTE_HOST) 'ps aux | grep $(notdir $(OUTPUT))'$(RESET)"; \
	else \
		echo "$(RED)[ERROR] Binary not found, skip dynamic check.$(RESET)"; \
	fi

	@echo "$(GREEN)[SUCCESS] Memory detection completed.$(RESET)"
	@sleep 2s

# 部署目标（极简单命令，无嵌套）
.PHONY: deploy
deploy:
	@echo "$(YELLOW)[INFO] Checking remote file...$(RESET)"
	@sleep 3s
	@ssh -i $(SSH_KEY) $(REMOTE_USER)@$(REMOTE_HOST) "if [ -f $(REMOTE_FILE) ]; then rm -f $(REMOTE_FILE); echo '$(RED)[WARNING] Old file removed.$(RESET)'; else echo '$(GREEN)[INFO] No old file.$(RESET)'; fi"
	@echo "$(YELLOW)[INFO] Deploying file...$(RESET)"
	@sleep 3s
	@scp -i $(SSH_KEY) $(OUTPUT) $(REMOTE_USER)@$(REMOTE_HOST):$(REMOTE_PATH)
	@echo "$(GREEN)[SUCCESS] File deployed to $(REMOTE_FILE)$(RESET)"

	# 部署后内存检查提示
	@echo "$(YELLOW)[INFO] Check runtime memory on remote:$(RESET)"
	@echo "$(GREEN)ssh -i $(SSH_KEY) $(REMOTE_USER)@$(REMOTE_HOST) 'top -p \$$(pgrep $(notdir $(OUTPUT))) -l 1'$(RESET)"