package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// -------------------------- CPU指标创建方法 --------------------------
func (f *MetricFactory) NewCPUUsageRatio() *prometheus.GaugeVec {
	return promauto.With(f.reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_ratio",
			Help: "CPU usage ratio (0-1) per core or total",
		},
		[]string{"cpu"}, // 标签：cpu0/cpu1/.../total
	)
}

func (f *MetricFactory) NewCPULoad1() prometheus.Gauge {
	return promauto.With(f.reg).NewGauge(prometheus.GaugeOpts{
		Name: "system_cpu_load_1",
		Help: "CPU 1-minute load average",
	},
	)
}

func (f *MetricFactory) NewCPULoad5() prometheus.Gauge {
	return promauto.With(f.reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "system_cpu_load_5",
			Help: "CPU 5-minute load average",
		},
	)
}

func (f *MetricFactory) NewCPULoad15() prometheus.Gauge {
	return promauto.With(f.reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "system_cpu_load_15",
			Help: "CPU 15-minute load average",
		},
	)
}
