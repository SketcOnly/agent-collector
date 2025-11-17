package registers

import (
	"github.com/agent-collector/config"
	"github.com/agent-collector/internal/collector"
	"github.com/agent-collector/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func InitPromeReaistry(enableProcess bool, cfg *config.Config) *prometheus.Registry {
	// 3. 初始化Prometheus指标注册器（禁用Go指标）
	promRegistry := prometheus.NewRegistry()
	// 仅注册进程指标（可选），不注册Go指标
	if enableProcess {
		promRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	metricFactory := metrics.NewMetricFactory(metrics.NewPromRegistry(promRegistry))

	//	// 4. 初始化采集器Agent（依赖接口）
	var agent collector.Agent = collector.NewRegistry(cfg.Collector.Interval)

	// 5. 注册采集器（统一入口，扩展仅需添加注册代码）
	RegisterCollectors(agent, cfg, metricFactory)

	return promRegistry
}
