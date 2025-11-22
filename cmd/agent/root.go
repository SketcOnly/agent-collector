package agent

import (
	"context"
	"fmt"
	"github.com/agent-collector/cmd/server"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/registers"
	"github.com/spf13/cobra"
	"os"
)

var (
	cfgFile   string
	GlobalCfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "monitor-collector",
	Short: "Production-grade system metrics collector (CPU/disk/network) with Prometheus",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		GlobalCfg, err = config.LoadConfigWithCli(cmd)
		if err != nil {
			// 统一输出错误到 stderr，不返回给 cobra
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "请检查配置文件路径或使用 -c 参数指定\n")
			os.Exit(1)
		}
		if err := runServer(cmd.Context(), GlobalCfg); err != nil {
			fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
			os.Exit(1)
		}
		//return runServer(cmd.Context(), GlobalCfg)
		return nil
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "configs/config.yaml", "配置文件路径")
	// 注册分组 flag
	initServerFlags(rootCmd)
	initMonitorFlags(rootCmd)
	initLogFlags(rootCmd)
}

func runServer(ctx context.Context, cfg *config.Config) error {

	//初始化配置校验
	//if err := cfg.Validate(); err != nil {
	//	return fmt.Errorf("配置校验失败: %w", err)
	//}

	//初始化日志
	initLogger, err := logger.InitLogger(&cfg.Log)
	if err != nil {
		return fmt.Errorf("日志初始化失败: %w", err)
	}

	// 修正：调用包级 Sync() 函数（不是实例方法），程序退出时刷盘
	defer logger.Sync()

	const enableProcess = true // 直接写死
	// init Registry
	registry, _, _ := registers.InitPromRegistry(context.Background(), enableProcess, cfg)
	httpServer := server.NewHTTPServer(cfg, initLogger, registry)
	if err := httpServer.Start(); err != nil {
		return fmt.Errorf("start HTTP server failed: %w", err)
	}

	server.WaitForShutdown(func() error {
		//withTimeout, cancelFunc := context.WithTimeout(ctx, 5*time.Second)
		//defer cancelFunc()

		// 调用HTTP服务的Shutdown方法优雅关闭
		if err := httpServer.Shutdown(); err != nil {
			return fmt.Errorf("shutdown HTTP server failed: %w", err)
		}

		logger.Info("all services shutdown successfully")
		return nil

	})
	return nil
}
