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
	"runtime"
)

var (
	cfgFile   string
	GlobalCfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "agent-collector",
	Short: "Production-grade system metrics collector (CPU/disk/network) with Prometheus",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检测是否为root用户
		isRoot, err := checkRootUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		} else {
			if !isRoot {
				fmt.Fprintf(os.Stderr, "Error: The program needs to be executed with root privileges (please use sudo or switch to root before running) \n")
				os.Exit(1) // 退出程序，避免后续权限不足问题
			}
			// 是 root 用户，打印提示信息（可选）
			fmt.Printf("Tip: Currently executing programs with root user privileges \n")
		}
		var errConfig error
		// 加载配置
		GlobalCfg, errConfig = config.LoadConfigWithCli(cmd)
		if errConfig != nil {
			// 如文件语法错误，权限不足，配置校验失败,必须报错退出
			fmt.Fprintf(os.Stderr, "Configuration loading failed：%v\n", errConfig)
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

	//fmt.Println("===== 日志初始化前的 Log 配置 =====") // 先用 Warn 级别（确保能输出，不受当前日志级别影响）
	//fmt.Printf("Log Level from GlobalCfg: %s\n", cfg.Log.Level)
	//fmt.Printf("Log Format from GlobalCfg: %s\n", cfg.Log.Format)
	//// 同时打印 Viper 中的 log 配置，交叉验证
	//fmt.Printf("Log Level from Viper: %s\n", viper.GetString("log.level"))
	//fmt.Printf("Log Format from Viper: %s\n", viper.GetString("log.format"))

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
	//cfgJson, err := json.MarshalIndent(GlobalCfg, "", "  ")
	//if err != nil {
	//	logger.Warn("Failed to marshal GlobalCfg to JSON: %v\", err")
	//} else {
	//	logger.Info("Complete configuration")
	//	logger.Info(string(cfgJson))
	//}

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

// checkRootUser 检测当前执行程序的用户是否为root（UID=0）
// 返回值： isRoot（是否为root）,err (检测失败)
func checkRootUser() (bool, error) {
	//	非linux系统(Windows等)不支持euid检测，直接返回false和提示
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return false, fmt.Errorf("当前系统 %s 不支持 root 用户检测，跳过", runtime.GOOS)
	}
	// 获取当前进程有效id
	// root用户等euid为0，普通用户euid大于0
	euid := os.Geteuid()
	return euid == 0, nil

}
