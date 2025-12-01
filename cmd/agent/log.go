package agent

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initLogFlags(root *cobra.Command) {
	f := root.PersistentFlags()
	logPrefix := "log."

	f.String(
		logPrefix+"level",
		defaultCfg.Log.Level,
		"-> Log level [info,debug] | 日志级别 [info,debug]")
	f.String(
		logPrefix+"format",
		defaultCfg.Log.Format,
		"-> Log format [console,json] | 日志格式 [console,json]")
	f.String(
		logPrefix+"path",
		defaultCfg.Log.Path,
		"-> Log file storage path | 日志路径")
	f.Int(
		logPrefix+"max-size",
		defaultCfg.Log.MaxSize,
		"-> Max size of single log file (MB) | 单文件最大MB")
	f.Int(
		logPrefix+"max-backup",
		defaultCfg.Log.MaxBackup,
		"-> Number of log backup files | 备份数量")
	f.Int(
		logPrefix+"max-age",
		defaultCfg.Log.MaxAge,
		"-> Maximum retention days of log files | 保存天数")
	f.Bool(
		logPrefix+"compress",
		defaultCfg.Log.Compress,
		"-> Whether to compress expired log files | 是否压缩")

	err := viper.BindPFlags(f)
	if err != nil {
		return
	}
}
