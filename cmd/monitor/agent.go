package monitor

import (
	"context"
	"fmt"
	"github.com/agent-collector/cmd/server"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/registers"
	"github.com/agent-collector/pkg/util"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"os"
)

func Run(c *cli.Context) error {
	//	1，初始化配置
	cfg, err := LoadConfig(c)
	if err != nil {
		return err
	}
	
	// 2，初始化banner
	projectName := "monitor-collector"
	bannerColor := "ColorBlue" // 整体统一颜色
	// PrintBanner(projectName, bannerColor)
	util.PrintBanner(projectName, bannerColor)
	
	// 3. 设置全局默认 collector 为 complet（主程序相关日志自动使用）
	logger.SetDefaultCollector("complete")
	collector := logger.GetDefaultCollector()
	
	// 4, 设置当前goid
	//logger.SetDefaultGid(strconv.FormatUint(goid.GetGID(), 10))
	//goroutineid := logger.GetDefaultGid()
	
	logger.Info("log initialization successful ", collector, zap.String("path", cfg.Log.Path), zap.String("level", cfg.Log.Level),
		zap.String("format", cfg.Log.Format))
	logger.Debug("configuration initialization successful", collector, zap.String("path", c.String("config")))
	
	// 初始化 Prometheus 注册器
	registry, _, _ := registers.InitPromRegistry(context.Background(), cfg.Collector.EnableCPU, cfg)
	
	// 7. 启动采集器
	//server.Start(context.Background())
	
	// 6. 初始化 HTTP 服务（注入配置、collector、注册器）
	httpServer := server.NewHTTPServer(cfg.HTTP.Addr, registry)
	if err := httpServer.Start(); err != nil {
		return fmt.Errorf("start HTTP server failed: %w", err)
	}
	
	server.WaitForShutdown(collector, func() error {
		logger.Info("starting graceful shutdown...", "")
		// 调用HTTP服务的Shutdown方法优雅关闭
		if err := httpServer.Shutdown(); err != nil {
			return fmt.Errorf("shutdown HTTP server failed: %w", err)
		}
		
		// 步骤2：关闭采集器（若启动了，补充关闭逻辑）
		// if err := collectorServer.Shutdown(context.Background()); err != nil {
		//     logger.Error(collector, goroutineID, "collector server shutdown failed", zap.Error(err))
		//     // 采集器关闭失败不阻断整体退出，仅警告
		// }
		// 步骤3：同步日志（确保日志刷盘）
		if err := logger.Sync(); err != nil {
			logger.Warn("logger sync failed", "", zap.Error(err))
		}
		logger.Info("all services shutdown successfully", "")
		return nil
		
	})
	return nil
}

// InitDefaultLogger initDefaultLogger 初始化默认日志（用于启动阶段的错误输出，避免日志未初始化时无法打印）
func InitDefaultLogger() error {
	defaultLog, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(defaultLog)
	return nil
}

func LoadConfig(c *cli.Context) (*config.Config, error) {
	s := c.String("config")
	
	// 验证配置文件是否存在
	if _, err := os.Stat(s); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file %s does not exist", s)
	}
	//加载配置
	cfg, err := config.Load(s)
	if err != nil {
		return nil, fmt.Errorf("load config failed: %w", err)
	}
	//初始化全局一次日志
	if err := logger.Init(cfg.Log); err != nil {
		return nil, fmt.Errorf("init logger failed: %w", err)
	}
	defer func() {
		err := logger.Sync()
		if err != nil {
		}
	}()
	return cfg, nil
}
