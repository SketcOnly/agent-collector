package monitor

import "github.com/prometheus/client_golang/prometheus"

// -------------------------- 磁盘采集器指标结构体 --------------------------
type DiskCollectorMetrics struct {
	UsageRatio *prometheus.GaugeVec // 磁盘使用率（0-1）
	UsedBytes  *prometheus.GaugeVec // 已用空间（字节）
	FreeBytes  *prometheus.GaugeVec // 空闲空间（字节）
}

// -------------------------- 网络采集器指标结构体 --------------------------
type NetCollectorMetrics struct {
	TransmitBytes  *prometheus.CounterVec // 发送字节数（累计）
	ReceiveBytes   *prometheus.CounterVec // 接收字节数（累计）
	TransmitErrors *prometheus.CounterVec // 发送错误数（累计）
	ReceiveErrors  *prometheus.CounterVec // 接收错误数（累计）
}

// -------------------------- CPU采集器指标结构体 --------------------------
type CPUCollectorMetrics struct {
	UsageRatio       *prometheus.GaugeVec
	Load1            prometheus.Gauge
	Load5            prometheus.Gauge
	Load15           prometheus.Gauge
	UsagePercent     *prometheus.GaugeVec
	UsageModePercent *prometheus.GaugeVec
	CPUInfo          *prometheus.GaugeVec
}
