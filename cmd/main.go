package main

import (
	"github.com/agent-collector/cmd/agent"
)

func main() {
	//app := &cli.App{
	//	Name:  "metrics-collector",
	//	Usage: "Production-grade system metrics collector (CPU/disk/network) with Prometheus",
	//	Flags: []cli.Flag{
	//		&cli.StringFlag{
	//			Name:  "config",
	//			Value: "config.yaml",
	//			Usage: "path to config file (e.g. --config=./conf/config.yaml)",
	//		},
	//	},
	//	Action: metrics.Run,
	//}

	agent.Execute()
	//	初始化默认日志(用于启动阶段错误输出)
	//if err := server.InitDefaultLogger(); err != nil {
	//	_, err := fmt.Fprintf(os.Stderr, "init default logger error: %v\n", err)
	//	if err != nil {
	//		return
	//	}
	//	os.Exit(1)
	//}
	//defer func(l *zap.Logger) {
	//	err := l.Sync()
	//	if err != nil {
	//		return
	//	}
	//}(zap.L())

}

//// run 主启动逻辑（依赖接口，无具体实现耦合）
//func run(c *cli.Context) error {
//	// 1. 加载配置
//	cfgPath := c.String("config.yaml")
//	cfg, err := config.Load(cfgPath)
//	if err != nil {
//		return fmt.Errorf("load config failed: %w", err)
//	}
//
//	// 2. 初始化日志
//	zapLogger, err := logger.NewZapLogger(config.ZapLogConfig{
//		Level:  cfg.Log.Level,
//		Format: cfg.Log.Format,
//		Path:   cfg.Log.Path,
//	})
//	if err != nil {
//		return fmt.Errorf("init logger failed: %w", err)
//	}
//	defer zapLogger.Sync()
//	log := zapLogger.Sugar()
//
//	// 3. 初始化Prometheus指标注册器（禁用Go指标）
//	promRegistry := prometheus.NewRegistry()
//	// 仅注册进程指标（可选），不注册Go指标
//	// promRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
//	metricFactory := metrics.NewMetricFactory(metrics.NewPromRegistry(promRegistry))
//

//
//
//
//	log.Infof("metrics started successfully (HTTP: %s, collect interval: %v)", cfg.HTTP.Addr, cfg.Collector.Interval)
//
//	// 8. 监听退出信号（优雅关闭）
//	signal.WaitForShutdown(zapLogger, func() error {
//		// 关闭顺序：HTTP服务 → 采集器
//		var err1 error
//		if httpServer != nil {
//			err1 = httpServer.Shutdown()
//		}
//
//		err2 := metrics.Shutdown(context.Background())
//
//		if err1 != nil || err2 != nil {
//			return fmt.Errorf("shutdown errors: http=%v, metrics=%v", err1, err2)
//		}
//		return nil
//	})
//
//	return nil
//}
