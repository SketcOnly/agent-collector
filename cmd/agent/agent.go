package agent

import (
	"fmt"
	"github.com/agent-collector/config"
	"github.com/agent-collector/internal/registers"
	"github.com/agent-collector/internal/server"
	"github.com/agent-collector/pkg"
	"github.com/agent-collector/pkg/goid"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/signal"
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
	projectName := "agent-collector"
	bannerColor := "ColorBlue" // 整体统一颜色
	pkg.PrintBanner(projectName, bannerColor)

	// 3. 设置全局默认 collector 为 complet（主程序相关日志自动使用）
	logger.SetDefaultCollector("complete")
	logger.Info("", fmt.Sprintf("%d", goid.GetGID()),
		"Log initialization successful ", zap.String("path", cfg.Log.Path),
		zap.String("level", cfg.Log.Level), zap.String("format", cfg.Log.Format))
	logger.Debug("", fmt.Sprintf("%d", goid.GetGID()),
		"Configuration initialization successful", zap.String("path", c.String("config")))
	// 获取全局实例zap.Logger
	//getLogger := logger.GetLogger()

	// 注册
	reaistry := registers.InitPromeReaistry(false, cfg)
	// 7. 启动采集器
	//server.Start(context.Background())
	// 6. 初始化HTTP服务（注入自定义注册器）
	httpServer := server.NewHTTPServer(cfg.HTTP.Addr, reaistry)
	if err := httpServer.Start(); err != nil {
		return fmt.Errorf("start HTTP server failed: %w", err)
	}

	// 5. 阻塞主goroutine（关键！不退出的核心）
	// 监听退出信号，收到信号后执行优雅关闭
	signal.WaitForShutdown(logger.GetLogger(), func() error {
		// 调用HTTP服务的Shutdown方法优雅关闭
		return httpServer.Shutdown()
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
	defer logger.Sync()
	return cfg, nil
}
