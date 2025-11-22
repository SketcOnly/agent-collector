package config

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var valid = validator.New()

// Config 全局配置结构体（聚合所有核心模块）
type Config struct {
	Server  ServerConfig  `yaml:"server" mapstructure:"server" comment:"HTTP服务配置"` // 简化yaml键名（原 server_config → server，更简洁）
	Monitor MonitorConfig `yaml:"monitor" mapstructure:"monitor" comment:"监控采集配置"` // 简化yaml键名（原 monitor_config → monitor）
	Log     ZapLogConfig  `yaml:"log" mapstructure:"log" comment:"日志配置"`           // 简化yaml键名（原 logs_config → log）
}

// ServerConfig HTTP服务配置（超时统一为time.Duration，支持"30s"解析）
type ServerConfig struct {
	Addr         string        `yaml:"addr" mapstructure:"addr" env:"HTTP_ADDR" validate:"required,hostname_port" comment:"HTTP监听地址（格式：ip:port）"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout" env:"HTTP_READ_TIMEOUT" validate:"required,gt=0" comment:"读取超时时间（如30s）"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout" env:"HTTP_WRITE_TIMEOUT" validate:"required,gt=0" comment:"写入超时时间（如30s）"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" validate:"required,gt=0" comment:"空闲连接超时时间（如60s）"`
}

// MonitorConfig 监控采集全局配置
type MonitorConfig struct {
	Interval   time.Duration   `yaml:"interval" mapstructure:"interval" env:"MONITOR_INTERVAL" validate:"required,gt=0" comment:"监控采集间隔（如10s）" default:"10s"`
	Collectors CollectorConfig `yaml:"collectors" mapstructure:"collectors" comment:"各类数据源采集器配置"` // 键名改为collectors（复数更合理）
}

// CollectorConfig 多数据源采集器配置（简化字段名，避免冗余）
type CollectorConfig struct {
	Proc      ProcDataSourceConfig   `yaml:"proc" mapstructure:"proc" comment:"Linux /proc 数据源（CPU/内存等）"`                               // 原 enable_proc_data_source → proc（语义更清晰）
	Sys       SysDataSourceConfig    `yaml:"sys" mapstructure:"sys" comment:"Linux /sys 数据源（磁盘/网络等）"`                                   // 原 enable_sys_data_source → sys
	Cgroup    CgroupDataSourceConfig `yaml:"cgroup" mapstructure:"cgroup" comment:"Cgroup v1/v2 数据源（容器资源限制）"`                           // 原 enable_cgroup_data_source → cgroup
	Container ContainerRuntimeConfig `yaml:"container_runtime" mapstructure:"container_runtime" comment:"容器运行时API（Docker/containerd等）"` // 简化结构体名
}

// ProcDataSourceConfig /proc 数据源配置（去掉冗余Enable前缀）
type ProcDataSourceConfig struct {
	Enable         bool `yaml:"enable" mapstructure:"enable" env:"COLLECTOR_PROC_ENABLE" comment:"是否启用/proc数据源" default:"false"`
	CollectPerCore bool `yaml:"collect_per_core" mapstructure:"collect_per_core" env:"COLLECTOR_PROC_PER_CORE" comment:"是否按每核心采集CPU指标" default:"false"`
}

// SysDataSourceConfig /sys 数据源配置（修复env标签冲突）
type SysDataSourceConfig struct {
	Enable         bool     `yaml:"enable" mapstructure:"enable" env:"COLLECTOR_SYS_ENABLE" comment:"是否启用/sys数据源" default:"false"`
	IgnoreDisks    []string `yaml:"ignore_disks" mapstructure:"ignore_disks" env:"COLLECTOR_SYS_IGNORE_DISKS" comment:"忽略的磁盘列表（如/dev/sda）" default:"[]"`
	IgnoreNetworks []string `yaml:"ignore_networks" mapstructure:"ignore_networks" env:"COLLECTOR_SYS_IGNORE_NETWORKS" comment:"忽略的网络接口列表（如eth0）" default:"[]"` // 修复yaml标签（原ignore_network → ignore_networks，复数一致）
}

// CgroupDataSourceConfig Cgroup 数据源配置
type CgroupDataSourceConfig struct {
	Enable bool `yaml:"enable" mapstructure:"enable" env:"COLLECTOR_CGROUP_ENABLE" comment:"是否启用Cgroup数据源" default:"false"`
}

// ContainerRuntimeConfig 容器运行时API配置（简化结构体名）
type ContainerRuntimeConfig struct {
	Enable bool `yaml:"enable" mapstructure:"enable" env:"COLLECTOR_CONTAINER_ENABLE" comment:"是否启用容器运行时API" default:"false"`
}

// ZapLogConfig 日志配置（修复标签笔误、补充默认值）
type ZapLogConfig struct {
	Level     string `yaml:"level" mapstructure:"level" env:"LOG_LEVEL" validate:"required,oneof=debug info warn error dpanic panic fatal" comment:"日志级别" default:"info"`
	Format    string `yaml:"format" mapstructure:"format" env:"LOG_FORMAT" validate:"required,oneof=json console" comment:"日志格式（json/console）" default:"json"`
	Path      string `yaml:"path" mapstructure:"path" env:"LOG_PATH" validate:"required" comment:"日志存储路径" default:"./logs"`
	MaxSize   int    `yaml:"max_size" mapstructure:"max_size" env:"LOG_MAX_SIZE" validate:"required,gt=0" comment:"单个日志文件最大大小（MB）" default:"100"`
	MaxBackup int    `yaml:"max_backup" mapstructure:"max_backup" env:"LOG_MAX_BACKUP" validate:"required,gte=0" comment:"日志文件最大备份数" default:"30"` // 修复原max_size标签错误
	MaxAge    int    `yaml:"max_age" mapstructure:"max_age" env:"LOG_MAX_AGE" validate:"required,gte=0" comment:"日志文件最大保存天数" default:"7"`
	Compress  bool   `yaml:"compress" mapstructure:"compress" env:"LOG_COMPRESS" comment:"是否压缩过期日志" default:"true"`
}

// NewDefaultConfig 创建默认配置（所有字段兜底，避免空指针/非法值）
func NewDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Addr:         "0.0.0.0:8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Monitor: MonitorConfig{
			Interval: 10 * time.Second,
			Collectors: CollectorConfig{
				Proc: ProcDataSourceConfig{
					Enable:         false,
					CollectPerCore: true,
				},
				Sys: SysDataSourceConfig{
					Enable:         false,
					IgnoreDisks:    []string{},
					IgnoreNetworks: []string{},
				},
				Cgroup: CgroupDataSourceConfig{
					Enable: false,
				},
				Container: ContainerRuntimeConfig{
					Enable: false,
				},
			},
		},
		Log: ZapLogConfig{
			Level:     "info",
			Format:    "json",
			Path:      "./logs",
			MaxSize:   100,
			MaxBackup: 30,
			MaxAge:    7,
			Compress:  true,
		},
	}
}

// LoadConfigWithCli 支持 time.Duration，(Flags + YAML + ENV)
func LoadConfigWithCli(cmd *cobra.Command) (*Config, error) {
	cfg := NewDefaultConfig()
	v := viper.New()

	// 1. 绑定 Cobra Flags → Viper
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return nil, fmt.Errorf("bind flags: %w", err)
	}

	// 2. 解析配置文件 (--config)
	configFile, _ := cmd.Flags().GetString("config")
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config file %s: %w", configFile, err)
		}
	}

	// 3. 绑定环境变量 ENV -> Viper （HTTP_ADDR -> http.addr）
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("_", "."))

	// 4. 解码反序列化到结构体（支持 time.Duration）
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           cfg,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("new decoder: %w", err)
	}

	if err := decoder.Decode(v.AllSettings()); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	// 5. 校验配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// Validate 配置校验
func (c *Config) Validate() error {
	err := valid.Struct(c)
	if err != nil {
		return err
	}
	// 	1,校验Server服务配置
	if err := c.Server.Validate(); err != nil {
		return err
	}
	// 	2，校验采集配置
	if err := c.Monitor.Validate(); err != nil {
		return err
	}

	// 	3，校验日志配置
	if err := c.Log.Validate(); err != nil {
		return err
	}
	return nil
}
