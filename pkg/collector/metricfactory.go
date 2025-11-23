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

// NewCPUUsageRatio CPU指标
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

// NewCPUUsagePercent 创建并注册 CPU 总体使用率指标
// 这个指标通常是一个 Gauge，因为它表示的是一个瞬时值
func (m *MetricFactory) NewCPUUsagePercent() *prometheus.GaugeVec {
	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cpu_usage_percent",
		Help: "Total CPU usage percentage",
	},
		[]string{"cpu", "mode"},
	)
	m.reg.MustRegister(gv)
	return gv
}

// NewCPUUsageModePercent 创建并注册按模式（user, system, idle等）划分的 CPU 使用率指标
// 这是一个 GaugeVec，带有一个 "mode" 标签
func (m *MetricFactory) NewCPUUsageModePercent() *prometheus.GaugeVec {
	gv := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cpu_usage_mode_percent",
			Help: "CPU usage percentage by mode (user, system, idle, iowait, etc.)",
		},
		[]string{"cpu", "mode"}, // 标签名，用于区分不同的 CPU 模式
	)
	m.reg.MustRegister(gv)
	return gv
}

// NewCPUInfo 创建并注册 CPU 信息指标
// 这是一个 GaugeVec，带有 "cpu", "model", "cores" 等标签，值通常为 1，用于暴露元信息
func (m *MetricFactory) NewCPUInfo() *prometheus.GaugeVec {
	gv := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cpu_info",
			Help: "CPU information (model, cores, etc.)",
		},
		[]string{"cpu", "model", "cores"}, // 标签名，用于携带 CPU 信息
	)
	m.reg.MustRegister(gv)
	return gv
}

// NewAgentCollectErrorsTotal Agent指标（采集器错误和耗时）
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
