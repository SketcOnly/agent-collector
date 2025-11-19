package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// -------------------------- 磁盘指标创建方法 --------------------------
func (f *MetricFactory) NewDiskUsageRatio() *prometheus.GaugeVec {
	return promauto.With(f.reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_disk_usage_ratio",
			Help: "Disk usage ratio (used / total)",
		},
		[]string{"device", "mountpoint"},
	)
}

func (f *MetricFactory) NewDiskUsedBytes() *prometheus.GaugeVec {
	return promauto.With(f.reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_disk_used_bytes",
			Help: "Disk used space in bytes",
		},
		[]string{"device", "mountpoint"},
	)
}

func (f *MetricFactory) NewDiskFreeBytes() *prometheus.GaugeVec {
	return promauto.With(f.reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_disk_free_bytes",
			Help: "Disk free space in bytes",
		},
		[]string{"device", "mountpoint"},
	)
}
