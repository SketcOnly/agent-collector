package registers

import (
	"context"
	"fmt"
	"github.com/agent-collector/pkg/collector"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Module struct {
	Enabled bool
	Name    string
	NewFunc func() Collector
}

// InitPromRegistry 返回值
// promReg	*prometheus.Registry	Prometheus 指标注册器，可用于 HTTP endpoint 暴露 metrics 或做单元测试
// agent	Agent	                采集器管理器，后台周期性调用已注册的采集器进行指标采集
// nil	    error	                初始化成功时返回 nil，如果初始化或注册失败则返回具体错误
func InitPromRegistry(ctx context.Context, enableProcess bool, cfg *config.Config) (*prometheus.Registry, Agent, error) {
	// 3. 初始化Prometheus指标注册器（禁用Go指标）
	promReg := prometheus.NewRegistry()
	// 仅注册进程指标（可选），不注册Go指标
	if enableProcess {
		promReg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	// 初始化工厂包装成自己的 Registry
	metricFactory := metrics.NewMetricFactory(metrics.NewPromRegistry(promReg))

	//	// 4. 初始化采集器Agent（依赖接口）
	agent := NewRegistry(cfg.Monitor.Interval)

	// 5. 注册采集器（统一入口，扩展仅需添加注册代码）
	registeredCollectors, err := RegisterCollectors(agent, cfg, *metricFactory)
	// 新增调试日志：打印所有采集器的启用状态
	logger.Debug("collector enable status",
		zap.Bool("proc_enable", cfg.Monitor.Collectors.Proc.Enable),
		zap.Bool("sys_enable", cfg.Monitor.Collectors.Sys.Enable),
		zap.Bool("cgroup_enable", cfg.Monitor.Collectors.Cgroup.Enable),
		zap.Bool("container_enable", cfg.Monitor.Collectors.Container.Enable),
	)
	if err != nil {
		logger.Error("failed to register collectors", zap.Error(err))
		return nil, nil, err
	}

	// 5. 调用Agent.Start（传入正确的Collector实例，无类型错误）
	agent.Start(ctx)

	logger.Debug("failed to register collectors", zap.String("name", registeredCollectors[0].Name()), zap.Int("first_collector", len(registeredCollectors)), zap.Duration("interval", cfg.Monitor.Interval))

	return promReg, agent, nil
}

// RegisterCollectors  采集器注册统一入口（扩展仅需修改此函数）核心：开关控制 + 标识选择）
// 循环注册
// 新增采集器只需在 modules 列表添加一条，不必写重复的 if/else。
// 日志结构化
// zap.String、zap.Strings 保证结构化日志规范。
// 返回所有已注册采集器
// 避免单一 targetCollector 覆盖问题。
// 可扩展性强
// 支持 /proc、/sys、Cgroup、Container，未来添加新的数据源只需要新增一条 module 配置即可。
func RegisterCollectors(agent Agent, cfg *config.Config, metricFactory metrics.MetricFactory) ([]Collector, error) {

	modu := []Module{
		{
			Enabled: cfg.Monitor.Collectors.Proc.Enable,
			Name:    "/proc",
			NewFunc: func() Collector {
				return collector.NewCPUCollector(&cfg.Monitor.Collectors, metricFactory)
			},
		},
		//{
		//	enabled: cfg.Monitor.Collectors.Sys.Enable,
		//	name:    "/sys",
		//	newFunc: func() Collector {
		//		return collector.NewSysCollector(cfg.Sys.IgnoreDisks, cfg.Sys.IgnoreNetworks, metricFactory)
		//	},
		//},
		//{
		//	enabled: cfg.Cgroup.Enable,
		//	name:    "cgroup",
		//	newFunc: func() collector2.Collector {
		//		return collector2.NewCgroupCollector(metricFactory)
		//	},
		//},
		//{
		//	enabled: cfg.Container.Enable,
		//	name:    "container",
		//	newFunc: func() collector2.Collector {
		//		return collector2.NewContainerCollector(metricFactory)
		//	},
		//},
	}

	var registered []Collector
	for _, m := range modu {
		if m.Enabled {
			c := m.NewFunc()
			agent.Register(c)
			registered = append(registered, c)
			logger.Debug("registered collector", zap.String("name", m.Name))
		} else {
			logger.Debug("collector disabled", zap.String("name", m.Name))
		}
	}
	if len(registered) == 0 {
		return nil, fmt.Errorf("no collectors enabled; check your CollectorConfig")
	}
	// 日志输出所有已启用的采集器（便于排查配置）
	var names []string
	for _, m := range registered {
		names = append(names, m.Name())
	}
	logger.Debug("all enabled collectors registered", zap.Strings("enabled_collectors", names))

	return registered, nil

}
