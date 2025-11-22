package collector

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

// MetricFactory 指标工厂，用于统一创建指标（counter/gauge/histogram）。
type MetricFactory struct {
	reg Registers
}

// NewMetricFactory 创建指标工厂
func NewMetricFactory(reg Registers) *MetricFactory {
	return &MetricFactory{reg: reg}
}

// CPU指标
func (m *MetricFactory) NewCPUUsageRatio() *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cpu_usage_ratio",
		Help: "CPU usage ratio per core",
	}, []string{"core"})
	m.reg.MustRegister(g)
	return g
}

func (m *MetricFactory) NewCPULoad1() prometheus.Gauge {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_load1",
		Help: "1 minute load average",
	})
	m.reg.MustRegister(g)
	return g
}

func (m *MetricFactory) NewCPULoad5() prometheus.Gauge {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_load5",
		Help: "5 minute load average",
	})
	m.reg.MustRegister(g)
	return g
}

func (m *MetricFactory) NewCPULoad15() prometheus.Gauge {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_load15",
		Help: "15 minute load average",
	})
	m.reg.MustRegister(g)
	return g
}

// Agent指标（采集器错误和耗时）
func (m *MetricFactory) NewAgentCollectErrorsTotal() *prometheus.CounterVec {
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "agent_collect_errors_total",
		Help: "Total collection errors",
	}, []string{"collector"})
	m.reg.MustRegister(c)
	return c
}

func (m *MetricFactory) NewAgentCollectDurationSeconds() *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "agent_collect_duration_seconds",
		Help:    "Collection duration per collector",
		Buckets: prometheus.DefBuckets,
	}, []string{"collector"})
	m.reg.MustRegister(h)
	return h
}
