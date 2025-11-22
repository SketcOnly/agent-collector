package collector

import (
	"context"
	"fmt"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/monitor"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	cload "github.com/shirou/gopsutil/v3/load"
	"go.uber.org/zap"
)

// CPUCollector CPU采集器（实现Collector接口）
type CPUCollector struct {
	name            string
	cfg             *config.CollectorConfig
	metrics         monitor.CPUCollectorMetrics
	collectErrors   *prometheus.CounterVec
	collectDuration *prometheus.HistogramVec
}

// NewCPUCollector 创建CPU采集器
func NewCPUCollector(cfg *config.CollectorConfig, metricFactory MetricFactory) *CPUCollector {
	return &CPUCollector{
		name: "cpu-collector",
		cfg:  cfg,
		metrics: monitor.CPUCollectorMetrics{
			UsageRatio: metricFactory.NewCPUUsageRatio(),
			Load1:      metricFactory.NewCPULoad1(),
			Load5:      metricFactory.NewCPULoad5(),
			Load15:     metricFactory.NewCPULoad15(),
		},
		collectErrors:   metricFactory.NewAgentCollectErrorsTotal(),
		collectDuration: metricFactory.NewAgentCollectDurationSeconds(),
	}
}

// Name 返回采集器名称
func (c *CPUCollector) Name() string { return c.name }

// Init 初始化数据以及检查项
func (c *CPUCollector) Init() error {
	// 预检查CPU可用性
	if _, err := cpu.Counts(false); err != nil {
		logger.Error("failed to get CPU counts", zap.Error(err))
		return err
	}
	return nil
}

//Collect 执行指标采集

func (c *CPUCollector) Collect(ctx context.Context) error {
	start := time.Now()
	defer func() {
		c.collectDuration.WithLabelValues(c.name).Observe(time.Since(start).Seconds())
	}()

	// 1. 采集CPU使用率
	usageList, err := cpu.Percent(0, c.cfg.Proc.CollectPerCore)
	if err != nil {
		c.collectErrors.WithLabelValues(c.name).Inc()
		return fmt.Errorf("get cpu usage failed: %w", err)
	}

	// 2. 更新使用率指标
	if c.cfg.Proc.CollectPerCore {
		for i, usage := range usageList {
			c.metrics.UsageRatio.WithLabelValues(fmt.Sprintf("cpu%d", i)).Set(usage / 100)
		}
	} else {
		c.metrics.UsageRatio.WithLabelValues("total").Set(usageList[0] / 100)
	}

	// 3. 采集CPU负载
	load, err := cload.Avg()
	if err != nil {
		logger.Warn("failed to get CPU load", zap.Error(err))
		c.collectErrors.WithLabelValues(c.name).Inc()
		return nil
	}
	c.metrics.Load1.Set(load.Load1)
	c.metrics.Load5.Set(load.Load5)
	c.metrics.Load15.Set(load.Load15)
	logger.Debug("collected CPU metrics", zap.Float64("load1", load.Load1))
	return nil
}

func (c *CPUCollector) Close() error {
	return nil
}
