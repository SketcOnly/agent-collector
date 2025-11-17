package metrics

import "github.com/prometheus/client_golang/prometheus"

// ----------------------
// Registry 接口（隔离 Prometheus 实现）
// ----------------------
type Registry interface {
	prometheus.Registerer
	Register(collector prometheus.Collector) error
}

// ----------------------
// promRegistry Prometheus 实现
// ----------------------
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

// ----------------------
// MetricFactory 指标工厂
// ----------------------
type MetricFactory struct {
	reg Registry
}

// NewMetricFactory 创建指标工厂
func NewMetricFactory(reg Registry) *MetricFactory {
	return &MetricFactory{reg: reg}
}
