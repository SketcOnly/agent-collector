package collector

import (
	"bufio"
	"context"
	"fmt"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/monitor"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	cload "github.com/shirou/gopsutil/v3/load"
	"go.uber.org/zap"
)

// CPUTimes 存储CPU各模式的累计时间
type CPUTimes struct {
	User    float64
	Nice    float64
	System  float64
	Idle    float64
	Iowait  float64
	Irq     float64
	Softirq float64
	Steal   float64
}

// CPUCollector CPU采集器（实现Collector接口）
type CPUCollector struct {
	name            string
	cfg             *config.CollectorConfig
	metrics         monitor.CPUCollectorMetrics
	collectErrors   *prometheus.CounterVec
	collectDuration *prometheus.HistogramVec

	cpuInfoInitialized bool                // 用来防止重复采集 CPU 静态信息，提升程序效率。
	lastCPUTimes       map[string]CPUTimes // 存储上一次的CPU时间，用于计算使用率

	cpuId         string              // 逻辑CPU序号（processor字段）
	modelName     string              // 型号名称
	coreID        string              // 物理核心ID（core id字段，ARM用）
	logicalCores  int64               // 逻辑核心数（processor最大序号+1）
	coreIDSet     map[string]struct{} // 去重存储coreID，统计物理核心数
	hasCPUCores   bool                // 是否存在cpu cores字段（x86架构）
	physicalCores int64               // 物理核心数
	finalCores    int64
}

// NewCPUCollector 创建CPU采集器
func NewCPUCollector(cfg *config.CollectorConfig, metricFactory MetricFactory) *CPUCollector {
	return &CPUCollector{
		name:          "cpu-collector",
		cfg:           cfg,
		lastCPUTimes:  make(map[string]CPUTimes),
		cpuId:         "",
		modelName:     "",
		coreID:        "0",
		coreIDSet:     make(map[string]struct{}),
		hasCPUCores:   false,
		physicalCores: 0,
		finalCores:    0,
		metrics: monitor.CPUCollectorMetrics{
			UsageRatio:       metricFactory.NewCPUUsageRatio(),
			Load1:            metricFactory.NewCPULoad1(),
			Load5:            metricFactory.NewCPULoad5(),
			Load15:           metricFactory.NewCPULoad15(),
			UsagePercent:     metricFactory.NewCPUUsagePercent(),
			UsageModePercent: metricFactory.NewCPUUsageModePercent(),
			CPUInfo:          metricFactory.NewCPUInfo(),
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

	logger.Debug("collect CPU info", zap.String("name", c.name))

	// 1. 采集CPU使用率 整体/每核
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
	// 更新CPU负载指标
	c.metrics.Load1.Set(load.Load1)
	c.metrics.Load5.Set(load.Load5)
	c.metrics.Load15.Set(load.Load15)
	logger.Debug("collected CPU metrics", zap.Float64("load1", load.Load1))
	logger.Debug("collected CPU metrics", zap.Float64("load5", load.Load5))
	logger.Debug("collected CPU metrics", zap.Float64("load15", load.Load15))

	// UsagePercent/UsageModePercent
	if err = c.collectCPUFromProc(); err != nil {
		logger.Error("failed to collect CPU info", zap.Error(err))
		c.collectErrors.WithLabelValues(c.name).Inc()
	}

	// CPUInfo
	if err := c.collectCPUInfoFromProc(); err != nil {
		logger.Error("failed to collect CPU info from proc", zap.Error(err))
		c.collectErrors.WithLabelValues(c.name).Inc()
	}
	return nil
}

// collectCPUFromProc 从/proc/stat读取CPU各模式时间（兼容ARM/x86）
// 动态适配字段数量差异，缺失字段默认0.0，确保跨架构正常计算使用率
func (c *CPUCollector) collectCPUFromProc() error {
	open, err := os.Open("/proc/stat")
	if err != nil {
		return fmt.Errorf("open /proc/stat: %w", err)
	}
	defer open.Close()
	scanner := bufio.NewScanner(open)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		// 至少需要 "cpu" + 4个基础时间字段(user/nice/system/idle），否则跳过
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		cpu_id := fields[0]
		if cpu_id == "cpu" {
			cpu_id = "total" // 总览行重命名为total，统一标签格式
		}

		// 初始化CPU时间结构体，缺失字段默认0.0
		times := CPUTimes{}
		// 动态解析字段：按顺序读取，存在则解析，不存在则保持默认0.0
		// 字段顺序（Linux标准）：user(1) → nice(2) → system(3) → idle(4) → iowait(5) → irq(6) → softirq(7) → steal(8)
		if len(fields) >= 2 {
			times.User, _ = strconv.ParseFloat(fields[1], 64)
		}
		if len(fields) >= 3 {
			times.Nice, _ = strconv.ParseFloat(fields[2], 64)
		}
		if len(fields) >= 4 {
			times.System, _ = strconv.ParseFloat(fields[3], 64)
		}
		if len(fields) >= 5 {
			times.Idle, _ = strconv.ParseFloat(fields[4], 64)
		}
		if len(fields) >= 6 {
			times.Iowait, _ = strconv.ParseFloat(fields[5], 64)
		}
		if len(fields) >= 7 {
			times.Irq, _ = strconv.ParseFloat(fields[6], 64)
		}
		if len(fields) >= 8 {
			times.Softirq, _ = strconv.ParseFloat(fields[7], 64)
		}
		if len(fields) >= 9 {
			times.Steal, _ = strconv.ParseFloat(fields[8], 64)
		}

		//  计算总时间
		total := times.User + times.Nice + times.System + times.Idle + times.Iowait + times.Irq + times.Softirq + times.Steal
		//  如果有上一次的记录，计算使用率
		lastTime, exists := c.lastCPUTimes[cpu_id]
		if !exists {
			// 首次采集：仅存储当前时间，不计算使用率（无历史数据对比）
			c.lastCPUTimes[cpu_id] = times
			logger.Debug("first collect CPU times (skip usage calc)", zap.String("cpu", cpu_id), zap.Any("times", times))
			continue
		}
		// 计算各模式时间差（当前 - 上次）
		deltaUser := times.User - lastTime.User
		deltaNice := times.Nice - lastTime.Nice
		deltaSystem := times.System - lastTime.System
		deltaIdle := times.Idle - lastTime.Idle
		deltaIowait := times.Iowait - lastTime.Iowait
		deltaIrq := times.Irq - lastTime.Irq
		deltaSoftirq := times.Softirq - lastTime.Softirq
		deltaSteal := times.Steal - lastTime.Steal
		deltaTotal := total - (lastTime.User + lastTime.Nice + lastTime.System + lastTime.Idle +
			lastTime.Iowait + lastTime.Irq + lastTime.Softirq + lastTime.Steal)

		// 避免除零（理论上deltaTotal不会为0，除非CPU完全未工作）
		if deltaTotal <= 0 {
			logger.Debug("CPU total time not changed (skip usage calc)", zap.String("cpu", cpu_id))
			c.lastCPUTimes[cpu_id] = times // 更新最新时间，避免下次仍用旧数据
			continue
		}

		// 1. 更新各模式使用率指标（兼容缺失字段：缺失模式的delta为0，使用率显示0%）
		modeMetrics := map[string]float64{
			"user":    deltaUser,
			"nice":    deltaNice,
			"system":  deltaSystem,
			"idle":    deltaIdle,
			"iowait":  deltaIowait,
			"irq":     deltaIrq,
			"softirq": deltaSoftirq,
			"steal":   deltaSteal,
		}

		for mode, delta := range modeMetrics {
			usagePercent := delta / deltaTotal * 100
			c.metrics.UsageModePercent.WithLabelValues(cpu_id, mode).Set(usagePercent)
		}
		// 2. 更新总使用率指标（100% - 空闲率）
		totalUsagePercent := (deltaTotal - deltaIdle) / deltaTotal * 100
		c.metrics.UsagePercent.WithLabelValues(cpu_id).Set(totalUsagePercent)

		// 调试日志：输出核心指标（仅保留关键信息，避免日志冗余）
		logger.Debug("collected CPU mode usage",
			zap.String("cpu", cpu_id),
			zap.Float64("user%", modeMetrics["user"]/deltaTotal*100),
			zap.Float64("system%", modeMetrics["system"]/deltaTotal*100),
			zap.Float64("idle%", modeMetrics["idle"]/deltaTotal*100),
			zap.Float64("total%", totalUsagePercent))

		// 保存当前时间，作为下次采集的历史基准
		c.lastCPUTimes[cpu_id] = times
	}
	return scanner.Err()
}

// 更新 CPUInfo (读取 /proc/cpuinfo)
// proc/cpuinfo 文件包含了每个 CPU 核心的详细信息
func (c *CPUCollector) collectCPUInfoFromProc() error {
	if c.cpuInfoInitialized {
		return nil
	}
	open, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return fmt.Errorf("open /proc/cpuinfo: %w", err)
	}
	defer open.Close()
	buf := bufio.NewScanner(open)

	for buf.Scan() {
		line := buf.Text()
		splitN := strings.SplitN(line, ":", 2)
		if len(splitN) != 2 {
			continue
		}
		key := strings.TrimSpace(splitN[0])
		value := strings.TrimSpace(splitN[1])
		switch key {
		case "model name":
			c.modelName = value
		case "cpu cores":
			// x86架构: 直接读取物理核心数
			c.physicalCores, _ = strconv.ParseInt(value, 10, 64)
			c.hasCPUCores = true
		case "cpu id":
			//	ARM架构：采集core id, 用于去重统计物理核心数
			c.coreID = value
			c.coreIDSet[c.coreID] = struct{}{} // 去重统计
		case "processor":
			// 更新逻辑CPU序号，同时统计逻辑核心数（取最大序号+1）
			c.cpuId = value
			currentLogical, _ := strconv.ParseInt(value, 10, 64)
			if currentLogical+1 > c.logicalCores {
				c.logicalCores = currentLogical + 1
			}
		}
	}
	// 确定最准的核心数(优先级：cpu cores > core id 去重数 > 逻辑核心数)
	if c.hasCPUCores {
		c.finalCores = c.physicalCores
	} else if len(c.coreIDSet) > 0 {
		c.finalCores = int64(len(c.coreIDSet)) // ARM架构：物理核心数=core id去重后的数量
	} else {
		c.finalCores = c.logicalCores // 兜底：逻辑核心数
	}
	//	保存每个逻辑cPU的信息(如果是多核,会循环采集多个processor)
	if c.cpuId != "" {
		c.metrics.CPUInfo.WithLabelValues(
			c.cpuId, c.modelName, strconv.FormatInt(c.finalCores, 10), // 最终核心数转字符串
		).Set(1)

		logger.Debug("collected CPU static info",
			zap.String("cpu_id", c.cpuId),
			zap.String("model_name", c.modelName),
			zap.Int64("physical_cores", c.finalCores),
			zap.Int64("logical_cores", c.logicalCores))
	}
	c.cpuInfoInitialized = true
	logger.Info("CPU static info collection completed",
		zap.Int64("physical_cores", c.finalCores),
		zap.Int64("logical_cores", c.logicalCores))

	return buf.Err()

}
func (c *CPUCollector) Close() error {
	return nil
}
