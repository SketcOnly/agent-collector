package registers

import (
	"fmt"
	"github.com/agent-collector/config"
	"github.com/agent-collector/internal/collector"
	"github.com/agent-collector/internal/metrics"
	"github.com/agent-collector/logger"
	"go.uber.org/zap"
)

// RegisterCollectors  采集器注册统一入口（扩展仅需修改此函数）核心：开关控制 + 标识选择）
func RegisterCollectors(agent collector.Agent, cfg *config.Config, metricFactory metrics.MetricFactory,
) (targetCollector collector.Collector, err error) {
	// 存储所有已启用+已注册的采集器（用于后续校验和日志）
	enabledCollectors := make([]string, 0)

	// 1. CPU采集器：仅当 EnableCPU 为 true 时注册（优先级最高）
	if cfg.Collector.EnableCPU {
		// 构建配置（从全局配置读取，与开关强绑定）
		cpuCfg := collector.CPUCollectorConfig{
			CollectPerCore: cfg.Collector.CPU.CPUCollectorConfig,
		}
		// 创建采集器实例（实现 Collector 接口）
		cpuCollector := collector.NewCPUCollector(cpuCfg, metricFactory)
		// 注册到 Agent
		agent.Register(cpuCollector)
		// 标记为核心标识采集器（优先级1）
		targetCollector = cpuCollector
		// 记录已启用的采集器名称
		enabledCollectors = append(enabledCollectors, cpuCollector.Name())
		logger.Info(
			"registered CPU collector",
			cpuCollector.Name(),
			zap.Bool("enabled", cfg.Collector.EnableCPU),
		)
	} else {
		logger.Debug(
			"CPU collector disabled by config",
			"",
			zap.Bool("EnableCPU", cfg.Collector.EnableCPU),
		)
	}
	// 4. 关键校验：是否有至少一个采集器被启用并注册
	if len(enabledCollectors) == 0 {
		return nil, fmt.Errorf("no collectors enabled (check EnableCPU/EnableMem/EnableDisk in config)")
	}

	// 日志输出所有已启用的采集器（便于排查配置）
	logger.Info(
		"all enabled collectors registered",
		targetCollector.Name(),
		zap.Strings("enabled_collectors", enabledCollectors)) // 最终选中的标识采集器

	return targetCollector, nil

}
