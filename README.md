# agent-Collector

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/doc/install)
[![CI](https://github.com/SketcOnly/agent-collector/actions/workflows/ci.yml/badge.svg)](https://github.com/SketcOnly/agent-collector/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prometheus/client_golang)](https://goreportcard.com/report/github.com/prometheus/client_golang)
[![Go Reference](https://pkg.go.dev/badge/github.com/prometheus/client_golang.svg)](https://pkg.go.dev/github.com/prometheus/client_golang)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/prometheus/client_golang/badge)](https://securityscorecards.dev/viewer/?uri=github.com/prometheus/client_golang)
[![Slack](https://img.shields.io/badge/join%20slack-%23prometheus--client_golang-brightgreen.svg)](https://slack.cncf.io/)
[![快速开始](https://img.shields.io/badge/快速开始-🚀%20立即上手-blue?logo=rocket)](#quickstart)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/SketcOnly/agent-collector/blob/main/LICENSE)

## 项目简介

[![Go Reference](https://pkg.go.dev/badge/github.com/prometheus/client_golang.svg)](https://pkg.go.dev/github.com/prometheus/client_golang)

Agent-Collector 是一款轻量级、高性能的数据采集代理工具，专注于日志采集、指标上报、数据转发等核心场景。支持自定义日志格式、多维度数据过滤、高并发数据传输，适用于微服务架构、云原生环境下的统一数据采集需求。

## 核心价值：

[![Go Reference](https://pkg.go.dev/badge/github.com/prometheus/client_golang.svg)](https://pkg.go.dev/github.com/prometheus/client_golang)

+ 统一采集入口：整合日志、指标等多类型数据，避免重复开发采集逻辑； 
+ 高性能低开销：基于 Go 语言开发，支持百万级日志 / 指标的实时采集； 
+ 高度可配置：支持自定义日志字段、采集规则、转发目标，适配复杂业务场景； 
+ 易集成易部署：支持本地部署、容器化部署，提供简洁的 SDK 供业务系统集成


## 核心特性

[![Go Reference](https://pkg.go.dev/badge/github.com/prometheus/client_golang.svg)](https://pkg.go.dev/github.com/prometheus/client_golang)

1. 日志采集与处理
+ 支持自定义日志编码器（控制台彩色输出 + 文件 JSON 格式存储）； 
+ 日志字段自定义排序（默认：时间 → 级别 → 调用者 → collector → gid → 消息 → 扩展字段）； 
+ 日志级别过滤（Debug/Info/Warn/Error/Panic/Fatal）； 
+ 日志轮转（按时间 / 文件大小拆分，自动清理过期日志）； 
+ 支持多目录日志采集、日志内容过滤（关键词匹配）。

2. 指标采集与上报 
+ 系统指标采集（CPU / 内存 / 磁盘 / 网络使用率）；
+ 自定义业务指标上报（计数器、 gauge、直方图）；
+ 支持 Prometheus 协议导出，兼容 Grafana 可视化； 
+ 指标聚合计算（平均值、最大值、求和等）。

3. 数据转发与输出 
+ 支持本地文件存储（JSON 格式，便于后续分析）； 
+ 支持转发至 Kafka/RabbitMQ 消息队列； 
+ 支持 HTTP 接口转发（自定义回调地址）； 
+ 数据批量转发，降低网络开销。

4. 高可用与扩展性
+ 支持集群部署，避免单点故障； 
+ 插件化架构，可扩展新的采集 / 转发插件； 
+ 配置热更新，无需重启服务即可生效； 
+ 完善的错误重试机制，确保数据不丢失。


## <a id="quickstart"></a>快速开始

### 1. 环境要求
+ Go 版本：≥ 1.19（推荐 1.20+）
+ 依赖工具：Git、Make（可选）、Docker（可选，容器化部署） 
+ 支持系统：Linux、macOS、Windows

### 2. 项目克隆

```bash
# 克隆代码仓库
git clone https://github.com/SketcOnly/agent-collector.git
cd monitor-collector
```

### 2. 依赖安装
```bash
# 拉取项目依赖
go mod tidy
```

### 3. 配置文件修改
复制默认配置文件并修改（支持 yaml/json 格式）：
```bash
# 复制默认配置
cp configs/config.example.yaml configs/config.yaml
```
#### 核心配置示例（configs/config.yaml）：
```yaml
# 服务配置
server:
port: 8080                  # HTTP 服务端口（用于指标查询、配置管理）
log_level: "info"           # 服务日志级别（debug/info/warn/error）
timeout: 30s                # 请求超时时间

# 日志采集配置
logger:
enable: true                # 是否启用日志采集
log_dir: "./logs"           # 日志存储目录
rotate:
max_age: 7d               # 日志最大保留时间（7天）
max_size: 100MB           # 单个日志文件最大大小（100MB）
rotate_time: 24h          # 日志轮转周期（24小时）
fields:                     # 自定义日志固定字段
collector: "default"      # 默认 collector 字段值
gid: "default-group"      # 默认 gid 字段值

# 指标采集配置
metrics:
enable: true                # 是否启用指标采集
scrape_interval: 15s        # 指标采集间隔（15秒）
prometheus:
enable: true              # 是否启用 Prometheus 导出
path: "/metrics"          # Prometheus 指标暴露路径

# 数据转发配置
forward:
kafka:
enable: false             # 是否启用 Kafka 转发
brokers: ["127.0.0.1:9092"] # Kafka 集群地址
topic: "monitor-collector-data" # 转发目标 Topic
http:
enable: false             # 是否启用 HTTP 转发
url: "http://127.0.0.1:8081/receive" # 回调地址
```

### 4. 本地启动
```bash
# 直接启动
go run cmd/monitor-collector/main.go
# 或使用 Make 启动（推荐，包含编译优化）
make run
```

### 5. 验证服务可用性
```bash
# 访问指标接口，验证服务是否启动成功
curl http://127.0.0.1:8080/metrics
# 若返回 Prometheus 格式的指标数据，说明服务启动成功！
```

## 开发指南
### 1. 分支规范 
   + main：主分支，保持稳定可部署状态，仅通过 PR 合并 dev 分支； 
   + dev：开发分支，所有功能开发、bug 修复均在 dev 或基于 dev 的 Feature 分支进行； 
   + feature/xxx：功能分支，基于 dev 创建，命名格式：feature/功能名称（如 feature/kafka-forward）； 
   + bugfix/xxx：bug 修复分支，基于 dev 创建，命名格式：bugfix/问题描述（如 bugfix/log-encoder-error）。
---
### 2. 本地开发流程
```bash
# 1. 切换到 dev 分支并同步最新代码
git switch dev
git pull origin dev

# 2. 基于 dev 创建 Feature 分支
git switch -c feature/your-feature-name

# 3. 开发完成后，提交代码（遵循 commit 规范）
git add .
git commit -m "【功能新增】：新增 Kafka 数据转发功能
- 实现 Kafka 生产者初始化逻辑，支持集群地址配置
- 新增重试机制，确保数据转发可靠性
- 添加配置校验，避免无效配置导致服务启动失败"

# 4. 推送 Feature 分支到远程
git push origin feature/your-feature-name

# 5. 在 GitHub 上创建 PR，目标分支为 dev
```
---
### 3. Commit 规范
   遵循「结构化备注」格式，示例： 
   
```plaintext
  【类型】：核心修改概括（≤50字）

- 具体修改1：说明做了什么 + 目的
- 具体修改2：说明做了什么 + 目的
- 修复/优化：问题描述 + 解决方案（如有）
  类型可选：功能新增、bug修复、代码优化、配置调整、文档更新、依赖升级。
```
---
### 4. 代码编译与测试
```bash
# 编译项目（生成二进制文件）
make build

# 运行单元测试
make test

# 运行代码 lint 检查（确保代码规范）
make lint
```
---

### 5. 冲突处理
若开发过程中遇到分支冲突（如 dev 分支有新提交）：
```bash
# 1. 切换到 dev 分支，拉取最新代码
git switch dev
git pull origin dev

# 2. 切换回 Feature 分支，合并 dev 分支
git switch feature/your-feature-name
git merge dev

# 3. 手动解决冲突（编辑冲突文件，删除冲突标记）
# 4. 标记冲突已解决并提交
git add .
git commit -m "合并 dev 分支，解决冲突"
```
---


## 部署文档

1.本地部署（开发 / 测试环境）
```bash
# 1. 编译生成二进制文件
make build

# 2. 启动服务（指定配置文件）
./bin/monitor-collector --config configs/config.yaml

# 3. 后台运行（Linux/macOS）
nohup ./bin/monitor-collector --config configs/config.yaml > ./logs/monitor-collector.log 2>&1 &
```

## 更新日志：
+ v1.0.0：初始版本，支持日志采集、指标上报、本地存储； 
+ v1.1.0：新增 Kafka/HTTP 数据转发、Prometheus 指标导出； 
+ v1.2.0：优化日志编码器、支持配置热更新、完善 Docker 部署。

# 许可证
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/SketcOnly/agent-collector/blob/main/LICENSE)

本项目基于 MIT 许可证开源，允许自由使用、复制、修改、合并、发布、分发、 sublicense 和/或出售本软件的副本，前提是保留原版权声明和许可证文本。

完整许可证条款请查看：
- 项目根目录：[LICENSE](LICENSE)（本地查看）
- 在线查看：[MIT License](https://github.com/SketcOnly/agent-collector/blob/main/LICENSE)（GitHub 地址）