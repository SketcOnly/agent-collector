package agent

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initLogFlags(root *cobra.Command) {
	f := root.PersistentFlags()

	f.String("log.level", defaultCfg.Log.Level, "日志级别")
	f.String("log.format", defaultCfg.Log.Format, "日志格式")
	f.String("log.path", defaultCfg.Log.Path, "日志路径")
	f.Int("log.max-size", defaultCfg.Log.MaxSize, "单文件最大MB")
	f.Int("log.max-backup", defaultCfg.Log.MaxBackup, "备份数量")
	f.Int("log.max-age", defaultCfg.Log.MaxAge, "保存天数")
	f.Bool("log.compress", defaultCfg.Log.Compress, "是否压缩")

	err := viper.BindPFlags(f)
	if err != nil {
		return
	}
}
