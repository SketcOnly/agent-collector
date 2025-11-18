package registers

import (
	"github.com/agent-collector/config"
	"github.com/agent-collector/internal/collector"
	"github.com/agent-collector/internal/metrics"
)

// registerCollectors 采集器注册统一入口（扩展仅需修改此函数）
func RegisterCollectors(
	agent collector.Agent,
	cfg *config.Config,
	metricFactory *metrics.MetricFactory,
) {
	// // 注册磁盘采集器
	// if cfg.Collector.EnableDisk {
	// 	diskCfg := collector.DiskCollectorConfig{IgnoreDisks: cfg.Collector.IgnoreDisks}
	// 	agent.Register(collector.NewDiskCollector(logger, diskCfg, metricFactory))
	// }
	//
	// // 注册网络采集器
	// if cfg.Collector.EnableNetwork {
	// 	netCfg := collector.NetCollectorConfig{IgnoreNetworks: cfg.Collector.IgnoreNetworks}
	// 	agent.Register(collector.NewNetCollector(logger, netCfg, metricFactory))
	// }

	// 注册CPU采集器
	if cfg.Collector.EnableCPU {
		cpuCfg := collector.CPUCollectorConfig{CollectPerCore: cfg.Collector.CPU.CPUCollectorConfig}
		agent.Register(collector.NewCPUCollector(cpuCfg, metricFactory))
	}

	// -------------------------- 扩展示例：新增内存采集器 --------------------------
	// if cfg.Collector.EnableMem {
	//     memCfg := collector.MemCollectorConfig{...}
	//     agent.Register(collector.NewMemCollector(logger, memCfg, metricFactory))
	// }
}
