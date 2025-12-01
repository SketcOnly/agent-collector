package metrics

import "github.com/prometheus/client_golang/prometheus"

// NewAgentCollectErrorsTotal 创建「采集器错误总数」指标
// 指标类型：Counter（计数器）- 仅支持单调递增，服务重启后会重置为0
// 核心作用：统计各采集器在运行过程中发生的采集错误累计次数
// 标签说明：
// collector: 采集器名称（如 "log_collector" 日志采集器、"metric_collector" 指标采集器），用于区分不同采集模块
func (m *MetricFactory) NewAgentCollectErrorsTotal() *prometheus.CounterVec {
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "agent_collect_errors_total",
		Help: "Total collection errors",
	}, []string{"collector"})
	m.reg.MustRegister(c)
	return c
}

// NewAgentCollectDurationSeconds 创建「采集器采集耗时分布」指标
// 指标类型：Histogram（直方图）- 记录数值分布，支持计算分位数、平均值等统计量
// 核心作用：记录每个采集器每次采集操作的耗时（单位：秒），反映采集性能
// 标签说明：
//
//	collector: 采集器名称，用于区分不同采集模块的耗时表现
//
// 分桶说明：使用Prometheus默认分桶 [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10] 秒
//
//	覆盖从毫秒级到秒级的常见采集耗时场景
func (m *MetricFactory) NewAgentCollectDurationSeconds() *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "agent_collect_duration_seconds",
		Help:    "Collection duration per collector",
		Buckets: prometheus.DefBuckets,
	}, []string{"collector"})
	m.reg.MustRegister(h)
	return h
}
