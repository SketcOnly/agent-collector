package registers

import (
	"context"
	collector2 "github.com/agent-collector/pkg/collector"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Registry 接口（隔离 Prometheus 实现）
type Registry interface {
	prometheus.Registerer                          // 嵌入 Prometheus 官方注册器接口
	Register(collector prometheus.Collector) error //自定义扩展方法
}

// promRegistry Prometheus 实现
type promRegistry struct {
	registry *prometheus.Registry
}

// NewPromRegistry 创建 Prometheus 指标注册器
func NewPromRegistry(registry *prometheus.Registry) Registry {
	return &promRegistry{registry: registry}
}

// MustRegister 实现 prometheus.Registerer
func (p *promRegistry) MustRegister(collectors ...prometheus.Collector) {
	for _, c := range collectors {
		_ = p.registry.Register(c)
	}
}

// Unregister 实现 prometheus.Registerer
func (p *promRegistry) Unregister(collector prometheus.Collector) bool {
	return p.registry.Unregister(collector)
}

// Register 实现自定义 Registry 接口
func (p *promRegistry) Register(collector prometheus.Collector) error {
	return p.registry.Register(collector)
}

// MetricFactory 指标工厂
type MetricFactory struct {
	reg Registry
}

// NewMetricFactory 创建指标工厂
func NewMetricFactory(reg Registry) *MetricFactory {
	return &MetricFactory{reg: reg}
}

func InitPromRegistry(ctx context.Context, enableProcess bool, cfg *config.Config) (*prometheus.Registry, collector2.Agent, error) {
	// 3. 初始化Prometheus指标注册器（禁用Go指标）
	promRegistry := prometheus.NewRegistry()
	// 仅注册进程指标（可选），不注册Go指标
	if enableProcess {
		promRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	// 初始化工厂
	metricFactory := NewMetricFactory(NewPromRegistry(promRegistry))

	//	// 4. 初始化采集器Agent（依赖接口）
	agent := collector2.NewRegistry(cfg.Collector.Interval)

	// 5. 注册采集器（统一入口，扩展仅需添加注册代码）
	targetCollector, err := RegisterCollectors(agent, cfg, *metricFactory)
	if err != nil {
		logger.Error("failed to register collectors", targetCollector.Name(), zap.Error(err))
		return nil, nil, err
	}

	// 5. 调用Agent.Start（传入正确的Collector实例，无类型错误）
	agent.Start(ctx)
	logger.Info(
		"collector monitor started",
		targetCollector.Name(), // 正常调用Name()方法
		zap.Duration("interval", cfg.Collector.Interval),
	)

	return promRegistry, agent, nil
}
