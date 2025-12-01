package agent

import (
	"github.com/agent-collector/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var defaultCfg = config.NewDefaultConfig()

func initServerFlags(root *cobra.Command) {
	f := root.PersistentFlags()

	f.String("server.addr", defaultCfg.Server.Addr, "-> HTTP listening address (HTTP监听地址)")
	f.Duration("server.read-timeout", defaultCfg.Server.ReadTimeout, "-> Read timeout duration (读取超时时间)")
	f.Duration("server.write-timeout", defaultCfg.Server.WriteTimeout, "-> Write timeout duration (写入超时时间)")
	f.Duration("server.idle-timeout", defaultCfg.Server.IdleTimeout, "-> Idle connection timeout duration (空闲连接超时时间)")

	// 绑定到 viper
	err := viper.BindPFlags(f)
	if err != nil {
		return
	}
}
