package agent

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initMonitorFlags(root *cobra.Command) {
	f := root.PersistentFlags()

	f.Duration("metrics.interval", defaultCfg.Monitor.Interval, "-> Interval for metrics collection (采集间隔)")

	f.Bool("collectors.proc.enable", defaultCfg.Monitor.Collectors.Proc.Enable, "-> Enable /proc metrics collector (启用 /proc 采集器)")
	f.Bool("collectors.proc.collect_per_core", defaultCfg.Monitor.Collectors.Proc.CollectPerCore, "-> Enable per-core metrics collection for /proc (启用 /proc 每个核心的指标采集)")
	f.Duration("collectors.proc.load_sample_cycle", defaultCfg.Monitor.Collectors.Proc.LoadSampleCycle, "-> Cycle duration for load sampling in /proc collection ( /proc 采集中的负载采样周期)")

	f.Bool("collectors.sys.enable", defaultCfg.Monitor.Collectors.Sys.Enable, "-> Enable /sys metrics collector (启用 /sys 采集器)")
	f.StringSlice("collectors.sys.ignore-disks", defaultCfg.Monitor.Collectors.Sys.IgnoreDisks, "-> List of disk names to ignore in /sys collection ( /sys 采集中需要忽略的磁盘名称列表)")
	f.StringSlice("collectors.sys.ignore-networks", defaultCfg.Monitor.Collectors.Sys.IgnoreNetworks, "-> List of network interface names to ignore in /sys collection ( /sys 采集中需要忽略的网卡名称列表)")

	f.Bool("collectors.cgroup.enable", defaultCfg.Monitor.Collectors.Cgroup.Enable, "-> Enable cgroup metrics collector (启用 Cgroup 采集器)")
	f.Bool("collectors.container-runtime.enable", defaultCfg.Monitor.Collectors.Container.Enable, "-> Enable container runtime API collector (启用容器运行时 API 采集器)")

	err := viper.BindPFlags(f)
	if err != nil {
		return
	}
}
