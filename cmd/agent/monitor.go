package agent

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initMonitorFlags(root *cobra.Command) {
	f := root.PersistentFlags()

	f.Duration("monitor.interval", defaultCfg.Monitor.Interval, "采集间隔")

	f.Bool("collectors.proc.enable", defaultCfg.Monitor.Collectors.Proc.Enable, "启用 /proc")
	f.Bool("collectors.sys.enable", defaultCfg.Monitor.Collectors.Sys.Enable, "启用 /sys")

	f.StringSlice("collectors.sys.ignore-disks", defaultCfg.Monitor.Collectors.Sys.IgnoreDisks, "忽略磁盘")
	f.StringSlice("collectors.sys.ignore-networks", defaultCfg.Monitor.Collectors.Sys.IgnoreNetworks, "忽略网络网卡")

	f.Bool("collectors.cgroup.enable", defaultCfg.Monitor.Collectors.Cgroup.Enable, "启用 Cgroup")
	f.Bool("collectors.container-runtime.enable", defaultCfg.Monitor.Collectors.Container.Enable, "启用容器运行时 API")

	err := viper.BindPFlags(f)
	if err != nil {
		return
	}
}
