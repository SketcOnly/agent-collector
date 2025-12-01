package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Registers  接口通过接口隔离了 Prometheus 的默认实现，使后续可替换其它实现（例如自定义的组合器），适合单测 mock。 避免业务依赖 Prometheus 具体实现。
type Registers interface {
	prometheus.Registerer                          // 嵌入 Prometheus 官方注册器接口
	Register(collector prometheus.Collector) error //自定义扩展方法
}

// promRegistry Prometheus 实现，内部包裹了官方的 *prometheus.Registry
type promRegistry struct {
	registry *prometheus.Registry
}

// NewPromRegistry 创建 Prometheus 指标注册器
func NewPromRegistry(registry *prometheus.Registry) Registers {
	return &promRegistry{registry: registry}
}

// MustRegister 实现 prometheus.Registerer
func (p *promRegistry) MustRegister(collectors ...prometheus.Collector) {
	var err error
	for _, c := range collectors {
		if err = p.registry.Register(c); err != nil {
			panic(err)
		}
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
