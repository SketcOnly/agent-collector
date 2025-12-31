package agent

import (
	"github.com/agent-collector/pkg/config"
	log "github.com/agent-collector/pkg/logger"
	"github.com/spf13/cobra"
)

var defaultCfg = config.NewDefaultConfig()

func initServerFlags(root *cobra.Command) {
	if root == nil {
		log.Panic("invalid cobra root command is: nil pointer")
	}
	flags := root.PersistentFlags()
	
	// 注册server相关Flag，默认值取自默认配置
	flags.String("server.addr", defaultCfg.Server.Addr, "HTTP listening address (HTTP监听地址)")
	flags.Duration("server.read-timeout", defaultCfg.Server.ReadTimeout, "Read timeout duration (读取超时时间)")
	flags.Duration("server.write-timeout", defaultCfg.Server.WriteTimeout, "Write timeout duration (写入超时时间)")
	flags.Duration("server.idle-timeout", defaultCfg.Server.IdleTimeout, "Idle connection timeout duration (空闲连接超时时间)")
}
