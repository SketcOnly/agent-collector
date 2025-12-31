package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/agent-collector/cmd/server"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"github.com/agent-collector/pkg/registers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	cfgFile   string
	GlobalCfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "agent-collector",
	Short: "Production-grade system metrics collector (CPU/disk/network) with Prometheus",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		// 加载配置
		GlobalCfg, err = config.LoadConfigWithCli(cmd)
		if err != nil {
			// 如文件语法错误，权限不足，配置校验失败,必须报错退出
			fmt.Fprintf(os.Stderr, "Configuration loading failed：%v\n", err)
			fmt.Fprintf(os.Stderr, "Please check the syntax, permissions, or use - c to specify a valid path in the configuration file\n")
			os.Exit(1) // 退出避免后续 nil 指针 panic
		}
		if err := runServer(cmd.Context(), GlobalCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Service startup failed: %v\n", err)
			os.Exit(1)
		}

		return nil
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "configs/config.yaml", "configuration file path")
	// 注册分组 flag
	initServerFlags(rootCmd)
	initMonitorFlags(rootCmd)
	initLogFlags(rootCmd)
}

func runServer(ctx context.Context, cfg *config.Config) error {

	fmt.Println("===== 日志初始化前的 Log 配置 =====") // 先用 Warn 级别（确保能输出，不受当前日志级别影响）
	fmt.Printf("Log Level from GlobalCfg: %s\n", cfg.Log.Level)
	fmt.Printf("Log Format from GlobalCfg: %s\n", cfg.Log.Format)
	// 同时打印 Viper 中的 log 配置，交叉验证
	fmt.Printf("Log Level from Viper: %s\n", viper.GetString("log.level"))
	fmt.Printf("Log Format from Viper: %s\n", viper.GetString("log.format"))

	//初始化日志
	initLogger, err := logger.InitLogger(&cfg.Log)
	if err != nil {
		return fmt.Errorf("log initialization failed: %w", err)
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
	//return runServer(cmd.Context(), GlobalCfg)
	cfgJson, err := json.MarshalIndent(GlobalCfg, "", "  ")
	if err != nil {
		logger.Warn("Failed to marshal GlobalCfg to JSON: %v\", err")
	} else {
		logger.Info("==========完整配置==========")
		logger.Info(string(cfgJson))
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
