package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

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
		[]string{"cpu"},
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
