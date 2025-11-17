package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// -------------------------- 网络指标创建方法 --------------------------
func (f *MetricFactory) NewNetworkTransmitBytesTotal() *prometheus.CounterVec {
	return promauto.With(f.reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "system_network_transmit_bytes_total",
			Help: "Total bytes transmitted over the network interface",
		},
		[]string{"interface"},
	)
}

func (f *MetricFactory) NewNetworkReceiveBytesTotal() *prometheus.CounterVec {
	return promauto.With(f.reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "system_network_receive_bytes_total",
			Help: "Total bytes received over the network interface",
		},
		[]string{"interface"},
	)
}

func (f *MetricFactory) NewNetworkTransmitErrorsTotal() *prometheus.CounterVec {
	return promauto.With(f.reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "system_network_transmit_errors_total",
			Help: "Total transmit errors over the network interface",
		},
		[]string{"interface"},
	)
}

func (f *MetricFactory) NewNetworkReceiveErrorsTotal() *prometheus.CounterVec {
	return promauto.With(f.reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "system_network_receive_errors_total",
			Help: "Total receive errors over the network interface",
		},
		[]string{"interface"},
	)
}

// -------------------------- Agent自身监控指标 --------------------------
func (f *MetricFactory) NewAgentCollectDurationSeconds() *prometheus.HistogramVec {
	return promauto.With(f.reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agent_collect_duration_seconds",
			Help:    "Duration of collector execution",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 0.01s ~ 5.12s
		},
		[]string{"collector"},
	)
}

func (f *MetricFactory) NewAgentCollectErrorsTotal() *prometheus.CounterVec {
	return promauto.With(f.reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_collect_errors_total",
			Help: "Total number of collector errors",
		},
		[]string{"collector"},
	)
}
