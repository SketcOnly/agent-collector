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
all: clean compile deploy

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
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -x -v -o $(OUTPUT) $(SRC)
	@echo "$(GREEN)[SUCCESS] Compilation completed.$(RESET)"

# 部署目标
.PHONY: deploy
deploy:
	@echo "$(YELLOW)[INFO] Checking if remote file exists...$(RESET)"
	ssh -i $(SSH_KEY) $(REMOTE_USER)@$(REMOTE_HOST) '[ -f $(REMOTE_FILE) ] && rm -f $(REMOTE_FILE) && echo "$(RED)[WARNING] Old file removed.$(RESET)" || echo "$(GREEN)[INFO] No existing file to remove.$(RESET)"'
	@echo "$(YELLOW)[INFO] Deploying new file...$(RESET)"
	scp -i $(SSH_KEY) $(OUTPUT) $(REMOTE_USER)@$(REMOTE_HOST):$(REMOTE_PATH)
	@echo "$(GREEN)[SUCCESS] File deployed successfully.$(RESET)"
