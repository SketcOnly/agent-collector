package collector

import "github.com/prometheus/client_golang/prometheus"

// -------------------------- 磁盘采集器指标结构体 --------------------------
type DiskCollectorMetrics struct {
	usageRatio *prometheus.GaugeVec // 磁盘使用率（0-1）
	usedBytes  *prometheus.GaugeVec // 已用空间（字节）
	freeBytes  *prometheus.GaugeVec // 空闲空间（字节）
}

// -------------------------- 网络采集器指标结构体 --------------------------
type NetCollectorMetrics struct {
	transmitBytes  *prometheus.CounterVec // 发送字节数（累计）
	receiveBytes   *prometheus.CounterVec // 接收字节数（累计）
	transmitErrors *prometheus.CounterVec // 发送错误数（累计）
	receiveErrors  *prometheus.CounterVec // 接收错误数（累计）
}

// -------------------------- CPU采集器指标结构体 --------------------------
type CPUCollectorMetrics struct {
	usageRatio *prometheus.GaugeVec // CPU使用率（0-1）
	load1      prometheus.Gauge     // 1分钟负载
	load5      prometheus.Gauge     // 5分钟负载
	load15     prometheus.Gauge     // 15分钟负载
}
