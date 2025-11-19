package registers

import (
	"context"
	"github.com/agent-collector/config"
	"github.com/agent-collector/internal/collector"
	"github.com/agent-collector/internal/metrics"
	"github.com/agent-collector/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func InitPromRegistry(ctx context.Context, enableProcess bool, cfg *config.Config) (*prometheus.Registry, collector.Agent, error) {
	// 3. 初始化Prometheus指标注册器（禁用Go指标）
	promRegistry := prometheus.NewRegistry()
	// 仅注册进程指标（可选），不注册Go指标
	if enableProcess {
		promRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	// 初始化工厂
	metricFactory := metrics.NewMetricFactory(metrics.NewPromRegistry(promRegistry))

	//	// 4. 初始化采集器Agent（依赖接口）
	agent := collector.NewRegistry(cfg.Collector.Interval)

	// 5. 注册采集器（统一入口，扩展仅需添加注册代码）
	targetCollector, err := RegisterCollectors(agent, cfg, *metricFactory)
	if err != nil {
		logger.Error("failed to register collectors", targetCollector.Name(), zap.Error(err))
		return nil, nil, err
	}

	// 5. 调用Agent.Start（传入正确的Collector实例，无类型错误）
	agent.Start(ctx)
	logger.Info(
		"collector agent started",
		targetCollector.Name(), // 正常调用Name()方法
		zap.Duration("interval", cfg.Collector.Interval),
	)

	return promRegistry, agent, nil
}
